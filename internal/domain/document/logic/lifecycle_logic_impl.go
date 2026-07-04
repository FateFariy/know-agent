package logic

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

type LifecycleLogicImpl struct {
	port          *adapter.DocumentPort
	repo          adapter.DocumentRepository
	chunkStrategy ChunkStrategyLogic
	parseTopic    string
	indexTopic    string
}

var _ LifecycleLogic = (*LifecycleLogicImpl)(nil)

func NewLifecycleLogicImpl(svcCtx *svc.ServiceContext, port *adapter.DocumentPort, repo adapter.DocumentRepository, chunkStrategy ChunkStrategyLogic) *LifecycleLogicImpl {
	return &LifecycleLogicImpl{
		port:          port,
		repo:          repo,
		chunkStrategy: chunkStrategy,
		parseTopic:    svcCtx.Config.MQ.ParseTopic,
		indexTopic:    svcCtx.Config.MQ.IndexTopic,
	}
}

// Upload 上传文档：完成文件上传、存储、文档记录创建及解析任务下发
func (d *LifecycleLogicImpl) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, document *entity.Document) (*vo.DocumentUpload, error) {
	// 校验文件类型是否支持
	fileType := vo.DetectFileType(header.Filename)
	if fileType == vo.FileTypeUnknown {
		return nil, errorx.ErrUnsupportedFileType.Format(fileType)
	}

	// 读取文件前512字节用于MIME类型检测
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	mimeType := http.DetectContentType(buf[:n])
	_, _ = file.Seek(0, io.SeekStart)

	// 读取完整文件内容
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, errorx.ErrEmptyFileContent.Format(err.Error())
	}

	// 生成全局唯一文档ID
	documentId := utils.GetSnowflakeNextID()

	// 上传原文件至MinIO存储
	storedObjectInfo, err := d.port.UploadOriginalFile(ctx, documentId, header.Filename, fileBytes, mimeType)
	if err != nil {
		return nil, err
	}

	// 填充文档实体字段
	document.ID = documentId
	document.DocumentName = utils.Ternary(strutil.IsNotBlank(document.DocumentName), strutil.Trim(document.DocumentName), header.Filename)
	document.OriginalFileName = header.Filename
	document.FileType = fileType
	document.MimeType = mimeType
	document.FileSize = int64(len(fileBytes))
	document.StorageType = vo.StorageTypeMINIO
	document.BucketName = storedObjectInfo.BucketName
	document.ObjectName = storedObjectInfo.ObjectName
	document.ObjectUrl = storedObjectInfo.ObjectUrl
	document.ParseStatus = vo.ParseStatusParsing
	document.StrategyStatus = vo.StrategyStatusWaitRecommend
	document.IndexStatus = vo.IndexStatusWaitBuild
	document.KnowledgeScopeCode = strutil.Trim(document.KnowledgeScopeCode)
	document.KnowledgeScopeName = strutil.Trim(document.KnowledgeScopeName)
	document.BusinessCategory = strutil.Trim(document.BusinessCategory)
	document.DocumentTags = strutil.Trim(document.DocumentTags)

	// 创建解析任务
	taskId := utils.GetSnowflakeNextID()
	task := &entity.DocumentTask{
		ID:            taskId,
		DocumentId:    documentId,
		TaskType:      vo.TaskTypeParseRoute,
		TaskStatus:    vo.TaskStatusNew,
		CurrentStage:  vo.TaskStageFileUpload,
		TriggerSource: utils.Ternary(document.OperatorId == 0, vo.TriggerSourceSystem, vo.TriggerSourceUser),
	}

	// 记录文件上传完成的任务日志
	detail, _ := json.Marshal(map[string]any{
		"originalFileName": header.Filename,
		"fileSize":         len(fileBytes),
	})

	taskLog := &entity.DocumentTaskLog{
		TaskId:       taskId,
		DocumentId:   documentId,
		StageType:    vo.TaskStageFileUpload,
		EventType:    vo.TaskEventComplete,
		LogLevel:     vo.LogLevelInfo,
		OperatorType: utils.Ternary(document.OperatorId == 0, vo.OperatorTypeSystem, vo.OperatorTypeUser),
		OperatorId:   document.OperatorId,
		Content:      "文件上传完成，已进入解析与策略推荐队列",
		DetailJson:   string(detail),
	}

	// 事务性插入文档、任务、任务日志
	insertTx := func(txCtx context.Context) error {
		if err = d.repo.InsertDocument(txCtx, document); err != nil {
			return err
		}
		if err = d.repo.InsertTask(txCtx, task); err != nil {
			return err
		}
		return d.repo.InsertTaskLog(txCtx, taskLog)
	}
	if err = d.repo.Do(ctx, insertTx); err != nil {
		return nil, err
	}

	// 发送解析消息至MQ，触发后续解析流程
	parseMessage := map[string]any{"documentId": documentId, "taskId": taskId}
	if err = d.port.Send(ctx, d.parseTopic, strconv.FormatInt(documentId, 10), parseMessage); err != nil {
		return nil, err
	}

	// 返回上传结果
	return &vo.DocumentUpload{
		DocumentId:     documentId,
		TaskId:         taskId,
		DocumentName:   document.DocumentName,
		ParseStatus:    document.ParseStatus,
		StrategyStatus: document.StrategyStatus,
		IndexStatus:    document.IndexStatus,
	}, nil
}

