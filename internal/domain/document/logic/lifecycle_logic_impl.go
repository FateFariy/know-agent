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
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"

	"github.com/swiftbit/know-agent/common/utils"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

type LifecycleLogicImpl struct {
	port       *adapter.DocumentPort
	repo       adapter.DocumentRepository
	parseTopic string
	indexTopic string
}

var _ LifecycleLogic = (*LifecycleLogicImpl)(nil)

func NewDocumentLifecycleLogicImpl(svcCtx *svc.ServiceContext, port *adapter.DocumentPort, repo adapter.DocumentRepository) *LifecycleLogicImpl {
	return &LifecycleLogicImpl{
		port:       port,
		repo:       repo,
		parseTopic: svcCtx.Config.MQ.ParseTopic,
		indexTopic: svcCtx.Config.MQ.IndexTopic,
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

	// 聚合文档、任务、任务日志并持久化
	agg := &aggregate.Document{
		Document: document,
		Task:     task,
		TaskLog:  taskLog,
	}
	if err = d.repo.InsertDocumentAggregate(ctx, agg); err != nil {
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
		plan.ParentPipeline = &entity.DocumentStrategyPipeline{PipelineType: vo.PipelineTypeParent}
		plan.ParentPipeline.FillAndProcessSteps(stepList)
		plan.ChildPipeline = &entity.DocumentStrategyPipeline{PipelineType: vo.PipelineTypeChild}
		plan.ChildPipeline.FillAndProcessSteps(stepList)
		document.PlanReady = true
		document.FillEnumNames()
	}

	return document, plan, nil
}

// ConfirmStrategy 确认策略
func (d *LifecycleLogicImpl) ConfirmStrategy(ctx context.Context, req *entity.DocumentStrategyConfirmDto) (*vo.DocumentStrategyConfirmVo, error) {
	document, err := d.getDocumentOrThrow(ctx, req.DocumentId)
	if err != nil {
		return nil, err
	}

	if document.ParseStatus != vo.ParseStatusParseSuccess {
		return nil, common.NewBizError(DocumentManageCodeDocumentStatusInvalid, "当前文档还未完成解析，不能确认策略。")
	}

	if document.CurrentPlanId != req.BasePlanId {
		return nil, common.NewBizError(DocumentManageCodeStrategyPlanNotFound, "当前文档的基础方案不存在或已切换。")
	}

	basePlan, err := d.repo.GetPlanById(ctx, req.BasePlanId)
	if err != nil {
		return nil, err
	}
	if basePlan == nil || basePlan.Status != int(entity.BusinessStatusYes) {
		return nil, common.NewBizError(DocumentManageCodeStrategyPlanNotFound, "策略方案不存在")
	}

	baseStepList, err := d.repo.QueryStepListByPlanId(ctx, basePlan.Id)
	if err != nil {
		return nil, err
	}

	requestParentTypeList := d.extractStrategyTypes(req.ParentSteps)
	requestChildTypeList := d.extractStrategyTypes(req.ChildSteps)

	// TODO: 实现策略标准化
	// normalizedStepList := d.strategyService.NormalizeSteps(basePlan, baseStepList, requestParentTypeList, requestChildTypeList, req.DocumentId)
	normalizedStepList := baseStepList

	normalizedParentTypeList := d.extractPipelineTypes(normalizedStepList, vo.PipelineTypeParent)
	normalizedChildTypeList := d.extractPipelineTypes(normalizedStepList, vo.PipelineTypeChild)

	if len(normalizedParentTypeList) == 0 {
		return nil, common.NewBizError(DocumentManageCodeStrategyStepEmpty, "父块流水线不能为空。")
	}
	if len(normalizedChildTypeList) == 0 {
		return nil, common.NewBizError(DocumentManageCodeStrategyStepEmpty, "子块流水线不能为空。")
	}

	if len(normalizedStepList) == 0 {
		return nil, common.NewBizError(DocumentManageCodeStrategyStepEmpty, "策略步骤不能为空。")
	}

	baseParentTypeList := d.extractPipelineTypes(baseStepList, vo.PipelineTypeParent)
	baseChildTypeList := d.extractPipelineTypes(baseStepList, vo.PipelineTypeChild)

	requestDistinctParentTypeList := d.distinctIntList(requestParentTypeList)
	requestDistinctChildTypeList := d.distinctIntList(requestChildTypeList)

	normalized := !d.intListEqual(requestDistinctParentTypeList, normalizedParentTypeList) ||
		!d.intListEqual(requestDistinctChildTypeList, normalizedChildTypeList)

	changed := !d.intListEqual(baseParentTypeList, normalizedParentTypeList) ||
		!d.intListEqual(baseChildTypeList, normalizedChildTypeList)

	var targetPlanId int64
	var targetPlanVersion int
	var targetStepList []*entity.DocumentStrategyStep

	if !changed {
		basePlan.PlanStatus = int(vo.PlanStatusConfirmed)
		if basePlan.PlanSource == 0 {
			basePlan.PlanSource = int(vo.PlanSourceSystemRecommend)
		}
		basePlan.AdjustNote = req.AdjustNote
		basePlan.ConfirmUserId = d.parseOptionalLong(req.OperatorId)
		basePlan.ConfirmTime = time.Now().UnixMilli()
		err = d.repo.UpdatePlan(ctx, basePlan)
		if err != nil {
			return nil, err
		}
		targetPlanId = basePlan.Id
		targetPlanVersion = basePlan.PlanVersion
		targetStepList = baseStepList
	} else {
		basePlan.PlanStatus = int(vo.PlanStatusDiscarded)
		err = d.repo.UpdatePlan(ctx, basePlan)
		if err != nil {
			return nil, err
		}

		newPlanId := utils.GetSnowflakeNextID()
		newPlanVersion, err := d.repo.GetLatestPlanVersion(ctx, document.Id)
		if err != nil {
			return nil, err
		}
		if newPlanVersion == 0 {
			newPlanVersion = 1
		}

		newPlan := &entity.DocumentStrategyPlan{
			Model: common.Model{
				Id:         newPlanId,
				CreateTime: time.Now().UnixMilli(),
				EditTime:   time.Now().UnixMilli(),
				Status:     int(entity.BusinessStatusYes),
			},
			DocumentId:       document.Id,
			PlanVersion:      newPlanVersion,
			PlanSource:       int(vo.PlanSourceUserAdjust),
			PlanStatus:       int(vo.PlanStatusConfirmed),
			StrategyCount:    len(normalizedStepList),
			StrategySnapshot: d.buildStrategySnapshot(normalizedStepList),
			RecommendReason:  basePlan.RecommendReason,
			AdjustNote:       req.AdjustNote,
			ConfirmUserId:    d.parseOptionalLong(req.OperatorId),
			ConfirmTime:      time.Now().UnixMilli(),
		}

		err = d.repo.InsertPlan(ctx, newPlan)
		if err != nil {
			return nil, err
		}

		for _, step := range normalizedStepList {
			step.Id = utils.GetSnowflakeNextID()
			step.PlanId = newPlanId
			step.Status = int(entity.BusinessStatusYes)
			err = d.repo.InsertStep(ctx, step)
			if err != nil {
				return nil, err
			}
		}

		targetPlanId = newPlanId
		targetPlanVersion = newPlanVersion
		targetStepList = normalizedStepList
	}

	document.CurrentPlanId = targetPlanId
	document.StrategyStatus = int(vo.StrategyStatusConfirmed)
	err = d.repo.UpdateDocument(ctx, document)
	if err != nil {
		return nil, err
	}

	latestParseTask, err := d.repo.GetLatestTask(ctx, document.Id, int(vo.TaskTypeParseRoute))
	if err != nil {
		return nil, err
	}
	if latestParseTask != nil {
		latestParseTask.CurrentStage = int(vo.TaskStageStrategyConfirm)
		err = d.repo.UpdateTask(ctx, latestParseTask)
		if err != nil {
			return nil, err
		}

		if changed {
			err = d.saveTaskLog(ctx, latestParseTask.ID, document.Id,
				int(vo.TaskStageStrategyConfirm),
				int(vo.TaskEventUserAdjust),
				int(vo.LogLevelInfo),
				d.resolveOperatorType(d.parseOptionalLong(req.OperatorId)),
				d.parseOptionalLong(req.OperatorId),
				"用户调整了系统推荐策略。",
				d.detail(
					"parentStrategyTypes", normalizedParentTypeList,
					"childStrategyTypes", normalizedChildTypeList,
					"adjustNote", req.AdjustNote,
				))
			if err != nil {
				return nil, err
			}
		}

		err = d.saveTaskLog(ctx, latestParseTask.ID, document.Id,
			int(vo.TaskStageStrategyConfirm),
			int(vo.TaskEventUserConfirm),
			int(vo.LogLevelInfo),
			d.resolveOperatorType(d.parseOptionalLong(req.OperatorId)),
			d.parseOptionalLong(req.OperatorId),
			"用户已确认最终策略方案。",
			map[string]any{
				"planId":              targetPlanId,
				"parentStrategyTypes": normalizedParentTypeList,
				"childStrategyTypes":  normalizedChildTypeList,
			})
		if err != nil {
			return nil, err
		}
	}

	return &vo.DocumentStrategyConfirmVo{
		DocumentId:        document.Id,
		PlanId:            targetPlanId,
		PlanVersion:       targetPlanVersion,
		StrategyStatus:    document.StrategyStatus,
		StrategyStatusMsg: d.enumMsg(vo.StrategyStatus(document.StrategyStatus)),
		Normalized:        normalized,
		ParentPipeline:    d.toPipelineVo(vo.PipelineTypeParent, targetStepList),
		ChildPipeline:     d.toPipelineVo(vo.PipelineTypeChild, targetStepList),
	}, nil
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

	// 更新文档状态为"构建中"
	document.IndexStatus = vo.IndexStatusBuilding

	// 创建索引构建任务实体
	taskId := utils.GetSnowflakeNextID() // 生成全局唯一任务ID
	task := &entity.DocumentTask{
		ID:               taskId,
		DocumentId:       documentId,
		PlanId:           planId,
		TaskType:         vo.TaskTypeBuildIndex,                                                       // 任务类型：索引构建
		TaskStatus:       vo.TaskStatusNew,                                                            // 初始状态：新建
		CurrentStage:     vo.TaskStageChunkExecute,                                                    // 当前阶段：切分执行
		TriggerSource:    utils.Ternary(operatorId > 0, vo.TriggerSourceUser, vo.TriggerSourceSystem), // 判断触发来源
		StrategySnapshot: plan.StrategySnapshot,                                                       // 策略快照，确保任务执行时策略不变
		RetryCount:       0,                                                                           // 重试次数初始化为0
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

	// 聚合文档、任务、日志，执行原子性保存
	agg := &aggregate.Document{
		Document: document,
		Task:     task,
		TaskLog:  taskLog,
	}
	if err = d.repo.InsertOrUpdateDocumentAggregate(ctx, agg); err != nil {
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

// QueryDocumentChunks 查询文档块

func (d *LifecycleLogicImpl) QueryDocumentChunks(ctx context.Context, documentId, taskId int64, pageNo, pageSize int) ([]*entity.DocumentChunk, int64, error) {
	document, err := d.getDocumentOrThrow(ctx, documentId)
	if err != nil {
		return nil, err
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	effectiveTaskId := d.resolveChunkTaskId(document, req.TaskId)
	if effectiveTaskId == 0 {
		return &vo.DocumentChunkQueryVo{
			DocumentId: document.Id,
			TaskId:     0,
			PlanId:     document.CurrentPlanId,
			PageNo:     pageNo,
			PageSize:   pageSize,
			Total:      0,
			Records:    []*vo.DocumentChunkItemVo{},
		}, nil
	}

	task, err := d.repo.GetTaskById(ctx, effectiveTaskId)
	if err != nil {
		return nil, err
	}
	if task == nil || task.Status != int(entity.BusinessStatusYes) || task.DocumentId != document.Id {
		return nil, common.NewBizError(DocumentManageCodeDocumentNotFound, "切块任务不存在。")
	}

	chunkList, total, err := d.repo.QueryChunkPage(ctx, document.Id, effectiveTaskId, pageNo, pageSize)
	if err != nil {
		return nil, err
	}

	parentBlockIds := make([]int64, 0, len(chunkList))
	for _, chunk := range chunkList {
		if chunk.ParentBlockId > 0 {
			parentBlockIds = append(parentBlockIds, chunk.ParentBlockId)
		}
	}

	parentBlockMap, err := d.listParentBlockMap(ctx, parentBlockIds)
	if err != nil {
		return nil, err
	}

	records := make([]*vo.DocumentChunkItemVo, 0, len(chunkList))
	for _, chunk := range chunkList {
		parentBlock := parentBlockMap[chunk.ParentBlockId]
		records = append(records, d.toDocumentChunkItemVo(chunk, parentBlock))
	}

	return &vo.DocumentChunkQueryVo{
		DocumentId: document.Id,
		TaskId:     effectiveTaskId,
		PlanId:     task.PlanId,
		PageNo:     pageNo,
		PageSize:   pageSize,
		Total:      total,
		Records:    records,
	}, nil
}

//
// // QueryDocumentChunkDetail 查询文档块详情
//
//	func (d *LifecycleLogicImpl) QueryDocumentChunkDetail(ctx context.Context, req *entity.DocumentChunkDetailQueryDto) (*vo.DocumentChunkDetailVo, error) {
//		document, err := d.getDocumentOrThrow(ctx, req.DocumentId)
//		if err != nil {
//			return nil, err
//		}
//
//		effectiveTaskId := d.resolveChunkTaskId(document, req.TaskId)
//		if effectiveTaskId == 0 {
//			return nil, common.NewBizError(DocumentManageCodeDocumentNotFound, "当前文档还没有可查看的 chunk 详情。")
//		}
//
//		task, err := d.repo.GetTaskById(ctx, effectiveTaskId)
//		if err != nil {
//			return nil, err
//		}
//		if task == nil || task.Status != int(entity.BusinessStatusYes) || task.DocumentId != document.Id {
//			return nil, common.NewBizError(DocumentManageCodeDocumentNotFound, "切块任务不存在。")
//		}
//
//		chunk, err := d.repo.GetChunkById(ctx, req.ChunkId, document.Id, effectiveTaskId)
//		if err != nil {
//			return nil, err
//		}
//		if chunk == nil {
//			return nil, common.NewBizError(DocumentManageCodeDocumentNotFound, "chunk 详情不存在。")
//		}
//
//		var parentBlock *entity.DocumentParentBlock
//		if chunk.ParentBlockId > 0 {
//			parentBlock, err = d.repo.GetParentBlockById(ctx, chunk.ParentBlockId, document.Id, effectiveTaskId)
//			if err != nil {
//				return nil, err
//			}
//		}
//
//		var siblingChunkList []*entity.DocumentChunk
//		if chunk.ParentBlockId > 0 {
//			siblingChunkList, err = d.repo.QueryChunkListByParentBlockId(ctx, document.Id, effectiveTaskId, chunk.ParentBlockId)
//			if err != nil {
//				return nil, err
//			}
//		} else {
//			siblingChunkList = []*entity.DocumentChunk{chunk}
//		}
//
//		siblingVoList := make([]*vo.DocumentChunkItemVo, 0, len(siblingChunkList))
//		for _, sibling := range siblingChunkList {
//			siblingVoList = append(siblingVoList, d.toDocumentChunkItemVo(sibling, parentBlock))
//		}
//
//		return &vo.DocumentChunkDetailVo{
//			DocumentId:    document.Id,
//			TaskId:        effectiveTaskId,
//			PlanId:        task.PlanId,
//			Chunk:         d.toDocumentChunkItemVo(chunk, parentBlock),
//			ParentBlock:   d.toDocumentParentBlockItemVo(parentBlock),
//			SiblingChunks: siblingVoList,
//		}, nil
//	}
//
//	func (d *LifecycleLogicImpl) resolveChunkTaskId(document *entity.Document, requestedTaskId int64) int64 {
//		if requestedTaskId > 0 {
//			return requestedTaskId
//		}
//		if document.LastIndexTaskId > 0 {
//			return document.LastIndexTaskId
//		}
//		task, err := d.repo.GetLatestTask(ctx, document.Id, int(vo.TaskTypeBuildIndex))
//		if err != nil || task == nil {
//			return 0
//		}
//		return task.ID
//	}
//
//	func (d *LifecycleLogicImpl) listParentBlockMap(ctx context.Context, parentBlockIds []int64) (map[int64]*entity.DocumentParentBlock, error) {
//		if len(parentBlockIds) == 0 {
//			return map[int64]*entity.DocumentParentBlock{}, nil
//		}
//
//		parentBlockList, err := d.repo.QueryParentBlockListByIds(ctx, parentBlockIds)
//		if err != nil {
//			return nil, err
//		}
//
//		result := make(map[int64]*entity.DocumentParentBlock)
//		for _, pb := range parentBlockList {
//			result[pb.Id] = pb
//		}
//		return result, nil
//	}
//
//	func (d *LifecycleLogicImpl) saveTaskLog(ctx context.Context, taskId, documentId, stageType, eventType, logLevel, operatorType int, operatorId int64, content string, detail map[string]any) error {
//		// TODO: 序列化detail为JSON
//		detailJson := ""
//		if len(detail) > 0 {
//			// 简化实现，实际应使用json.Marshal
//			var parts []string
//			for k, v := range detail {
//				parts = append(parts, fmt.Sprintf("%s:%v", k, v))
//			}
//			detailJson = "{" + strings.Join(parts, ", ") + "}"
//		}
//
//		logRecord := &entity.DocumentTaskLog{
//			Model: common.Model{
//				Id:         utils.GetSnowflakeNextID(),
//				CreateTime: time.Now().UnixMilli(),
//				EditTime:   time.Now().UnixMilli(),
//				Status:     int(entity.BusinessStatusYes),
//			},
//			TaskId:       taskId,
//			DocumentId:   documentId,
//			StageType:    stageType,
//			EventType:    eventType,
//			LogLevel:     logLevel,
//			OperatorType: operatorType,
//			OperatorId:   operatorId,
//			Content:      content,
//			DetailJson:   detailJson,
//		}
//
//		return d.repo.InsertTaskLog(ctx, logRecord)
//	}

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

//
// // ===== 转换方法 =====
//
// func (d *LifecycleLogicImpl) toDocumentChunkItemVo(chunk *entity.DocumentChunk, parentBlock *entity.DocumentParentBlock) *vo.DocumentChunkItemVo {
// 	vo := &vo.DocumentChunkItemVo{
// 		Id:              chunk.Id,
// 		ParentBlockId:   chunk.ParentBlockId,
// 		ChunkNo:         chunk.ChunkNo,
// 		SectionPath:     chunk.SectionPath,
// 		SourceType:      chunk.SourceType,
// 		SourceTypeMsg:   d.enumMsg(vo.ChunkSourceType(chunk.SourceType)),
// 		CharCount:       chunk.CharCount,
// 		TokenCount:      chunk.TokenCount,
// 		VectorStatus:    chunk.VectorStatus,
// 		VectorStatusMsg: d.enumMsg(vo.VectorStatus(chunk.VectorStatus)),
// 		ChunkText:       chunk.ChunkText,
// 	}
//
// 	if parentBlock != nil {
// 		vo.ParentNo = parentBlock.ParentNo
// 		vo.ChildCount = parentBlock.ChildCount
// 		vo.StartChunkNo = parentBlock.StartChunkNo
// 		vo.EndChunkNo = parentBlock.EndChunkNo
// 	}
//
// 	return vo
// }
//
// func (d *LifecycleLogicImpl) toDocumentParentBlockItemVo(parentBlock *entity.DocumentParentBlock) *vo.DocumentParentBlockItemVo {
// 	if parentBlock == nil {
// 		return nil
// 	}
// 	return &vo.DocumentParentBlockItemVo{
// 		Id:            parentBlock.Id,
// 		ParentNo:      parentBlock.ParentNo,
// 		SectionPath:   parentBlock.SectionPath,
// 		SourceType:    parentBlock.SourceType,
// 		SourceTypeMsg: d.enumMsg(vo.ChunkSourceType(parentBlock.SourceType)),
// 		CharCount:     parentBlock.CharCount,
// 		TokenCount:    parentBlock.TokenCount,
// 		ChildCount:    parentBlock.ChildCount,
// 		StartChunkNo:  parentBlock.StartChunkNo,
// 		EndChunkNo:    parentBlock.EndChunkNo,
// 		ParentText:    parentBlock.ParentText,
// 	}
// }
//

//
// 	// 按步骤序号排序
// 	sort.Slice(pipelineSteps, func(i, j int) bool {
// 		return pipelineSteps[i].StepNo < pipelineSteps[j].StepNo
// 	})
//
// 	// 构建策略快照
// 	var snapshotParts []string
// 	for _, step := range pipelineSteps {
// 		snapshotParts = append(snapshotParts, strconv.Itoa(step.StrategyType))
// 	}
// 	snapshot := strings.Join(snapshotParts, ",")
//
// 	return &vo.DocumentStrategyPipelineVo{
// 		PipelineType:     int(pipelineType),
// 		PipelineTypeMsg:  d.enumMsg(pipelineType),
// 		StrategySnapshot: snapshot,
// 		Steps:            d.toStepVoList(pipelineSteps),
// 	}
// }
//
// func (d *LifecycleLogicImpl) toStepVoList(stepList []*entity.DocumentStrategyStep) []*vo.DocumentStrategyStepVo {
// 	// 排序
// 	sortedSteps := make([]*entity.DocumentStrategyStep, len(stepList))
// 	copy(sortedSteps, stepList)
// 	sort.Slice(sortedSteps, func(i, j int) bool {
// 		orderI := d.pipelineOrder(sortedSteps[i].PipelineType)
// 		orderJ := d.pipelineOrder(sortedSteps[j].PipelineType)
// 		if orderI != orderJ {
// 			return orderI < orderJ
// 		}
// 		if sortedSteps[i].StepNo != sortedSteps[j].StepNo {
// 			return sortedSteps[i].StepNo < sortedSteps[j].StepNo
// 		}
// 		return sortedSteps[i].Id < sortedSteps[j].Id
// 	})
//
// 	voList := make([]*vo.DocumentStrategyStepVo, 0, len(sortedSteps))
// 	for _, step := range sortedSteps {
// 		voList = append(voList, &vo.DocumentStrategyStepVo{
// 			StepNo:           step.StepNo,
// 			PipelineType:     d.parsePipelineType(step.PipelineType),
// 			PipelineTypeMsg:  d.enumMsg(vo.PipelineType(d.parsePipelineType(step.PipelineType))),
// 			StrategyType:     step.StrategyType,
// 			StrategyTypeMsg:  d.enumMsg(vo.StrategyType(step.StrategyType)),
// 			StrategyRole:     step.StrategyRole,
// 			StrategyRoleMsg:  d.enumMsg(vo.StrategyRole(step.StrategyRole)),
// 			SourceType:       step.SourceType,
// 			SourceTypeMsg:    d.enumMsg(vo.StrategySourceType(step.SourceType)),
// 			ExecuteStatus:    step.ExecuteStatus,
// 			ExecuteStatusMsg: d.enumMsg(vo.ExecuteStatus(step.ExecuteStatus)),
// 			RecommendReason:  step.RecommendReason,
// 		})
// 	}
// 	return voList
// }
//
// // ===== 工具方法 =====
//
// func (d *LifecycleLogicImpl) extractStrategyTypes(items []*entity.DocumentStrategyStepItemDto) []int {
// 	if items == nil {
// 		return []int{}
// 	}
// 	// 按步骤序号排序
// 	sortedItems := make([]*entity.DocumentStrategyStepItemDto, len(items))
// 	copy(sortedItems, items)
// 	sort.Slice(sortedItems, func(i, j int) bool {
// 		noI := sortedItems[i].StepNo
// 		noJ := sortedItems[j].StepNo
// 		if noI == 0 {
// 			noI = int(^uint(0) >> 1) // max int
// 		}
// 		if noJ == 0 {
// 			noJ = int(^uint(0) >> 1)
// 		}
// 		return noI < noJ
// 	})
//
// 	result := make([]int, 0, len(sortedItems))
// 	for _, item := range sortedItems {
// 		if item.StrategyType > 0 {
// 			result = append(result, item.StrategyType)
// 		}
// 	}
// 	return result
// }
//
// func (d *LifecycleLogicImpl) extractPipelineTypes(stepList []*entity.DocumentStrategyStep, pipelineType vo.PipelineType) []int {
// 	result := make([]*entity.DocumentStrategyStep, 0)
// 	for _, step := range stepList {
// 		pt := step.PipelineType
// 		if pt == "" {
// 			pt = "CHILD"
// 		}
// 		if strings.EqualFold(pt, vo.PipelineType(pipelineType).String()) {
// 			result = append(result, step)
// 		}
// 	}
//
// 	sort.Slice(result, func(i, j int) bool {
// 		return result[i].StepNo < result[j].StepNo
// 	})
//
// 	types := make([]int, 0, len(result))
// 	for _, step := range result {
// 		types = append(types, step.StrategyType)
// 	}
// 	return types
// }
//
// func (d *LifecycleLogicImpl) buildStrategySnapshot(stepList []*entity.DocumentStrategyStep) string {
// 	parentVo := d.toPipelineVo(vo.PipelineTypeParent, stepList)
// 	childVo := d.toPipelineVo(vo.PipelineTypeChild, stepList)
// 	return "PARENT:" + parentVo.StrategySnapshot + ";CHILD:" + childVo.StrategySnapshot
// }
//
// func (d *LifecycleLogicImpl) pipelineOrder(pipelineType string) int {
// 	if strings.EqualFold(pipelineType, "PARENT") {
// 		return 0
// 	}
// 	return 1
// }
//
// func (d *LifecycleLogicImpl) parsePipelineType(pipelineType string) int {
// 	if strings.EqualFold(pipelineType, "PARENT") {
// 		return int(vo.PipelineTypeParent)
// 	}
// 	return int(vo.PipelineTypeChild)
// }
//
// func (d *LifecycleLogicImpl) parseRequiredLong(rawValue, fieldName string) int64 {
// 	if strings.TrimSpace(rawValue) == "" {
// 		panic(common.NewBizError(BaseCodeParameterError, fieldName+"不能为空。"))
// 	}
// 	value, err := strconv.ParseInt(rawValue, 10, 64)
// 	if err != nil || value <= 0 {
// 		panic(common.NewBizError(BaseCodeParameterError, fieldName+"格式不正确。"))
// 	}
// 	return value
// }
//
//
// func (d *LifecycleLogicImpl) distinctIntList(list []int) []int {
// 	seen := make(map[int]bool)
// 	result := make([]int, 0)
// 	for _, v := range list {
// 		if !seen[v] {
// 			seen[v] = true
// 			result = append(result, v)
// 		}
// 	}
// 	return result
// }
//
// func (d *LifecycleLogicImpl) intListEqual(a, b []int) bool {
// 	if len(a) != len(b) {
// 		return false
// 	}
// 	for i := range a {
// 		if a[i] != b[i] {
// 			return false
// 		}
// 	}
// 	return true
// }
//
