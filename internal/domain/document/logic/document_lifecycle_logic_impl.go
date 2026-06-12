package logic

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/slice"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/svc"
)

// DocumentManageCode 文档管理错误码
const (
	DocumentManageCodeDocumentNotFound     = 20003
	DocumentManageCodeStrategyPlanNotFound = 20005
	DocumentManageCodeStrategyStepEmpty    = 20006
	DocumentManageCodeIndexTaskRunning     = 20007
)

// BaseCodeParameterError 参数错误码
const (
	BaseCodeParameterError = 400
)

type DocumentLifecycleLogicImpl struct {
	conf config.Config
	port *adapter.DocumentPort
	repo adapter.DocumentRepository
}

var _ DocumentLifecycleLogic = (*DocumentLifecycleLogicImpl)(nil)

func NewDocumentLifecycleLogicImpl(svcCtx *svc.ServiceContext, port *adapter.DocumentPort, repo adapter.DocumentRepository) DocumentLifecycleLogic {
	return &DocumentLifecycleLogicImpl{
		conf: svcCtx.Config,
		port: port,
		repo: repo,
	}
}

// Upload 上传文档：完成文件上传、存储、文档记录创建及解析任务下发
func (d *DocumentLifecycleLogicImpl) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, document *entity.Document) (*vo.DocumentUpload, error) {
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
	storedObjectInfo, err := d.port.UploadOriginalFile(ctx, documentId, header.Filename, fileBytes, header.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}

	// 填充文档实体字段
	document.ID = documentId
	document.DocumentName = utils.Ternary(strings.TrimSpace(document.DocumentName) != "", document.DocumentName, header.Filename)
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
	document.KnowledgeScopeCode = strings.TrimSpace(document.KnowledgeScopeCode)
	document.KnowledgeScopeName = strings.TrimSpace(document.KnowledgeScopeName)
	document.BusinessCategory = strings.TrimSpace(document.BusinessCategory)
	document.DocumentTags = strings.TrimSpace(document.DocumentTags)

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
		Content:      "文件上传完成，已进入解析与策略推荐队列。",
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
	parseMessage := vo.DocumentParseMessage{
		DocumentId: documentId,
		TaskId:     taskId,
	}
	if err = d.port.Send(ctx, d.conf.MQ.ParseTopic, strconv.FormatInt(documentId, 10), parseMessage); err != nil {
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
func (d *DocumentLifecycleLogicImpl) QueryDocumentPage(ctx context.Context, pageNo, pageSize int, keyword string) ([]*entity.Document, int64, error) {
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
func (d *DocumentLifecycleLogicImpl) QueryDocumentDetail(ctx context.Context, documentId int64) (*entity.Document, error) {
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

//
// // DeleteDocument 删除文档
// func (d *DocumentLifecycleLogicImpl) DeleteDocument(ctx context.Context, documentId int64) (string, error) {
// 	document, err := d.repo.SelectDocumentById(ctx, documentId)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	// 检查是否有活跃任务
// 	activeTaskCount, err := d.repo.CountActiveTask(ctx, documentId, vo.TaskStatusNew, vo.TaskStatusRunning)
// 	if err != nil {
// 		return "", err
// 	}
// 	if activeTaskCount > 0 {
// 		return "", errorx.ErrDocumentStatusInvalid.Format("当前文档存在进行中的任务，请等待任务结束后再删除")
// 	}
//
// 	// 删除存储对象
// 	err = d.port.DeleteObjects(ctx, []string{document.ObjectName, document.ParseTextPath})
// 	if err != nil {
// 		return "", err
// 	}
//
// 	// 删除向量索引（TODO: 实现向量网关）
// 	// d.vectorGateway.DeleteByDocumentId(documentId)
//
// 	// 删除其他索引（TODO: 实现关键词搜索、导航索引、知识路由索引、结构图投影）
//
// 	// 删除相关数据
// 	err = d.repo.DeleteProfileByDocumentId(ctx, documentId)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	err = d.repo.DeleteTopicDocumentRelationByDocumentId(ctx, documentId)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	err = d.repo.DeleteParentBlockByDocumentId(ctx, documentId)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	err = d.repo.DeleteChunkByDocumentId(ctx, documentId)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	// TODO: 删除结构化节点
// 	// d.structureNodeService.DeleteByDocumentId(documentId)
//
// 	err = d.repo.DeleteTaskLogByDocumentId(ctx, documentId)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	err = d.repo.DeleteStepByDocumentId(ctx, documentId)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	err = d.repo.DeleteTaskByDocumentId(ctx, documentId)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	err = d.repo.DeletePlanByDocumentId(ctx, documentId)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	err = d.repo.DeleteDocumentById(ctx, documentId)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	return "", nil
// }
//
// // QueryStrategyPlan 查询策略方案
// func (d *DocumentLifecycleLogicImpl) QueryStrategyPlan(ctx context.Context, documentId int64) (*vo.DocumentStrategyPlanQueryVo, error) {
// 	document, err := d.repo.SelectDocumentById(ctx, documentId)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	var planVo *vo.DocumentStrategyPlanVo
// 	planReady := false
//
// 	if document.CurrentPlanId > 0 {
// 		plan, err := d.repo.SelectPlanById(ctx, document.CurrentPlanId)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if plan != nil && plan.Status == int(entity.BusinessStatusYes) {
// 			stepList, err := d.repo.QueryStepListByPlanId(ctx, plan.Id)
// 			if err != nil {
// 				return nil, err
// 			}
// 			planVo = d.toPlanVo(plan, stepList)
// 			planReady = true
// 		}
// 	}
//
// 	return &vo.DocumentStrategyPlanQueryVo{
// 		DocumentId:        document.Id,
// 		DocumentName:      document.DocumentName,
// 		ParseStatus:       document.ParseStatus,
// 		ParseStatusMsg:    d.enumMsg(vo.ParseStatus(document.ParseStatus)),
// 		StrategyStatus:    document.StrategyStatus,
// 		StrategyStatusMsg: d.enumMsg(vo.StrategyStatus(document.StrategyStatus)),
// 		IndexStatus:       document.IndexStatus,
// 		IndexStatusMsg:    d.enumMsg(vo.IndexStatus(document.IndexStatus)),
// 		ParseErrorMsg:     document.ParseErrorMsg,
// 		PlanReady:         planReady,
// 		Plan:              planVo,
// 	}, nil
// }
//
// // ConfirmStrategy 确认策略
// func (d *DocumentLifecycleLogicImpl) ConfirmStrategy(ctx context.Context, req *entity.DocumentStrategyConfirmDto) (*vo.DocumentStrategyConfirmVo, error) {
// 	document, err := d.getDocumentOrThrow(ctx, req.DocumentId)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if document.ParseStatus != int(vo.ParseStatusParseSuccess) {
// 		return nil, common.NewBizError(DocumentManageCodeDocumentStatusInvalid, "当前文档还未完成解析，不能确认策略。")
// 	}
//
// 	if document.CurrentPlanId != req.BasePlanId {
// 		return nil, common.NewBizError(DocumentManageCodeStrategyPlanNotFound, "当前文档的基础方案不存在或已切换。")
// 	}
//
// 	basePlan, err := d.repo.GetPlanById(ctx, req.BasePlanId)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if basePlan == nil || basePlan.Status != int(entity.BusinessStatusYes) {
// 		return nil, common.NewBizError(DocumentManageCodeStrategyPlanNotFound, "策略方案不存在")
// 	}
//
// 	baseStepList, err := d.repo.QueryStepListByPlanId(ctx, basePlan.Id)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	requestParentTypeList := d.extractStrategyTypes(req.ParentSteps)
// 	requestChildTypeList := d.extractStrategyTypes(req.ChildSteps)
//
// 	// TODO: 实现策略标准化
// 	// normalizedStepList := d.strategyService.NormalizeSteps(basePlan, baseStepList, requestParentTypeList, requestChildTypeList, req.DocumentId)
// 	normalizedStepList := baseStepList
//
// 	normalizedParentTypeList := d.extractPipelineTypes(normalizedStepList, vo.PipelineTypeParent)
// 	normalizedChildTypeList := d.extractPipelineTypes(normalizedStepList, vo.PipelineTypeChild)
//
// 	if len(normalizedParentTypeList) == 0 {
// 		return nil, common.NewBizError(DocumentManageCodeStrategyStepEmpty, "父块流水线不能为空。")
// 	}
// 	if len(normalizedChildTypeList) == 0 {
// 		return nil, common.NewBizError(DocumentManageCodeStrategyStepEmpty, "子块流水线不能为空。")
// 	}
//
// 	if len(normalizedStepList) == 0 {
// 		return nil, common.NewBizError(DocumentManageCodeStrategyStepEmpty, "策略步骤不能为空。")
// 	}
//
// 	baseParentTypeList := d.extractPipelineTypes(baseStepList, vo.PipelineTypeParent)
// 	baseChildTypeList := d.extractPipelineTypes(baseStepList, vo.PipelineTypeChild)
//
// 	requestDistinctParentTypeList := d.distinctIntList(requestParentTypeList)
// 	requestDistinctChildTypeList := d.distinctIntList(requestChildTypeList)
//
// 	normalized := !d.intListEqual(requestDistinctParentTypeList, normalizedParentTypeList) ||
// 		!d.intListEqual(requestDistinctChildTypeList, normalizedChildTypeList)
//
// 	changed := !d.intListEqual(baseParentTypeList, normalizedParentTypeList) ||
// 		!d.intListEqual(baseChildTypeList, normalizedChildTypeList)
//
// 	var targetPlanId int64
// 	var targetPlanVersion int
// 	var targetStepList []*entity.DocumentStrategyStep
//
// 	if !changed {
// 		basePlan.PlanStatus = int(vo.PlanStatusConfirmed)
// 		if basePlan.PlanSource == 0 {
// 			basePlan.PlanSource = int(vo.PlanSourceSystemRecommend)
// 		}
// 		basePlan.AdjustNote = req.AdjustNote
// 		basePlan.ConfirmUserId = d.parseOptionalLong(req.OperatorId)
// 		basePlan.ConfirmTime = time.Now().UnixMilli()
// 		err = d.repo.UpdatePlan(ctx, basePlan)
// 		if err != nil {
// 			return nil, err
// 		}
// 		targetPlanId = basePlan.Id
// 		targetPlanVersion = basePlan.PlanVersion
// 		targetStepList = baseStepList
// 	} else {
// 		basePlan.PlanStatus = int(vo.PlanStatusDiscarded)
// 		err = d.repo.UpdatePlan(ctx, basePlan)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		newPlanId := utils.GetSnowflakeNextID()
// 		newPlanVersion, err := d.repo.GetLatestPlanVersion(ctx, document.Id)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if newPlanVersion == 0 {
// 			newPlanVersion = 1
// 		}
//
// 		newPlan := &entity.DocumentStrategyPlan{
// 			Model: common.Model{
// 				Id:         newPlanId,
// 				CreateTime: time.Now().UnixMilli(),
// 				EditTime:   time.Now().UnixMilli(),
// 				Status:     int(entity.BusinessStatusYes),
// 			},
// 			DocumentId:       document.Id,
// 			PlanVersion:      newPlanVersion,
// 			PlanSource:       int(vo.PlanSourceUserAdjust),
// 			PlanStatus:       int(vo.PlanStatusConfirmed),
// 			StrategyCount:    len(normalizedStepList),
// 			StrategySnapshot: d.buildStrategySnapshot(normalizedStepList),
// 			RecommendReason:  basePlan.RecommendReason,
// 			AdjustNote:       req.AdjustNote,
// 			ConfirmUserId:    d.parseOptionalLong(req.OperatorId),
// 			ConfirmTime:      time.Now().UnixMilli(),
// 		}
//
// 		err = d.repo.InsertPlan(ctx, newPlan)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		for _, step := range normalizedStepList {
// 			step.Id = utils.GetSnowflakeNextID()
// 			step.PlanId = newPlanId
// 			step.Status = int(entity.BusinessStatusYes)
// 			err = d.repo.InsertStep(ctx, step)
// 			if err != nil {
// 				return nil, err
// 			}
// 		}
//
// 		targetPlanId = newPlanId
// 		targetPlanVersion = newPlanVersion
// 		targetStepList = normalizedStepList
// 	}
//
// 	document.CurrentPlanId = targetPlanId
// 	document.StrategyStatus = int(vo.StrategyStatusConfirmed)
// 	err = d.repo.UpdateDocument(ctx, document)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	latestParseTask, err := d.repo.GetLatestTask(ctx, document.Id, int(vo.TaskTypeParseRoute))
// 	if err != nil {
// 		return nil, err
// 	}
// 	if latestParseTask != nil {
// 		latestParseTask.CurrentStage = int(vo.TaskStageStrategyConfirm)
// 		err = d.repo.UpdateTask(ctx, latestParseTask)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		if changed {
// 			err = d.saveTaskLog(ctx, latestParseTask.ID, document.Id,
// 				int(vo.TaskStageStrategyConfirm),
// 				int(vo.TaskEventUserAdjust),
// 				int(vo.LogLevelInfo),
// 				d.resolveOperatorType(d.parseOptionalLong(req.OperatorId)),
// 				d.parseOptionalLong(req.OperatorId),
// 				"用户调整了系统推荐策略。",
// 				d.detail(
// 					"parentStrategyTypes", normalizedParentTypeList,
// 					"childStrategyTypes", normalizedChildTypeList,
// 					"adjustNote", req.AdjustNote,
// 				))
// 			if err != nil {
// 				return nil, err
// 			}
// 		}
//
// 		err = d.saveTaskLog(ctx, latestParseTask.ID, document.Id,
// 			int(vo.TaskStageStrategyConfirm),
// 			int(vo.TaskEventUserConfirm),
// 			int(vo.LogLevelInfo),
// 			d.resolveOperatorType(d.parseOptionalLong(req.OperatorId)),
// 			d.parseOptionalLong(req.OperatorId),
// 			"用户已确认最终策略方案。",
// 			map[string]interface{}{
// 				"planId":              targetPlanId,
// 				"parentStrategyTypes": normalizedParentTypeList,
// 				"childStrategyTypes":  normalizedChildTypeList,
// 			})
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
//
// 	return &vo.DocumentStrategyConfirmVo{
// 		DocumentId:        document.Id,
// 		PlanId:            targetPlanId,
// 		PlanVersion:       targetPlanVersion,
// 		StrategyStatus:    document.StrategyStatus,
// 		StrategyStatusMsg: d.enumMsg(vo.StrategyStatus(document.StrategyStatus)),
// 		Normalized:        normalized,
// 		ParentPipeline:    d.toPipelineVo(vo.PipelineTypeParent, targetStepList),
// 		ChildPipeline:     d.toPipelineVo(vo.PipelineTypeChild, targetStepList),
// 	}, nil
// }
//
// // BuildIndex 构建索引
// func (d *DocumentLifecycleLogicImpl) BuildIndex(ctx context.Context, req *entity.DocumentIndexBuildDto) (*vo.DocumentIndexBuildVo, error) {
// 	document, err := d.getDocumentOrThrow(ctx, req.DocumentId)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if document.ParseStatus != int(vo.ParseStatusParseSuccess) ||
// 		document.StrategyStatus != int(vo.StrategyStatusConfirmed) {
// 		return nil, common.NewBizError(DocumentManageCodeDocumentStatusInvalid, "当前文档尚未完成\"解析成功 + 策略确认\"，不能构建索引。")
// 	}
//
// 	if document.CurrentPlanId != req.PlanId {
// 		return nil, common.NewBizError(DocumentManageCodeStrategyPlanNotFound, "当前文档的生效方案与请求方案不一致。")
// 	}
//
// 	runningTaskCount, err := d.repo.CountActiveTask(ctx, document.Id, int(vo.TaskTypeBuildIndex))
// 	if err != nil {
// 		return nil, err
// 	}
// 	if runningTaskCount > 0 {
// 		return nil, common.NewBizError(DocumentManageCodeIndexTaskRunning, "索引任务正在运行中")
// 	}
//
// 	plan, err := d.repo.GetPlanById(ctx, req.PlanId)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if plan == nil || plan.Status != int(entity.BusinessStatusYes) {
// 		return nil, common.NewBizError(DocumentManageCodeStrategyPlanNotFound, "策略方案不存在")
// 	}
//
// 	taskId := utils.GetSnowflakeNextID()
// 	task := &entity.DocumentTask{
// 		Model: common.Model{
// 			Id:         taskId,
// 			CreateTime: time.Now().UnixMilli(),
// 			EditTime:   time.Now().UnixMilli(),
// 			Status:     int(entity.BusinessStatusYes),
// 		},
// 		DocumentId:       document.Id,
// 		PlanId:           req.PlanId,
// 		TaskType:         int(vo.TaskTypeBuildIndex),
// 		TaskStatus:       int(vo.TaskStatusNew),
// 		CurrentStage:     int(vo.TaskStageChunkExecute),
// 		TriggerSource:    d.resolveTriggerSource(d.parseOptionalLong(req.OperatorId)),
// 		StrategySnapshot: plan.StrategySnapshot,
// 		RetryCount:       0,
// 	}
//
// 	err = d.repo.InsertTask(ctx, task)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	document.IndexStatus = int(vo.IndexStatusBuilding)
// 	err = d.repo.UpdateDocument(ctx, document)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	err = d.saveTaskLog(ctx, taskId, document.Id,
// 		int(vo.TaskStageChunkExecute),
// 		int(vo.TaskEventStart),
// 		int(vo.LogLevelInfo),
// 		d.resolveOperatorType(d.parseOptionalLong(req.OperatorId)),
// 		d.parseOptionalLong(req.OperatorId),
// 		"索引构建任务已创建，等待异步执行。",
// 		map[string]interface{}{
// 			"planId":           req.PlanId,
// 			"strategySnapshot": plan.StrategySnapshot,
// 		})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	// 发送索引构建消息（TODO: 实现消息发送）
// 	// d.kafkaProducer.SendIndexBuild(&DocumentIndexBuildMessage{DocumentId: document.Id, TaskId: taskId, PlanId: req.PlanId})
//
// 	return &vo.DocumentIndexBuildVo{
// 		DocumentId:     document.Id,
// 		TaskId:         taskId,
// 		TaskType:       task.TaskType,
// 		TaskTypeMsg:    d.enumMsg(vo.TaskType(task.TaskType)),
// 		TaskStatus:     task.TaskStatus,
// 		TaskStatusMsg:  d.enumMsg(vo.TaskStatus(task.TaskStatus)),
// 		IndexStatus:    document.IndexStatus,
// 		IndexStatusMsg: d.enumMsg(vo.IndexStatus(document.IndexStatus)),
// 	}, nil
// }
//
// // QueryTaskLogs 查询任务日志
// func (d *DocumentLifecycleLogicImpl) QueryTaskLogs(ctx context.Context, req *entity.DocumentTaskLogQueryDto) (*vo.DocumentTaskLogQueryVo, error) {
// 	task, err := d.repo.GetTaskById(ctx, req.TaskId)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if task == nil || task.Status != int(entity.BusinessStatusYes) {
// 		return nil, common.NewBizError(DocumentManageCodeDocumentNotFound, "任务不存在。")
// 	}
//
// 	pageNo := req.PageNo
// 	if pageNo <= 0 {
// 		pageNo = 1
// 	}
// 	pageSize := req.PageSize
// 	if pageSize <= 0 {
// 		pageSize = 20
// 	}
//
// 	logList, total, err := d.repo.QueryTaskLogPage(ctx, req.TaskId, pageNo, pageSize)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	logVoList := make([]*vo.DocumentTaskLogVo, 0, len(logList))
// 	for _, log := range logList {
// 		logVoList = append(logVoList, d.toTaskLogVo(log))
// 	}
//
// 	return &vo.DocumentTaskLogQueryVo{
// 		TaskId:          task.ID,
// 		DocumentId:      task.DocumentId,
// 		TaskType:        task.TaskType,
// 		TaskTypeMsg:     d.enumMsg(vo.TaskType(task.TaskType)),
// 		TaskStatus:      task.TaskStatus,
// 		TaskStatusMsg:   d.enumMsg(vo.TaskStatus(task.TaskStatus)),
// 		CurrentStage:    task.CurrentStage,
// 		CurrentStageMsg: d.enumMsg(vo.TaskStage(task.CurrentStage)),
// 		StartTime:       task.StartTime,
// 		FinishTime:      task.FinishTime,
// 		CostMillis:      task.CostMillis,
// 		ErrorCode:       task.ErrorCode,
// 		ErrorMsg:        task.ErrorMsg,
// 		Total:           total,
// 		Logs:            logVoList,
// 	}, nil
// }
//
// // QueryDocumentChunks 查询文档块
// func (d *DocumentLifecycleLogicImpl) QueryDocumentChunks(ctx context.Context, req *entity.DocumentChunkQueryDto) (*vo.DocumentChunkQueryVo, error) {
// 	document, err := d.getDocumentOrThrow(ctx, req.DocumentId)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	pageNo := req.PageNo
// 	if pageNo <= 0 {
// 		pageNo = 1
// 	}
// 	pageSize := req.PageSize
// 	if pageSize <= 0 {
// 		pageSize = 20
// 	}
//
// 	effectiveTaskId := d.resolveChunkTaskId(document, req.TaskId)
// 	if effectiveTaskId == 0 {
// 		return &vo.DocumentChunkQueryVo{
// 			DocumentId: document.Id,
// 			TaskId:     0,
// 			PlanId:     document.CurrentPlanId,
// 			PageNo:     pageNo,
// 			PageSize:   pageSize,
// 			Total:      0,
// 			Records:    []*vo.DocumentChunkItemVo{},
// 		}, nil
// 	}
//
// 	task, err := d.repo.GetTaskById(ctx, effectiveTaskId)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if task == nil || task.Status != int(entity.BusinessStatusYes) || task.DocumentId != document.Id {
// 		return nil, common.NewBizError(DocumentManageCodeDocumentNotFound, "切块任务不存在。")
// 	}
//
// 	chunkList, total, err := d.repo.QueryChunkPage(ctx, document.Id, effectiveTaskId, pageNo, pageSize)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	parentBlockIds := make([]int64, 0, len(chunkList))
// 	for _, chunk := range chunkList {
// 		if chunk.ParentBlockId > 0 {
// 			parentBlockIds = append(parentBlockIds, chunk.ParentBlockId)
// 		}
// 	}
//
// 	parentBlockMap, err := d.listParentBlockMap(ctx, parentBlockIds)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	records := make([]*vo.DocumentChunkItemVo, 0, len(chunkList))
// 	for _, chunk := range chunkList {
// 		parentBlock := parentBlockMap[chunk.ParentBlockId]
// 		records = append(records, d.toDocumentChunkItemVo(chunk, parentBlock))
// 	}
//
// 	return &vo.DocumentChunkQueryVo{
// 		DocumentId: document.Id,
// 		TaskId:     effectiveTaskId,
// 		PlanId:     task.PlanId,
// 		PageNo:     pageNo,
// 		PageSize:   pageSize,
// 		Total:      total,
// 		Records:    records,
// 	}, nil
// }
//
// // QueryDocumentChunkDetail 查询文档块详情
// func (d *DocumentLifecycleLogicImpl) QueryDocumentChunkDetail(ctx context.Context, req *entity.DocumentChunkDetailQueryDto) (*vo.DocumentChunkDetailVo, error) {
// 	document, err := d.getDocumentOrThrow(ctx, req.DocumentId)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	effectiveTaskId := d.resolveChunkTaskId(document, req.TaskId)
// 	if effectiveTaskId == 0 {
// 		return nil, common.NewBizError(DocumentManageCodeDocumentNotFound, "当前文档还没有可查看的 chunk 详情。")
// 	}
//
// 	task, err := d.repo.GetTaskById(ctx, effectiveTaskId)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if task == nil || task.Status != int(entity.BusinessStatusYes) || task.DocumentId != document.Id {
// 		return nil, common.NewBizError(DocumentManageCodeDocumentNotFound, "切块任务不存在。")
// 	}
//
// 	chunk, err := d.repo.GetChunkById(ctx, req.ChunkId, document.Id, effectiveTaskId)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if chunk == nil {
// 		return nil, common.NewBizError(DocumentManageCodeDocumentNotFound, "chunk 详情不存在。")
// 	}
//
// 	var parentBlock *entity.DocumentParentBlock
// 	if chunk.ParentBlockId > 0 {
// 		parentBlock, err = d.repo.GetParentBlockById(ctx, chunk.ParentBlockId, document.Id, effectiveTaskId)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
//
// 	var siblingChunkList []*entity.DocumentChunk
// 	if chunk.ParentBlockId > 0 {
// 		siblingChunkList, err = d.repo.QueryChunkListByParentBlockId(ctx, document.Id, effectiveTaskId, chunk.ParentBlockId)
// 		if err != nil {
// 			return nil, err
// 		}
// 	} else {
// 		siblingChunkList = []*entity.DocumentChunk{chunk}
// 	}
//
// 	siblingVoList := make([]*vo.DocumentChunkItemVo, 0, len(siblingChunkList))
// 	for _, sibling := range siblingChunkList {
// 		siblingVoList = append(siblingVoList, d.toDocumentChunkItemVo(sibling, parentBlock))
// 	}
//
// 	return &vo.DocumentChunkDetailVo{
// 		DocumentId:    document.Id,
// 		TaskId:        effectiveTaskId,
// 		PlanId:        task.PlanId,
// 		Chunk:         d.toDocumentChunkItemVo(chunk, parentBlock),
// 		ParentBlock:   d.toDocumentParentBlockItemVo(parentBlock),
// 		SiblingChunks: siblingVoList,
// 	}, nil
// }
//
// func (d *DocumentLifecycleLogicImpl) resolveChunkTaskId(document *entity.Document, requestedTaskId int64) int64 {
// 	if requestedTaskId > 0 {
// 		return requestedTaskId
// 	}
// 	if document.LastIndexTaskId > 0 {
// 		return document.LastIndexTaskId
// 	}
// 	task, err := d.repo.GetLatestTask(ctx, document.Id, int(vo.TaskTypeBuildIndex))
// 	if err != nil || task == nil {
// 		return 0
// 	}
// 	return task.ID
// }
//
// func (d *DocumentLifecycleLogicImpl) listParentBlockMap(ctx context.Context, parentBlockIds []int64) (map[int64]*entity.DocumentParentBlock, error) {
// 	if len(parentBlockIds) == 0 {
// 		return map[int64]*entity.DocumentParentBlock{}, nil
// 	}
//
// 	parentBlockList, err := d.repo.QueryParentBlockListByIds(ctx, parentBlockIds)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	result := make(map[int64]*entity.DocumentParentBlock)
// 	for _, pb := range parentBlockList {
// 		result[pb.Id] = pb
// 	}
// 	return result, nil
// }
//
// func (d *DocumentLifecycleLogicImpl) saveTaskLog(ctx context.Context, taskId, documentId, stageType, eventType, logLevel, operatorType int, operatorId int64, content string, detail map[string]interface{}) error {
// 	// TODO: 序列化detail为JSON
// 	detailJson := ""
// 	if len(detail) > 0 {
// 		// 简化实现，实际应使用json.Marshal
// 		var parts []string
// 		for k, v := range detail {
// 			parts = append(parts, fmt.Sprintf("%s:%v", k, v))
// 		}
// 		detailJson = "{" + strings.Join(parts, ", ") + "}"
// 	}
//
// 	logRecord := &entity.DocumentTaskLog{
// 		Model: common.Model{
// 			Id:         utils.GetSnowflakeNextID(),
// 			CreateTime: time.Now().UnixMilli(),
// 			EditTime:   time.Now().UnixMilli(),
// 			Status:     int(entity.BusinessStatusYes),
// 		},
// 		TaskId:       taskId,
// 		DocumentId:   documentId,
// 		StageType:    stageType,
// 		EventType:    eventType,
// 		LogLevel:     logLevel,
// 		OperatorType: operatorType,
// 		OperatorId:   operatorId,
// 		Content:      content,
// 		DetailJson:   detailJson,
// 	}
//
// 	return d.repo.InsertTaskLog(ctx, logRecord)
// }
//
// // ===== 转换方法 =====
//
// func (d *DocumentLifecycleLogicImpl) toDocumentListItemVo(document *entity.Document, latestTask *entity.DocumentTask) *vo.DocumentListItemVo {
// 	vo := &vo.DocumentListItemVo{
// 		DocumentId:         document.Id,
// 		DocumentName:       document.DocumentName,
// 		OriginalFileName:   document.OriginalFileName,
// 		FileType:           document.FileType,
// 		FileTypeMsg:        d.enumMsg(document.FileType),
// 		FileSize:           document.FileSize,
// 		CharCount:          document.CharCount,
// 		TokenCount:         document.TokenCount,
// 		ParseStatus:        document.ParseStatus,
// 		ParseStatusMsg:     d.enumMsg(document.ParseStatus),
// 		StrategyStatus:     document.StrategyStatus,
// 		StrategyStatusMsg:  d.enumMsg(document.StrategyStatus),
// 		IndexStatus:        document.IndexStatus,
// 		IndexStatusMsg:     d.enumMsg(document.IndexStatus),
// 		ParseErrorMsg:      document.ParseErrorMsg,
// 		KnowledgeScopeCode: document.KnowledgeScopeCode,
// 		KnowledgeScopeName: document.KnowledgeScopeName,
// 		BusinessCategory:   document.BusinessCategory,
// 		DocumentTags:       document.DocumentTags,
// 		CurrentPlanId:      document.CurrentPlanId,
// 		LastIndexTaskId:    document.LastIndexTaskId,
// 		CreateTime:         document.CreateTime,
// 		EditTime:           document.EditTime,
// 	}
//
// 	if latestTask != nil {
// 		vo.LatestTaskId = latestTask.ID
// 		vo.LatestTaskType = latestTask.TaskType
// 		vo.LatestTaskTypeMsg = d.enumMsg(vo.TaskType(latestTask.TaskType))
// 		vo.LatestTaskStatus = latestTask.TaskStatus
// 		vo.LatestTaskStatusMsg = d.enumMsg(vo.TaskStatus(latestTask.TaskStatus))
// 	}
//
// 	return vo
// }
//
// func (d *DocumentLifecycleLogicImpl) toDocumentChunkItemVo(chunk *entity.DocumentChunk, parentBlock *entity.DocumentParentBlock) *vo.DocumentChunkItemVo {
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
// func (d *DocumentLifecycleLogicImpl) toDocumentParentBlockItemVo(parentBlock *entity.DocumentParentBlock) *vo.DocumentParentBlockItemVo {
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
// func (d *DocumentLifecycleLogicImpl) toPlanVo(plan *entity.DocumentStrategyPlan, stepList []*entity.DocumentStrategyStep) *vo.DocumentStrategyPlanVo {
// 	return &vo.DocumentStrategyPlanVo{
// 		PlanId:           plan.Id,
// 		PlanVersion:      plan.PlanVersion,
// 		PlanSource:       plan.PlanSource,
// 		PlanSourceMsg:    d.enumMsg(vo.PlanSource(plan.PlanSource)),
// 		PlanStatus:       plan.PlanStatus,
// 		PlanStatusMsg:    d.enumMsg(vo.PlanStatus(plan.PlanStatus)),
// 		StrategySnapshot: plan.StrategySnapshot,
// 		RecommendReason:  plan.RecommendReason,
// 		ParentPipeline:   d.toPipelineVo(vo.PipelineTypeParent, stepList),
// 		ChildPipeline:    d.toPipelineVo(vo.PipelineTypeChild, stepList),
// 	}
// }
//
// func (d *DocumentLifecycleLogicImpl) toPipelineVo(pipelineType vo.PipelineType, stepList []*entity.DocumentStrategyStep) *vo.DocumentStrategyPipelineVo {
// 	pipelineSteps := make([]*entity.DocumentStrategyStep, 0)
// 	for _, step := range stepList {
// 		pt := step.PipelineType
// 		if pt == "" {
// 			pt = "CHILD"
// 		}
// 		if strings.EqualFold(pt, vo.PipelineType(pipelineType).String()) {
// 			pipelineSteps = append(pipelineSteps, step)
// 		}
// 	}
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
// func (d *DocumentLifecycleLogicImpl) toStepVoList(stepList []*entity.DocumentStrategyStep) []*vo.DocumentStrategyStepVo {
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
// func (d *DocumentLifecycleLogicImpl) toTaskLogVo(logRecord *entity.DocumentTaskLog) *vo.DocumentTaskLogVo {
// 	return &vo.DocumentTaskLogVo{
// 		Id:           logRecord.Id,
// 		StageType:    logRecord.StageType,
// 		StageTypeMsg: d.enumMsg(vo.TaskStage(logRecord.StageType)),
// 		EventType:    logRecord.EventType,
// 		EventTypeMsg: d.enumMsg(vo.TaskEventType(logRecord.EventType)),
// 		LogLevel:     logRecord.LogLevel,
// 		LogLevelMsg:  d.enumMsg(vo.LogLevel(logRecord.LogLevel)),
// 		Content:      logRecord.Content,
// 		DetailJson:   logRecord.DetailJson,
// 		CreateTime:   logRecord.CreateTime,
// 	}
// }
//
// // ===== 工具方法 =====
//
// func (d *DocumentLifecycleLogicImpl) extractStrategyTypes(items []*entity.DocumentStrategyStepItemDto) []int {
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
// func (d *DocumentLifecycleLogicImpl) extractPipelineTypes(stepList []*entity.DocumentStrategyStep, pipelineType vo.PipelineType) []int {
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
// func (d *DocumentLifecycleLogicImpl) buildStrategySnapshot(stepList []*entity.DocumentStrategyStep) string {
// 	parentVo := d.toPipelineVo(vo.PipelineTypeParent, stepList)
// 	childVo := d.toPipelineVo(vo.PipelineTypeChild, stepList)
// 	return "PARENT:" + parentVo.StrategySnapshot + ";CHILD:" + childVo.StrategySnapshot
// }
//
// func (d *DocumentLifecycleLogicImpl) pipelineOrder(pipelineType string) int {
// 	if strings.EqualFold(pipelineType, "PARENT") {
// 		return 0
// 	}
// 	return 1
// }
//
// func (d *DocumentLifecycleLogicImpl) parsePipelineType(pipelineType string) int {
// 	if strings.EqualFold(pipelineType, "PARENT") {
// 		return int(vo.PipelineTypeParent)
// 	}
// 	return int(vo.PipelineTypeChild)
// }
//
// func (d *DocumentLifecycleLogicImpl) resolveTriggerSource(operatorId int64) int {
// 	if operatorId > 0 {
// 		return vo.TriggerSourceUser
// 	}
// 	return vo.TriggerSourceSystem
// }
//
// func (d *DocumentLifecycleLogicImpl) parseOptionalLong(rawValue string) int64 {
// 	if strings.TrimSpace(rawValue) == "" {
// 		return 0
// 	}
// 	value, err := strconv.ParseInt(rawValue, 10, 64)
// 	if err != nil || value <= 0 {
// 		return 0
// 	}
// 	return value
// }
//
// func (d *DocumentLifecycleLogicImpl) parseRequiredLong(rawValue, fieldName string) int64 {
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
// func (d *DocumentLifecycleLogicImpl) distinctIntList(list []int) []int {
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
// func (d *DocumentLifecycleLogicImpl) intListEqual(a, b []int) bool {
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