// QueryDocumentPage 分页查询文档列表（含最新任务信息）
func (d *LifecycleLogicImpl) QueryDocumentPage(ctx context.Context, pageNo, pageSize int, keyword string) ([]*entity.Document, int64, error) {
	// 分页查询文档基础列表
	documentList, total, err := d.repo.SelectDocumentPage(ctx, pageNo, pageSize, keyword)
	if err != nil || total == 0 {
		return nil, 0, err
	}

	// 提取所有文档ID，用于批量查询关联任务
	documentIds := slice.Map(documentList, func(index int, document *entity.Document) int64 {
		return document.ID
	})

	// 根据文档ID批量查询关联的任务列表
	taskList, err := d.repo.SelectTaskListByDocumentIds(ctx, documentIds)
	if err != nil {
		return nil, 0, err
	}

	// 构建文档ID到最新任务的映射（利用遍历顺序保证取第一个/最新任务）
	latestTaskMap := make(map[int64]*entity.DocumentTask)
	for _, task := range taskList {
		if _, exists := latestTaskMap[task.DocumentId]; !exists {
			latestTaskMap[task.DocumentId] = task
		}
	}

	// 为每个文档填充枚举名称和最新任务信息
	for i, document := range documentList {
		documentList[i].FillEnumNames()                                // 填充状态等枚举字段的中文名称
		documentList[i].FillLatestTaskInfo(latestTaskMap[document.ID]) // 填充最新任务信息
	}

	return documentList, total, nil
}

// QueryDocumentDetail 查询文档详情
func (d *LifecycleLogicImpl) QueryDocumentDetail(ctx context.Context, documentId int64) (*entity.Document, error) {
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return nil, err
	}

	task, err := d.repo.SelectLatestTask(ctx, documentId)
	if err != nil {
		return nil, err
	}
	document.FillEnumNames()
	document.FillLatestTaskInfo(task)

	return document, nil
}

// DeleteDocument 删除文档 todo 删除其他索引,实现关键词搜索、导航索引、知识路由索引、结构图投影
func (d *LifecycleLogicImpl) DeleteDocument(ctx context.Context, documentId int64) (string, error) {
	// 检查是否有活跃任务
	activeTaskCount, err := d.repo.CountActiveTask(ctx, documentId, 0, vo.TaskStatusNew, vo.TaskStatusRunning)
	if err != nil {
		return "", err
	}
	if activeTaskCount > 0 {
		return "", errorx.ErrDocumentStatusInvalid.Format("当前文档存在进行中的任务，请等待任务结束后再删除")
	}

	return d.repo.DeleteDocumentRelatedDataById(ctx, documentId)
}

