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
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

type LifecycleLogicImpl struct {
	repo             adapter.DocumentRepository
	store            adapter.Storage
	knowledgeGateway adapter.KnowledgeGateway
	messageProducer  adapter.MessageProducer
	parseTopic       string
	indexTopic       string
}

var _ LifecycleLogic = (*LifecycleLogicImpl)(nil)

func NewLifecycleLogicImpl(svcCtx *svc.ServiceContext, messageProducer adapter.MessageProducer, store adapter.Storage,
	repo adapter.DocumentRepository, knowledgeGateway adapter.KnowledgeGateway) *LifecycleLogicImpl {
	return &LifecycleLogicImpl{
		repo:             repo,
		store:            store,
		knowledgeGateway: knowledgeGateway,
		messageProducer:  messageProducer,
		parseTopic:       svcCtx.Config.MQ.ParseTopic,
		indexTopic:       svcCtx.Config.MQ.IndexTopic,
	}
}

// Upload 上传文档：完成文件上传、存储、文档记录创建及解析任务下发
func (d *LifecycleLogicImpl) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, document *entity.Document) (*entity.Document, int64, error) {
	// 校验文件类型是否支持
	fileType := enum.DetectFileType(header.Filename)
	if fileType == enum.FileTypeUnknown {
		return nil, 0, errorx.ErrUnsupportedFileType.Format(fileType)
	}

	// 读取文件前512字节用于MIME类型检测
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	mimeType := http.DetectContentType(buf[:n])
	_, _ = file.Seek(0, io.SeekStart)

	// 读取完整文件内容
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, errorx.ErrEmptyFileContent.Format(err.Error())
	}

	// 校验知识库是否启用
	knowledgeBase, err := d.knowledgeGateway.RequireEnabled(ctx, document.KnowledgeBaseId)
	if err != nil {
		return nil, 0, err
	}

	// 生成全局唯一文档ID
	documentId := utils.GetSnowflakeNextID()

	// 上传原文件至MinIO存储
	storedObjectInfo, err := d.store.UploadOriginalFile(ctx, documentId, header.Filename, fileBytes, mimeType)
	if err != nil {
		return nil, 0, err
	}

	// 填充文档实体字段
	document.ID = documentId
	document.DocumentName = utils.BlankToDefault(strutil.Trim(document.DocumentName), header.Filename)
	document.OriginalFileName = header.Filename
	document.FileType = fileType
	document.MimeType = mimeType
	document.FileSize = int64(len(fileBytes))
	document.StorageType = d.store.Name()
	document.BucketName = storedObjectInfo.BucketName
	document.ObjectName = storedObjectInfo.ObjectName
	document.ObjectUrl = storedObjectInfo.ObjectUrl
	document.ParseStatus = enum.ParseStatusParsing
	document.StrategyStatus = enum.StrategyStatusWaitRecommend
	document.IndexStatus = enum.IndexStatusWaitBuild
	document.KnowledgeBaseId = knowledgeBase.ID
	document.KnowledgeBaseName = knowledgeBase.BaseName

	// 创建解析任务
	taskId := utils.GetSnowflakeNextID()
	task := &entity.DocumentTask{
		ID:            taskId,
		DocumentId:    documentId,
		TaskType:      enum.TaskTypeParseRoute,
		TaskStatus:    enum.TaskStatusNew,
		CurrentStage:  enum.TaskStageFileUpload,
		TriggerSource: utils.Ternary(document.OperatorId == 0, enum.TriggerSourceSystem, enum.TriggerSourceUser),
	}

	// 记录文件上传完成的任务日志
	detail, _ := json.Marshal(map[string]any{
		"originalFileName": header.Filename,
		"fileSize":         len(fileBytes),
	})

	taskLog := &entity.DocumentTaskLog{
		TaskId:       taskId,
		DocumentId:   documentId,
		StageType:    enum.TaskStageFileUpload,
		EventType:    enum.TaskEventComplete,
		LogLevel:     enum.LogLevelInfo,
		OperatorType: utils.Ternary(document.OperatorId == 0, enum.OperatorTypeSystem, enum.OperatorTypeUser),
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
		return nil, 0, err
	}

	// 发送解析消息至MQ，触发后续解析流程
	parseMessage := vo.DocumentParseRouteMessage{DocumentId: documentId, TaskId: taskId}
	if err = d.messageProducer.Send(ctx, d.parseTopic, strconv.FormatInt(documentId, 10), parseMessage); err != nil {
		return nil, 0, err
	}

	// 返回上传结果
	return document, taskId, nil
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
	activeTaskCount, err := d.repo.CountTaskByParams(ctx, documentId, 0, []int{enum.TaskStatusNew, enum.TaskStatusRunning})
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
	if document.ParseStatus != enum.ParseStatusParseSuccess {
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
	baseSteps, err := d.repo.SelectStepListByPlanId(ctx, basePlan.ID)
	if err != nil {
		return nil, nil, err
	}

	// 标准化策略步骤，过滤未知策略类型并去重
	normalized := cmd.NormalizeSteps()

	// 提取用户提交的策略类型列表
	parentStrategyTypes := cmd.GetSortedParentStrategyTypes()
	childStrategyTypes := cmd.GetSortedChildStrategyTypes()
	if len(parentStrategyTypes) == 0 {
		return nil, nil, common.NewBizError(errorx.ErrStrategyStepEmpty.Code, "父块流水线不能为空。")
	}
	if len(childStrategyTypes) == 0 {
		return nil, nil, common.NewBizError(errorx.ErrStrategyStepEmpty.Code, "子块流水线不能为空。")
	}

	// 构建父子流水线步骤
	steps := d.buildParentChildSteps(baseSteps, parentStrategyTypes, childStrategyTypes, cmd.DocumentId)

	// 判断是否发生了策略变更
	changed := !steps.EqualUnordered(baseSteps)

	// 查询最新解析任务信息
	latestParseTask, err := d.repo.SelectLatestTask(ctx, document.ID, enum.TaskTypeParseRoute)
	if err != nil {
		return nil, nil, err
	}

	newPlan := basePlan
	confirmTx := func(txCtx context.Context) error {
		// 根据是否变更处理方案
		if changed {
			// 策略发生变更：废弃旧方案
			if err = d.repo.UpdatePlanById(txCtx, &entity.DocumentStrategyPlan{ID: basePlan.ID, PlanStatus: enum.PlanStatusDiscarded}); err != nil {
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
				PlanSource:      enum.PlanSourceUserAdjust,
				PlanStatus:      enum.PlanStatusConfirmed,
				StrategyCount:   len(steps),
				RecommendReason: basePlan.RecommendReason,
				AdjustNote:      cmd.AdjustNote,
				ConfirmUserId:   cmd.OperatorId,
				ConfirmTime:     utils.Pointer(time.Now()),
			}

			newPlan.FillAndProcessPipeline(steps)

			// 更新文档的当前方案ID为新方案
			document.CurrentPlanId = newPlan.ID

			// 创建新方案
			if err = d.repo.InsertPlan(txCtx, newPlan); err != nil {
				return err
			}

			// 为步骤分配ID和方案ID
			for _, step := range steps {
				step.ID = utils.GetSnowflakeNextID()
				step.PlanId = newPlan.ID
			}

			// 插入新方案的步骤
			if err = d.repo.InsertStepBatch(txCtx, steps); err != nil {
				return err
			}

			// 构建调整日志
			if latestParseTask != nil {
				detailJson, _ := json.Marshal(map[string]any{
					"parentStrategyTypes": parentStrategyTypes,
					"childStrategyTypes":  childStrategyTypes,
					"adjustNote":          cmd.AdjustNote,
				})
				adjustLog := &entity.DocumentTaskLog{
					ID:           utils.GetSnowflakeNextID(),
					TaskId:       latestParseTask.ID,
					DocumentId:   document.ID,
					StageType:    enum.TaskStageStrategyConfirm,
					EventType:    enum.TaskEventUserAdjust,
					LogLevel:     enum.LogLevelInfo,
					OperatorType: utils.Ternary(cmd.OperatorId == 0, enum.OperatorTypeSystem, enum.OperatorTypeUser),
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
				PlanStatus:    enum.PlanStatusConfirmed,
				PlanSource:    utils.Ternary(basePlan.PlanSource == 0, enum.PlanSourceSystemRecommend, basePlan.PlanSource),
				AdjustNote:    cmd.AdjustNote,
				ConfirmUserId: cmd.OperatorId,
				ConfirmTime:   utils.Pointer(time.Now()),
			}); err != nil {
				return err
			}
		}

		// 更新文档状态
		document.StrategyStatus = enum.StrategyStatusConfirmed
		document.FillEnumNames()

		if latestParseTask != nil {
			// 创建确认日志
			detailJson, _ := json.Marshal(map[string]any{
				"planId":              document.CurrentPlanId,
				"parentStrategyTypes": parentStrategyTypes,
				"childStrategyTypes":  childStrategyTypes,
			})
			confirmLog := &entity.DocumentTaskLog{
				ID:           utils.GetSnowflakeNextID(),
				TaskId:       latestParseTask.ID,
				DocumentId:   document.ID,
				StageType:    enum.TaskStageStrategyConfirm,
				EventType:    enum.TaskEventUserConfirm,
				LogLevel:     enum.LogLevelInfo,
				OperatorType: utils.Ternary(cmd.OperatorId == 0, enum.OperatorTypeSystem, enum.OperatorTypeUser),
				OperatorId:   cmd.OperatorId,
				Content:      "用户确认了最终策略方案。",
				DetailJson:   string(detailJson),
			}
			if err = d.repo.InsertTaskLog(txCtx, confirmLog); err != nil {
				return err
			}
			// 更新任务阶段
			if err = d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: latestParseTask.ID, CurrentStage: enum.TaskStageStrategyConfirm}); err != nil {
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

	newPlan.Normalized = normalized

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
	if document.ParseStatus != enum.ParseStatusParseSuccess || document.StrategyStatus != enum.StrategyStatusConfirmed {
		return nil, common.NewBizError(errorx.ErrDocumentStatusInvalid.Code, `当前文档尚未完成"解析成功 + 策略确认"，不能构建索引`)
	}

	// 方案一致性校验：请求的方案需与文档当前生效方案一致
	if document.CurrentPlanId != planId {
		return nil, common.NewBizError(errorx.ErrStrategyPlanNotFound.Code, "当前文档的生效方案与请求方案不一致。")
	}

	// 并发控制：检查是否存在同类型的活跃任务，防止重复构建
	runningTaskCount, err := d.repo.CountTaskByParams(ctx, documentId, enum.TaskTypeBuildIndex, []int{enum.TaskStatusNew, enum.TaskStatusRunning})
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

	// 获取源解析任务
	sourceParseTaskId := document.LastParseTaskId
	var sourceParseTask *entity.DocumentTask
	if sourceParseTaskId > 0 {
		sourceParseTask, err = d.repo.SelectTaskById(ctx, sourceParseTaskId)
		if err != nil {
			return nil, err
		}
	}
	// 校验源解析任务是否存在且已完成
	if sourceParseTaskId <= 0 ||
		sourceParseTask.DocumentId != documentId ||
		sourceParseTask.TaskType != enum.TaskTypeParseRoute ||
		sourceParseTask.TaskStatus != enum.TaskStatusSuccess {
		return nil, common.NewBizError(errorx.ErrDocumentStatusInvalid.Code, "当前文档的解析任务不存在或未完成")
	}

	// 创建索引构建任务实体
	taskId := utils.GetSnowflakeNextID()
	task := &entity.DocumentTask{
		ID:                taskId,
		DocumentId:        documentId,
		PlanId:            planId,
		SourceParseTaskId: sourceParseTaskId,
		TaskType:          enum.TaskTypeBuildIndex,                                                         // 任务类型：索引构建
		TaskStatus:        enum.TaskStatusNew,                                                              // 初始状态：新建
		CurrentStage:      enum.TaskStageChunkExecute,                                                      // 当前阶段：切分执行
		TriggerSource:     utils.Ternary(operatorId > 0, enum.TriggerSourceUser, enum.TriggerSourceSystem), // 判断触发来源
		StrategySnapshot:  plan.StrategySnapshot,                                                           // 策略快照，确保任务执行时策略不变
	}

	// 构建任务日志详情JSON
	detail, _ := json.Marshal(map[string]any{
		"planId":            planId,
		"sourceParseTaskId": sourceParseTaskId,
		"strategySnapshot":  plan.StrategySnapshot,
	})
	// 创建任务日志实体
	taskLog := &entity.DocumentTaskLog{
		TaskId:       taskId,
		DocumentId:   documentId,
		StageType:    enum.TaskStageChunkExecute,
		EventType:    enum.TaskEventStart, // 事件类型：任务开始
		LogLevel:     enum.LogLevelInfo,   // 日志级别：信息
		OperatorType: utils.Ternary(operatorId > 0, enum.OperatorTypeUser, enum.OperatorTypeSystem),
		Content:      "索引构建任务已创建，等待异步执行",
		DetailJson:   string(detail),
	}

	// 事务性操作：更新文档状态、插入任务、插入任务日志
	fn := func(txCtx context.Context) error {
		// 更新文档状态为"构建中"
		if err := d.repo.UpdateDocumentById(txCtx, &entity.Document{ID: documentId, IndexStatus: enum.IndexStatusBuilding}); err != nil {
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

	// 组装返回结果，填充枚举名称便于前端展示
	indexBuild := &vo.DocumentIndexBuild{
		DocumentId:  documentId,
		TaskId:      taskId,
		TaskType:    enum.TaskTypeBuildIndex,
		TaskStatus:  enum.TaskStatusNew,
		IndexStatus: enum.IndexStatusBuilding,
	}

	// 发送MQ消息触发异步索引构建
	indexBuildMessage := vo.DocumentIndexBuildMessage{DocumentId: documentId, TaskId: taskId, PlanId: planId}
	if err = d.messageProducer.Send(ctx, d.indexTopic, strconv.FormatInt(documentId, 10), indexBuildMessage); err != nil {
		// 标记索引构建提交失败状态
		if err = d.markIndexBuildSubmitFailed(ctx, documentId, taskId, operatorId, err); err != nil {
			return nil, err
		}
		indexBuild.TaskStatus = enum.TaskStatusFailed
		indexBuild.IndexStatus = enum.IndexStatusBuildFailed
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
	chunks, total, err := d.repo.SelectChunkPage(ctx, document.ID, effectiveTaskId, pageNo, pageSize)
	if err != nil {
		return nil, 0, 0, err
	}

	// 提取所有文档块的父块ID列表
	parentChunkIds := slice.Map(chunks, func(index int, item *entity.DocumentChunk) int64 { return item.ParentChunkId })

	// 批量查询父块信息
	parentChunks, err := d.repo.SelectParentChunkListByIds(ctx, parentChunkIds)
	if err != nil {
		return nil, 0, 0, err
	}

	// 构建父块ID到父块对象的映射
	parentChunkMap := utils.MapBy(parentChunks,
		func(item *entity.DocumentParentChunk) (int64, *entity.DocumentParentChunk) { return item.ID, item })

	// 填充每个文档块的父块信息和枚举名称
	slice.ForEach(chunks, func(index int, item *entity.DocumentChunk) {
		item.FillParentInfo(parentChunkMap[item.ParentChunkId])
		item.FillEnumName()
	})

	return chunks, total, task.PlanId, nil
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
	var parentChunk *entity.DocumentParentChunk
	var siblingChunkList []*entity.DocumentChunk
	if chunk.ParentChunkId > 0 {
		parentChunk, err = d.repo.SelectParentChunkById(ctx, chunk.ParentChunkId, document.ID, effectiveTaskId)
		if err != nil {
			return nil, err
		}
		siblingChunkList, err = d.repo.SelectChunkListByParentChunkId(ctx, document.ID, effectiveTaskId, chunk.ParentChunkId)
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
	detail.FillParentInfo(parentChunk)

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
func (d *LifecycleLogicImpl) ListRetrievableDocuments(ctx context.Context, documentIds ...int64) ([]*vo.DocumentMetadata, error) {
	return d.repo.SelectRetrievableDocumentsByIds(ctx, documentIds...)
}

// QueryParentChunks 查询父块列表
func (d *LifecycleLogicImpl) QueryParentChunks(ctx context.Context, parentIds []int64) ([]*entity.DocumentParentChunk, error) {
	return d.repo.SelectParentChunkListByIds(ctx, parentIds)
}

// markIndexBuildSubmitFailed 标记索引构建提交失败状态
func (d *LifecycleLogicImpl) markIndexBuildSubmitFailed(ctx context.Context, documentId, taskId, operatorId int64, err error) error {
	logx.Errorf("索引构建消息投递失败，已标记任务失败，documentId=%d, taskId=%d, err=%v", documentId, taskId, err)
	fn := func(txCtx context.Context) error {
		task := &entity.DocumentTask{
			ID:         taskId,
			TaskStatus: enum.TaskStatusFailed,
			FinishTime: utils.Pointer(time.Now()),
			ErrorCode:  utils.Pointer("INDEX_BUILD_SUBMIT_FAILED"),
			ErrorMsg:   utils.Pointer(err.Error()),
		}
		if ferr := d.repo.UpdateTaskById(ctx, task); ferr != nil {
			return ferr
		}

		document := &entity.Document{
			ID:          documentId,
			IndexStatus: enum.IndexStatusBuildFailed,
		}
		if ferr := d.repo.UpdateDocumentById(ctx, document); ferr != nil {
			return ferr
		}

		detailJson, _ := json.Marshal(map[string]interface{}{
			"error": err.Error(),
		})
		log := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    enum.TaskStageChunkExecute,
			EventType:    enum.TaskEventFailed,
			LogLevel:     enum.LogLevelError,
			OperatorType: utils.Ternary(operatorId > 0, enum.OperatorTypeUser, enum.OperatorTypeSystem),
			OperatorId:   operatorId,
			Content:      "索引构建后台任务提交失败，未进入切块执行。",
			DetailJson:   string(detailJson),
		}
		_ = d.repo.InsertTaskLog(ctx, log)
		return nil
	}
	return d.repo.Do(ctx, fn)
}

// getChunkTaskId 获取文档块任务ID
func (d *LifecycleLogicImpl) getChunkTaskId(ctx context.Context, taskId int64, document *entity.Document) int64 {
	taskId = utils.Ternary(taskId == 0, document.LastIndexTaskId, taskId)
	if taskId == 0 {
		task, err := d.repo.SelectLatestTask(ctx, document.ID, enum.TaskTypeBuildIndex)
		if err != nil {
			return 0
		}
		taskId = task.ID
	}
	return taskId
}

// buildParentChildSteps 根据父、子策略类型生成对应的可执行步骤列表，并保留已存在的用户配置
func (d *LifecycleLogicImpl) buildParentChildSteps(baseSteps entity.DocumentStrategySteps, parentStrategyTypes []int, childStrategyTypes []int, documentId int64) entity.DocumentStrategySteps {
	// 按流水线+策略类型构建基础步骤映射（便于复用已存在的用户配置）
	baseStepMap := baseSteps.GroupByPipelineAndStrategyType()

	steps := make([]*entity.DocumentStrategyStep, 0, len(parentStrategyTypes)+len(childStrategyTypes))
	// 构建标准化步骤实体，若 baseStep 存在则标记为用户保留并复用原因；否则标记为用户追加
	buildSteps := func(pipelineType string, normalizedTypes []int) {
		for i, strategyType := range normalizedTypes {
			baseStep := baseStepMap[pipelineType][strategyType]
			step := &entity.DocumentStrategyStep{
				DocumentId:      documentId,
				PipelineType:    pipelineType,
				StepNo:          i + 1,
				StrategyType:    strategyType,
				StrategyRole:    enum.ResolveRole(i, strategyType),
				SourceType:      enum.StrategySourceTypeUserAdd,
				ExecuteStatus:   enum.StrategyExecuteStatusWaitExecute,
				RecommendReason: "用户手动追加该策略。",
			}
			if baseStep != nil {
				step.SourceType = enum.StrategySourceTypeUserKeep
				step.RecommendReason = baseStep.RecommendReason
			}
			steps = append(steps, step)
		}
	}

	// 生成父块标准化步骤
	buildSteps(enum.PipelineTypeParent, parentStrategyTypes)

	// 生成子块标准化步骤
	buildSteps(enum.PipelineTypeChild, childStrategyTypes)

	return steps
}