// QueryStrategyPlan 查询策略方案
func (d *LifecycleLogicImpl) QueryStrategyPlan(ctx context.Context, documentId int64) (*entity.Document, *entity.DocumentStrategyPlan, error) {
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return nil, nil, err
	}

	var plan *entity.DocumentStrategyPlan

	if document.CurrentPlanId > 0 {
		plan, err = d.repo.SelectPlanById(ctx, document.CurrentPlanId)
		if err != nil {
			return nil, nil, err
		}

		stepList, err := d.repo.SelectStepListByPlanId(ctx, plan.ID)
		if err != nil {
			return nil, nil, err
		}
		plan.FillEnumNames()
		plan.FillAndProcessPipeline(stepList)
		document.PlanReady = true
		document.FillEnumNames()
	}

	return document, plan, nil
}

// ConfirmStrategy 确认策略
func (d *LifecycleLogicImpl) ConfirmStrategy(ctx context.Context, cmd *vo.DocumentStrategyConfirmCmd) (*entity.DocumentStrategyPlan, *entity.Document, error) {
	// 查询文档信息
	document, err := d.repo.SelectDocumentById(ctx, cmd.DocumentId)
	if err != nil {
		return nil, nil, err
	}

	// 状态校验：文档必须完成解析
	if document.ParseStatus != vo.ParseStatusParseSuccess {
		return nil, nil, common.NewBizError(errorx.ErrDocumentStatusInvalid.Code, "当前文档还未完成解析，不能确认策略。")
	}

	// 方案一致性校验：请求的方案需与文档当前方案一致
	if document.CurrentPlanId != cmd.BasePlanId {
		return nil, nil, common.NewBizError(errorx.ErrStrategyPlanNotFound.Code, "当前文档的基础方案不存在或已切换。")
	}

	// 查询基础方案信息
	basePlan, err := d.repo.SelectPlanById(ctx, cmd.BasePlanId)
	if err != nil {
		return nil, nil, err
	}

	// 查询基础方案的步骤列表
	baseStepList, err := d.repo.SelectStepListByPlanId(ctx, basePlan.ID)
	if err != nil {
		return nil, nil, err
	}

	// 提取用户提交的策略类型列表
	extractStrategyTypes := func(steps []*vo.DocumentStrategyStepItem) []int {
		slice.SortBy(steps, func(a, b *vo.DocumentStrategyStepItem) bool { return a.StepNo < b.StepNo })
		return slice.Map(steps, func(index int, item *vo.DocumentStrategyStepItem) int { return item.StrategyType })
	}
	parentTypeList := extractStrategyTypes(cmd.ParentSteps)
	childTypeList := extractStrategyTypes(cmd.ChildSteps)

	// 标准化策略步骤
	normalizedStepList, err := d.chunkStrategy.NormalizeSteps(ctx, baseStepList, parentTypeList, childTypeList, cmd.DocumentId)
	if err != nil {
		return nil, nil, err
	}

	// 校验策略步骤不能为空
	if len(normalizedStepList) == 0 {
		return nil, nil, common.NewBizError(errorx.ErrStrategyStepEmpty.Code, "策略步骤不能为空。")
	}

	// 提取标准化后的流水线类型列表
	normalizedParentTypeList := d.extractPipelineTypes(normalizedStepList, vo.PipelineTypeParent)
	normalizedChildTypeList := d.extractPipelineTypes(normalizedStepList, vo.PipelineTypeChild)
	if len(normalizedParentTypeList) == 0 {
		return nil, nil, common.NewBizError(errorx.ErrStrategyStepEmpty.Code, "父块流水线不能为空。")
	}
	if len(normalizedChildTypeList) == 0 {
		return nil, nil, common.NewBizError(errorx.ErrStrategyStepEmpty.Code, "子块流水线不能为空。")
	}

	// 提取基础方案的流水线类型列表
	baseParentTypeList := d.extractPipelineTypes(baseStepList, vo.PipelineTypeParent)
	baseChildTypeList := d.extractPipelineTypes(baseStepList, vo.PipelineTypeChild)

	// 判断是否发生了策略变更
	distinctParentTypeList := stream.FromSlice(parentTypeList).Distinct().ToSlice()
	distinctChildTypeList := stream.FromSlice(childTypeList).Distinct().ToSlice()
	changed := !slice.Equal(baseParentTypeList, normalizedParentTypeList) || !slice.Equal(baseChildTypeList, normalizedChildTypeList)

	// 查询最新解析任务信息
	latestParseTask, err := d.repo.SelectLatestTask(ctx, document.ID, vo.TaskTypeParseRoute)
	if err != nil {
		return nil, nil, err
	}

	var newPlan *entity.DocumentStrategyPlan
	confirmTx := func(txCtx context.Context) error {
		// 根据是否变更处理方案
		if changed {
			// 策略发生变更：废弃旧方案
			if err = d.repo.UpdatePlanById(txCtx, &entity.DocumentStrategyPlan{ID: basePlan.ID, PlanStatus: vo.PlanStatusDiscarded}); err != nil {
				return err
			}

			// 查询最新方案版本号
			latestPlanVersion, err := d.repo.SelectLatestPlanVersion(ctx, document.ID)
			if err != nil {
				return err
			}

			// 创建新方案
			newPlan = &entity.DocumentStrategyPlan{
				ID:              utils.GetSnowflakeNextID(),
				DocumentId:      document.ID,
				PlanVersion:     latestPlanVersion + 1,
				PlanSource:      vo.PlanSourceUserAdjust,
				PlanStatus:      vo.PlanStatusConfirmed,
				StrategyCount:   len(normalizedStepList),
				RecommendReason: basePlan.RecommendReason,
				AdjustNote:      cmd.AdjustNote,
				ConfirmUserId:   cmd.OperatorId,
				ConfirmTime:     time.Now(),
			}
			newPlan.FillAndProcessPipeline(normalizedStepList)
			newPlan.Normalized = !slice.Equal(distinctParentTypeList, normalizedParentTypeList) || !slice.Equal(distinctChildTypeList, normalizedChildTypeList)

			// 更新文档的当前方案ID为新方案
			document.CurrentPlanId = newPlan.ID

			// 创建新方案
			if err = d.repo.InsertPlan(txCtx, newPlan); err != nil {
				return err
			}

			// 为步骤分配ID和方案ID
			for _, step := range normalizedStepList {
				step.ID = utils.GetSnowflakeNextID()
				step.PlanId = newPlan.ID
			}

			// 插入新方案的步骤
			if err = d.repo.InsertStepBatch(txCtx, normalizedStepList); err != nil {
				return err
			}

			// 构建调整日志
			if latestParseTask != nil {
				detailJson, _ := json.Marshal(map[string]any{
					"parentStrategyTypes": normalizedParentTypeList,
					"childStrategyTypes":  normalizedChildTypeList,
					"adjustNote":          cmd.AdjustNote,
				})
				adjustLog := &entity.DocumentTaskLog{
					ID:           utils.GetSnowflakeNextID(),
					TaskId:       latestParseTask.ID,
					DocumentId:   document.ID,
					StageType:    vo.TaskStageStrategyConfirm,
					EventType:    vo.TaskEventUserAdjust,
					LogLevel:     vo.LogLevelInfo,
					OperatorType: utils.Ternary(cmd.OperatorId == 0, vo.OperatorTypeSystem, vo.OperatorTypeUser),
					OperatorId:   cmd.OperatorId,
					Content:      "用户调整了系统推荐策略。",
					DetailJson:   string(detailJson),
				}
				// 记录调整日志
				if err = d.repo.InsertTaskLog(txCtx, adjustLog); err != nil {
					return err
				}
			}
		} else {
			// 策略未变更：更新基础方案状态
			if err = d.repo.UpdatePlanById(txCtx, &entity.DocumentStrategyPlan{
				ID:            basePlan.ID,
				PlanStatus:    vo.PlanStatusConfirmed,
				PlanSource:    utils.Ternary(basePlan.PlanSource == 0, vo.PlanSourceSystemRecommend, basePlan.PlanSource),
				AdjustNote:    cmd.AdjustNote,
				ConfirmUserId: cmd.OperatorId,
				ConfirmTime:   time.Now(),
			}); err != nil {
				return err
			}
		}

		// 更新文档状态
		document.StrategyStatus = vo.StrategyStatusConfirmed
		document.FillEnumNames()

		if latestParseTask != nil {
			// 创建确认日志
			detailJson, _ := json.Marshal(map[string]any{
				"planId":              document.CurrentPlanId,
				"parentStrategyTypes": normalizedParentTypeList,
				"childStrategyTypes":  normalizedChildTypeList,
			})
			confirmLog := &entity.DocumentTaskLog{
				ID:           utils.GetSnowflakeNextID(),
				TaskId:       latestParseTask.ID,
				DocumentId:   document.ID,
				StageType:    vo.TaskStageStrategyConfirm,
				EventType:    vo.TaskEventUserConfirm,
				LogLevel:     vo.LogLevelInfo,
				OperatorType: utils.Ternary(cmd.OperatorId == 0, vo.OperatorTypeSystem, vo.OperatorTypeUser),
				OperatorId:   cmd.OperatorId,
				Content:      "用户确认了最终策略方案。",
				DetailJson:   string(detailJson),
			}
			if err = d.repo.InsertTaskLog(txCtx, confirmLog); err != nil {
				return err
			}
			// 更新任务阶段
			if err = d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: latestParseTask.ID, CurrentStage: vo.TaskStageStrategyConfirm}); err != nil {
				return err
			}
		}

		// 更新文档状态
		return d.repo.UpdateDocumentById(txCtx, &entity.Document{
			ID:             document.ID,
			StrategyStatus: document.StrategyStatus,
			CurrentPlanId:  document.CurrentPlanId,
		})
	}

	// 确认策略
	if err = d.repo.Do(ctx, confirmTx); err != nil {
		return nil, nil, err
	}

	return newPlan, document, nil
}

// BuildIndex 构建文档索引
func (d *LifecycleLogicImpl) BuildIndex(ctx context.Context, documentId, planId, operatorId int64) (*vo.DocumentIndexBuild, error) {
	// 基础校验：查询文档详情
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return nil, err
	}

	// 状态校验：文档必须完成解析且策略已确认
	if document.ParseStatus != vo.ParseStatusParseSuccess || document.StrategyStatus != vo.StrategyStatusConfirmed {
		return nil, common.NewBizError(errorx.ErrDocumentStatusInvalid.Code, "当前文档尚未完成\"解析成功 + 策略确认\"，不能构建索引")
	}

	// 方案一致性校验：请求的方案需与文档当前生效方案一致
	if document.CurrentPlanId != planId {
		return nil, common.NewBizError(errorx.ErrStrategyPlanNotFound.Code, "当前文档的生效方案与请求方案不一致。")
	}

	// 并发控制：检查是否存在同类型的活跃任务，防止重复构建
	runningTaskCount, err := d.repo.CountActiveTask(ctx, documentId, vo.TaskTypeBuildIndex, vo.TaskStatusNew, vo.TaskStatusRunning)
	if err != nil {
		return nil, err
	}
	if runningTaskCount > 0 {
		return nil, errorx.ErrIndexTaskRunning // 已有索引任务在运行中
	}

	// 查询策略方案，获取策略快照用于任务执行
	plan, err := d.repo.SelectPlanById(ctx, planId)
	if err != nil {
		return nil, err
	}

	// 创建索引构建任务实体
	taskId := utils.GetSnowflakeNextID()
	task := &entity.DocumentTask{
		ID:               taskId,
		DocumentId:       documentId,
		PlanId:           planId,
		TaskType:         vo.TaskTypeBuildIndex,                                                       // 任务类型：索引构建
		TaskStatus:       vo.TaskStatusNew,                                                            // 初始状态：新建
		CurrentStage:     vo.TaskStageChunkExecute,                                                    // 当前阶段：切分执行
		TriggerSource:    utils.Ternary(operatorId > 0, vo.TriggerSourceUser, vo.TriggerSourceSystem), // 判断触发来源
		StrategySnapshot: plan.StrategySnapshot,                                                       // 策略快照，确保任务执行时策略不变
	}

	// 构建任务日志详情JSON
	detail, _ := json.Marshal(map[string]any{
		"planId":           planId,
		"strategySnapshot": plan.StrategySnapshot,
	})
	// 创建任务日志实体
	taskLog := &entity.DocumentTaskLog{
		TaskId:       taskId,
		DocumentId:   documentId,
		StageType:    vo.TaskStageChunkExecute,
		EventType:    vo.TaskEventStart, // 事件类型：任务开始
		LogLevel:     vo.LogLevelInfo,   // 日志级别：信息
		OperatorType: utils.Ternary(operatorId > 0, vo.OperatorTypeUser, vo.OperatorTypeSystem),
		Content:      "索引构建任务已创建，等待异步执行",
		DetailJson:   string(detail),
	}

	// 事务性操作：更新文档状态、插入任务、插入任务日志
	fn := func(txCtx context.Context) error {
		// 更新文档状态为"构建中"
		if err := d.repo.UpdateDocumentById(txCtx, &entity.Document{ID: documentId, IndexStatus: vo.IndexStatusBuilding}); err != nil {
			return err
		}
		if err := d.repo.InsertTask(txCtx, task); err != nil {
			return err
		}
		return d.repo.InsertTaskLog(txCtx, taskLog)
	}
	if err = d.repo.Do(ctx, fn); err != nil {
		return nil, err
	}

	// 发送MQ消息触发异步索引构建
	indexBuildMessage := map[string]any{"documentId": documentId, "taskId": taskId, "planId": planId}
	if err = d.port.Send(ctx, d.indexTopic, strconv.FormatInt(documentId, 10), indexBuildMessage); err != nil {
		return nil, err
	}

	// 组装返回结果，填充枚举名称便于前端展示
	indexBuild := &vo.DocumentIndexBuild{
		DocumentId:  documentId,
		TaskId:      taskId,
		TaskType:    vo.TaskTypeBuildIndex,
		TaskStatus:  vo.TaskStatusNew,
		IndexStatus: vo.IndexStatusBuilding,
	}
	indexBuild.FillEnumNames()

	return indexBuild, nil
}

// QueryDocumentChunks 分页查询文档块列表
// 支持按任务ID查询，taskId为0时使用文档当前任务，返回文档块列表、总数、计划ID
func (d *LifecycleLogicImpl) QueryDocumentChunks(ctx context.Context, documentId, taskId int64, pageNo, pageSize int) ([]*entity.DocumentChunk, int64, int64, error) {
	// 查询文档信息
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return nil, 0, 0, err
	}

	// 获取有效的任务ID（taskId为0时使用文档当前任务）
	effectiveTaskId := d.getChunkTaskId(ctx, taskId, document)
	if effectiveTaskId == 0 {
		return nil, 0, document.CurrentPlanId, nil
	}

	// 查询任务信息并验证任务归属
	task, err := d.repo.SelectTaskById(ctx, effectiveTaskId)
	if err != nil {
		return nil, 0, 0, err
	}
	if task.DocumentId != document.ID {
		return nil, 0, 0, common.NewBizError(errorx.ErrDocumentNotFound.Code, "切块任务不存在。")
	}

	// 分页查询文档块列表
	chunkList, total, err := d.repo.SelectChunkPage(ctx, document.ID, effectiveTaskId, pageNo, pageSize)
	if err != nil {
		return nil, 0, 0, err
	}

	// 提取所有文档块的父块ID列表
	parentBlockIds := slice.Map(chunkList, func(index int, item *entity.DocumentChunk) int64 { return item.ParentBlockId })

	// 批量查询父块信息
	parentBlockList, err := d.repo.SelectParentBlockListByIds(ctx, parentBlockIds)
	if err != nil {
		return nil, 0, 0, err
	}

	// 构建父块ID到父块对象的映射
	parentBlockMap := utils.SliceToMapBy(parentBlockList,
		func(item *entity.DocumentParentBlock) (int64, *entity.DocumentParentBlock) { return item.ID, item })

	// 填充每个文档块的父块信息和枚举名称
	slice.ForEach(chunkList, func(index int, item *entity.DocumentChunk) {
		item.FillParentInfo(parentBlockMap[item.ParentBlockId])
		item.FillEnumName()
	})

	return chunkList, total, task.PlanId, nil
}

// QueryDocumentChunkDetail 查询文档块详情
// 返回文档块及其父块信息、兄弟块列表，taskId为0时使用文档当前任务
func (d *LifecycleLogicImpl) QueryDocumentChunkDetail(ctx context.Context, documentId, taskId, chunkId int64) (*aggregate.DocumentChunkDetail, error) {
	// 查询文档信息
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return nil, err
	}

	// 获取有效的任务ID（taskId为0时使用文档当前任务）
	effectiveTaskId := d.getChunkTaskId(ctx, taskId, document)
	if effectiveTaskId == 0 {
		return nil, common.NewBizError(errorx.ErrDocumentNotFound.Code, "当前文档还没有可查看的 chunk 详情。")
	}

	// 查询任务信息并验证任务归属
	task, err := d.repo.SelectTaskById(ctx, effectiveTaskId)
	if err != nil {
		return nil, err
	}
	if task.DocumentId != document.ID {
		return nil, common.NewBizError(errorx.ErrDocumentNotFound.Code, "切块任务不存在。")
	}

	// 查询指定文档块
	chunk, err := d.repo.SelectChunkById(ctx, chunkId, document.ID, effectiveTaskId)
	if err != nil {
		return nil, err
	}

	// 查询父块信息和兄弟块列表（如果有父块）
	var parentBlock *entity.DocumentParentBlock
	var siblingChunkList []*entity.DocumentChunk
	if chunk.ParentBlockId > 0 {
		parentBlock, err = d.repo.SelectParentBlockById(ctx, chunk.ParentBlockId, document.ID, effectiveTaskId)
		if err != nil {
			return nil, err
		}
		siblingChunkList, err = d.repo.SelectChunkListByParentBlockId(ctx, document.ID, effectiveTaskId, chunk.ParentBlockId)
		if err != nil {
			return nil, err
		}
	} else {
		// 无父块时，兄弟块列表只包含自身
		siblingChunkList = []*entity.DocumentChunk{chunk}
	}

	// 组装详情对象并填充父块信息
	detail := &aggregate.DocumentChunkDetail{
		DocumentId:    documentId,
		TaskId:        taskId,
		PlanId:        task.PlanId,
		Chunk:         chunk,
		SiblingChunks: siblingChunkList,
	}
	detail.FillParentInfo(parentBlock)

	return detail, nil
}

// QueryTaskLogs 查询任务日志
func (d *LifecycleLogicImpl) QueryTaskLogs(ctx context.Context, taskId int64, pageNo, pageSize int) (*entity.DocumentTask, int64, error) {
	task, err := d.repo.SelectTaskById(ctx, taskId)
	if err != nil {
		return nil, 0, err
	}

	logList, total, err := d.repo.SelectTaskLogPage(ctx, taskId, pageNo, pageSize)
	if err != nil {
		return nil, 0, err
	}

	task.Logs = logList
	task.FillEnumNames()

	return task, total, nil
}

// ListRetrievableDocuments 列出可检索的文档
func (d *LifecycleLogicImpl) ListRetrievableDocuments(ctx context.Context, documentIds ...int64) ([]*vo.KnowledgeDocument, error) {
	return d.repo.SelectRetrievableDocuments(ctx, documentIds...)
}

// QueryParentBlocks 查询父块列表
func (d *LifecycleLogicImpl) QueryParentBlocks(ctx context.Context, parentIds []int64) ([]*entity.DocumentParentBlock, error) {
	return d.repo.SelectParentBlockListByIds(ctx, parentIds)
}

// getChunkTaskId 获取文档块任务ID
func (d *LifecycleLogicImpl) getChunkTaskId(ctx context.Context, taskId int64, document *entity.Document) int64 {
	taskId = utils.Ternary(taskId == 0, document.LastIndexTaskId, taskId)
	if taskId == 0 {
		task, err := d.repo.SelectLatestTask(ctx, document.ID, vo.TaskTypeBuildIndex)
		if err != nil {
			return 0
		}
		taskId = task.ID
	}
	return taskId
}

// extractPipelineTypes 提取流水线类型
func (d *LifecycleLogicImpl) extractPipelineTypes(stepList []*entity.DocumentStrategyStep, pipelineType string) []int {
	result := slice.Filter(stepList, func(index int, item *entity.DocumentStrategyStep) bool {
		return utils.EqualsIgnoreCase(pipelineType, utils.BlankToDefault(item.PipelineType, vo.PipelineTypeChild))
	})
	slice.SortBy(result, func(a, b *entity.DocumentStrategyStep) bool { return a.StepNo < b.StepNo })
	return slice.Map(result, func(index int, item *entity.DocumentStrategyStep) int { return item.StrategyType })
}
