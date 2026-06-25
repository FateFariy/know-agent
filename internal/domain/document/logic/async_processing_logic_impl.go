package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// AsyncProcessingLogicImpl 异步处理业务逻辑实现
type AsyncProcessingLogicImpl struct {
	repo        adapter.DocumentRepository
	port        *adapter.DocumentPort
	parserLogic ParserLogic
}

// NewAsyncProcessingLogic 创建异步处理逻辑实例
func NewAsyncProcessingLogic(repo adapter.DocumentRepository, port *adapter.DocumentPort, parserLogic ParserLogic) *AsyncProcessingLogicImpl {
	return &AsyncProcessingLogicImpl{
		repo:        repo,
		port:        port,
		parserLogic: parserLogic,
	}
}

// HandleParseRoute 处理解析路由任务
/*
  这是文档上传成功后的第一条异步业务主链，整体顺序如下：
  1. 读取文档记录和任务记录，确认异步处理上下文存在；
  2. 把任务状态切到 RUNNING，把当前阶段推进到 CONTENT_PARSE；
  3. 从对象存储下载原始文件，并调用解析器提取纯文本和结构信息；
  4. 把解析后的纯文本重新上传为 txt，便于后续索引构建直接复用；
  5. 用结构节点服务替换文档结构节点，并同步导航索引、图投影和画像；
  6. 基于解析结果调用策略服务生成推荐切块方案；
  7. 把推荐方案和步骤写入数据库，同时更新文档的解析状态、策略状态和统计信息；
  8. 以成功或失败状态收尾任务，并记录任务日志。

  这个方法本身不直接执行向量化和 chunk 落库，那是后续"索引构建任务"的职责；
  当前阶段的目标是把文档从"原始文件"推进到"解析完成并拿到推荐策略"。
*/
func (d *AsyncProcessingLogicImpl) HandleParseRoute(ctx context.Context, documentId, taskId int64) error {
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		Warnf("查询解析文档失败: documentId=%d, err=%v", documentId, err)
		return err
	}

	task, err := d.repo.SelectTaskById(ctx, taskId)
	if err != nil {
		Warnf("查询解析任务失败: taskId=%d, err=%v", taskId, err)
		return err
	}

	task.TaskStatus = vo.TaskStatusRunning
	task.CurrentStage = vo.TaskStageContentParse
	task.StartTime = time.Now()

	document.ParseStatus = vo.ParseStatusParsing

	detail, _ := json.Marshal(map[string]any{"objectName": document.ObjectName})
	taskLog := &entity.DocumentTaskLog{
		TaskId:       taskId,
		DocumentId:   documentId,
		StageType:    vo.TaskStageContentParse,
		EventType:    vo.TaskEventStart,
		LogLevel:     vo.LogLevelInfo,
		OperatorType: vo.OperatorTypeSystem,
		Content:      "开始解析文档内容",
		DetailJson:   string(detail),
	}

	// 聚合文档、任务、任务日志并持久化
	agg := &aggregate.Document{
		Document: document,
		Task:     task,
		TaskLog:  taskLog,
	}
	fileBytes, err := d.port.DownloadObject(ctx, document.ObjectName)
	if err != nil {
		Warnf("下载文件失败: documentId=%d, err=%v", documentId, err)
		return err
	}

	if err = d.repo.UpdateDocumentAggregate(ctx, agg); err != nil {
		return err
	}

	// TODO: 实际解析逻辑 - 这里需要注入parserService
	// 2. 解析文档: parserService.parse(fileBytes, document.OriginalFileName, document.MimeType, fileType)
	// 3. 上传解析文本: storageService.uploadParsedText(documentId, analysisResult.ParsedText)
	// 4. 替换结构节点: structureNodeService.replaceDocumentNodes(documentId, taskId, analysisResult.StructureNodes)
	// 5. 同步导航产物: syncNavigationArtifacts(documentId, taskId, structureNodes)
	// 6. 生成文档属性: documentProfileService.generateProfile(documentId, analysisResult, structureNodes)

	// 模拟解析结果
	analysisResult := &vo.DocumentAnalysisResult{
		ParsedText:          "模拟解析文本内容",
		StructureNodes:      make([]*entity.DocumentStructureNode, 0),
		CharCount:           1000,
		TokenCount:          500,
		StructureLevel:      3,
		ContentQualityLevel: 2,
	}

	parseTextPath := "parsed/" + utils.ToString(documentId) + "/text.txt" // TODO: 实际调用storageService

	// 保存结构节点
	structureNodeCount := len(analysisResult.StructureNodes)
	for _, node := range analysisResult.StructureNodes {
		node.DocumentId = documentId
		node.ParseTaskId = taskId
		if err := d.repo.InsertStructureNode(ctx, node); err != nil {
			log.Printf("插入结构节点失败: err=%v", err)
		}
	}

	// TODO: 同步导航产物
	// TODO: 生成文档属性

	// 保存解析完成日志
	d.saveTaskLog(ctx, taskId, documentId, vo.TaskStageContentParse, vo.TaskEventTypeComplete, vo.LogLevelInfo,
		vo.OperatorTypeSystem, "", "文档解析完成。", map[string]interface{}{
			"charCount":           analysisResult.CharCount,
			"tokenCount":          analysisResult.TokenCount,
			"structureLevel":      analysisResult.StructureLevel,
			"contentQualityLevel": analysisResult.ContentQualityLevel,
			"structureNodeCount":  structureNodeCount,
		})

	// 更新任务阶段为策略路由
	task.CurrentStage = vo.TaskStageStrategyRoute
	if err := d.repo.UpdateTask(ctx, task); err != nil {
		log.Printf("更新任务阶段失败: taskId=%d, err=%v", taskId, err)
		return
	}

	// TODO: 推荐策略 - strategyService.recommendStrategy(document, analysisResult)
	planId := utils.GetSnowflakeNextID()
	planVersion, _ := d.repo.SelectLatestPlanVersion(ctx, documentId)
	planVersion++

	// 模拟策略方案
	plan := &entity.DocumentStrategyPlan{
		ID:               planId,
		DocumentId:       documentId,
		PlanVersion:      planVersion,
		PlanSource:       vo.PlanSourceSystemRecommend,
		PlanStatus:       vo.PlanStatusWaitConfirm,
		StrategyCount:    2,
		StrategySnapshot: "PARENT:1,CHILD:2",
		RecommendReason:  "系统自动推荐",
	}

	if err := d.repo.InsertPlan(ctx, plan); err != nil {
		log.Printf("插入策略方案失败: err=%v", err)
		return
	}

	// 模拟插入步骤
	parentSteps := []*entity.DocumentStrategyStepDraft{
		{PipelineType: "PARENT", StrategyType: 1, StrategyRole: 1, SourceType: 1, RecommendReason: "父块步骤1"},
	}
	childSteps := []*entity.DocumentStrategyStepDraft{
		{PipelineType: "CHILD", StrategyType: 2, StrategyRole: 1, SourceType: 1, RecommendReason: "子块步骤1"},
	}

	for i, draft := range parentSteps {
		step := &entity.DocumentStrategyStep{
			ID:              utils.GetSnowflakeNextID(),
			PlanId:          planId,
			DocumentId:      documentId,
			PipelineType:    draft.PipelineType,
			StepNo:          i + 1,
			StrategyType:    draft.StrategyType,
			StrategyRole:    draft.StrategyRole,
			SourceType:      draft.SourceType,
			ExecuteStatus:   vo.ExecuteStatusWaitExecute,
			RecommendReason: draft.RecommendReason,
		}
		if err := d.repo.InsertStep(ctx, step); err != nil {
			log.Printf("插入父块步骤失败: err=%v", err)
		}
	}

	for i, draft := range childSteps {
		step := &entity.DocumentStrategyStep{
			ID:              utils.GetSnowflakeNextID(),
			PlanId:          planId,
			DocumentId:      documentId,
			PipelineType:    draft.PipelineType,
			StepNo:          i + 1,
			StrategyType:    draft.StrategyType,
			StrategyRole:    draft.StrategyRole,
			SourceType:      draft.SourceType,
			ExecuteStatus:   vo.ExecuteStatusWaitExecute,
			RecommendReason: draft.RecommendReason,
		}
		if err := d.repo.InsertStep(ctx, step); err != nil {
			log.Printf("插入子块步骤失败: err=%v", err)
		}
	}

	// 更新文档信息
	document.ParseStatus = vo.ParseStatusParseSuccess
	document.StrategyStatus = vo.StrategyStatusRecommended
	document.CharCount = analysisResult.CharCount
	document.TokenCount = analysisResult.TokenCount
	document.StructureLevel = analysisResult.StructureLevel
	document.ContentQualityLevel = analysisResult.ContentQualityLevel
	document.ParseTextPath = parseTextPath
	document.ParseErrorMsg = ""
	document.CurrentPlanId = planId
	document.LastParseTaskId = taskId
	document.StructureNodeCount = structureNodeCount

	if err := d.repo.UpdateDocument(ctx, document); err != nil {
		log.Printf("更新文档信息失败: documentId=%d, err=%v", documentId, err)
		return
	}

	// 完成任务
	d.finishTaskSuccess(ctx, task, vo.TaskStageStrategyRoute, startTime)

	// 保存策略推荐日志
	d.saveTaskLog(ctx, taskId, documentId, vo.TaskStageStrategyRoute, vo.TaskEventTypeRecommendStrategy, vo.LogLevelInfo,
		vo.OperatorTypeSystem, "", "系统已生成推荐策略。", map[string]interface{}{
			"planId":             planId,
			"strategySnapshot":   plan.StrategySnapshot,
			"parentStepCount":    len(parentSteps),
			"childStepCount":     len(childSteps),
			"structureNodeCount": structureNodeCount,
			"recommendReason":    plan.RecommendReason,
		})
}

// HandleIndexBuild 处理索引构建任务
func (d *AsyncProcessingLogicImpl) HandleIndexBuild(ctx context.Context, documentId, taskId, planId int64) error {
	task, err := d.repo.SelectTaskById(ctx, taskId)
	if err != nil {
		Warnf("查询索引任务失败，taskId=%d， err:%v", taskId, err)
		return err
	}
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		Warnf("查询索引任务对应的文档失败，documentId=%d，err=%v", documentId, err)
		return err
	}

	plan, err := d.repo.SelectPlanById(ctx, planId)
	if err != nil {
		Warnf("查询索引任务对应的策略方案失败，planId=%d，err=%v", planId, err)
		return err
	}

	startTime := time.Now()

	// 查询步骤列表
	stepList, err := d.repo.SelectStepListByPlanId(ctx, planId)
	if err != nil {
		return err
	}

	// 更新任务状态为运行中
	task.TaskStatus = vo.TaskStatusRunning
	task.CurrentStage = vo.TaskStageChunkExecute
	task.StartTime = startTime

	// 更新文档索引状态
	document.IndexStatus = vo.IndexStatusBuilding

	// 保存开始切块日志
	chunkStartDetail, _ := json.Marshal(map[string]any{
		"strategySnapshot": plan.StrategySnapshot,
	})
	chunkStartTaskLog := &entity.DocumentTaskLog{
		TaskId:       taskId,
		DocumentId:   documentId,
		StageType:    vo.TaskStageChunkExecute,
		EventType:    vo.TaskEventStart,
		LogLevel:     vo.LogLevelInfo,
		OperatorType: vo.OperatorTypeSystem,
		Content:      "开始执行切块流水线。",
		DetailJson:   string(chunkStartDetail),
	}

	// 更新步骤执行状态为执行中
	if err = d.repo.UpdateStepExecuteStatus(ctx, planId, vo.ExecuteStatusExecuting); err != nil {
		return err
	}

	parsedText, err := d.port.DownloadText(ctx, document.ParseTextPath)
	if err != nil {
		Warnf("下载解析文本失败，documentId=%d，err=%v", documentId, err)
		return err
	}
	// TODO: 构建父块: strategyService.buildParentBlocks(document, plan, stepList, parsedText)
	parentBlockCandidates := make([]*vo.ParentBlockCandidate, 0)

	// 更新步骤执行状态为执行成功
	if err = d.repo.UpdateStepExecuteStatus(ctx, planId, vo.ExecuteStatusExecuteSuccess); err != nil {
		log.Printf("更新步骤执行状态失败: planId=%d, err=%v", planId, err)
	}

	// 统计块数量
	parentCount := len(parentBlockCandidates)
	childCount := 0
	for _, pb := range parentBlockCandidates {
		childCount += len(pb.ChildChunks)
	}

	// 保存切块完成日志
	chunkCompleteDetail, _ := json.Marshal(map[string]interface{}{
		"parentCount": parentCount,
		"childCount":  childCount,
	})
	chunkCompleteTaskLog := &entity.DocumentTaskLog{
		TaskId:       taskId,
		DocumentId:   documentId,
		StageType:    vo.TaskStageChunkExecute,
		EventType:    vo.TaskEventComplete,
		LogLevel:     vo.LogLevelInfo,
		OperatorType: vo.OperatorTypeSystem,
		Content:      "切块执行完成。",
		DetailJson:   string(chunkCompleteDetail),
	}
	if err := d.repo.InsertTaskLog(ctx, chunkCompleteTaskLog); err != nil {
		log.Printf("保存任务日志失败: taskId=%d, err=%v", taskId, err)
	}

	// 更新任务阶段为切块后处理
	task.CurrentStage = vo.TaskStageChunkPostProcess
	if err := d.repo.UpdateTask(ctx, task); err != nil {
		log.Printf("更新任务阶段失败: taskId=%d, err=%v", taskId, err)
		return
	}

	// 过滤有效的父块
	var finalParentBlockList []*vo.ParentBlockCandidate
	for _, item := range parentBlockCandidates {
		if item != nil && strings.TrimSpace(item.Text) != "" && len(item.ChildChunks) > 0 {
			hasValidChild := false
			for _, child := range item.ChildChunks {
				if strings.TrimSpace(child.Text) != "" {
					hasValidChild = true
					break
				}
			}
			if hasValidChild {
				finalParentBlockList = append(finalParentBlockList, item)
			}
		}
	}

	// 保存切块后处理完成日志
	d.saveTaskLog(ctx, taskId, documentId, vo.TaskStageChunkPostProcess, vo.TaskEventComplete, vo.LogLevelInfo,
		vo.OperatorTypeSystem, "", "切块后处理完成。", map[string]interface{}{
			"parentCount": len(finalParentBlockList),
			"childCount":  childCount,
		})

	// 构建实体并保存
	parentBlockEntityList, chunkEntityList := d.buildParentChildEntities(ctx, documentId, taskId, planId, finalParentBlockList)

	for _, parentBlock := range parentBlockEntityList {
		if err := d.repo.InsertParentBlock(ctx, parentBlock); err != nil {
			log.Printf("插入父块失败: err=%v", err)
		}
	}

	for _, chunk := range chunkEntityList {
		if err := d.repo.InsertChunk(ctx, chunk); err != nil {
			log.Printf("插入块失败: err=%v", err)
		}
	}

	// 更新任务阶段为向量化
	task.CurrentStage = vo.TaskStageVectorize
	if err := d.repo.UpdateTask(ctx, task); err != nil {
		log.Printf("更新任务阶段失败: taskId=%d, err=%v", taskId, err)
		return
	}

	// 保存开始向量化日志
	d.saveTaskLog(ctx, taskId, documentId, vo.TaskStageVectorize, vo.TaskEventTypeStart, vo.LogLevelInfo,
		vo.OperatorTypeSystem, "", "开始执行向量化。", map[string]interface{}{
			"chunkCount":         len(chunkEntityList),
			"embeddingBatchSize": 100, // DefaultDocumentVectorGateway.EMBEDDING_BATCH_SIZE_LIMIT
			"vectorStoreType":    "PG_VECTOR",
			"parentCount":        len(parentBlockEntityList),
		})

	// TODO: 向量化: vectorGateway.vectorize(chunkEntityList)
	// TODO: 关键词搜索索引: keywordSearchGateway.indexChunks(chunkEntityList)

	// 更新块向量状态
	for _, chunk := range chunkEntityList {
		chunk.VectorStatus = vo.VectorStatusBuildSuccess
		chunk.VectorId = utils.GetSnowflakeNextIDStr()
		if err := d.repo.UpdateChunk(ctx, chunk); err != nil {
			log.Printf("更新块向量状态失败: chunkId=%d, err=%v", chunk.ID, err)
		}
	}

	// 保存向量化完成日志
	d.saveTaskLog(ctx, taskId, documentId, vo.TaskStageVectorize, vo.TaskEventTypeComplete, vo.LogLevelInfo,
		vo.OperatorTypeSystem, "", "向量化完成。", map[string]interface{}{
			"chunkCount":         len(chunkEntityList),
			"embeddingBatchSize": 100,
			"vectorStoreType":    "PG_VECTOR",
			"parentCount":        len(parentBlockEntityList),
		})

	// 更新任务阶段为存储完成
	task.CurrentStage = vo.TaskStageStoreComplete
	if err := d.repo.UpdateTask(ctx, task); err != nil {
		log.Printf("更新任务阶段失败: taskId=%d, err=%v", taskId, err)
		return
	}

	// 更新方案状态为已执行
	if err := d.repo.UpdatePlanStatus(ctx, planId, vo.PlanStatusExecuted); err != nil {
		log.Printf("更新方案状态失败: planId=%d, err=%v", planId, err)
	}

	// 更新文档索引状态
	document.IndexStatus = vo.IndexStatusBuildSuccess
	document.LastIndexTaskId = taskId
	if err := d.repo.UpdateDocument(ctx, document); err != nil {
		log.Printf("更新文档索引状态失败: documentId=%d, err=%v", documentId, err)
		return
	}

	// 完成任务
	d.finishTaskSuccess(ctx, task, vo.TaskStageStoreComplete, startTime)

	// 保存存储完成日志
	d.saveTaskLog(ctx, taskId, documentId, vo.TaskStageStoreComplete, vo.TaskEventTypeComplete, vo.LogLevelInfo,
		vo.OperatorTypeSystem, "", "索引构建完成。", map[string]interface{}{
			"parentBlockCount": len(parentBlockEntityList),
			"chunkCount":       len(chunkEntityList),
		})
}

//
// // buildParentChildEntities 构建父块和子块实体列表
// func (d *DocumentAsyncProcessingLogicImpl) buildParentChildEntities(ctx context.Context, documentId, taskId, planId int64, candidates []*vo.ParentBlockCandidate) ([]*entity.DocumentParentBlock, []*entity.DocumentChunk) {
// 	parentBlockList := make([]*entity.DocumentParentBlock, 0, len(candidates))
// 	chunkList := make([]*entity.DocumentChunk, 0)
//
// 	for _, candidate := range candidates {
// 		parentBlockId := utils.GetSnowflakeNextID()
//
// 		// 创建父块实体
// 		parentBlock := &entity.DocumentParentBlock{
// 			DocumentId:  documentId,
// 			TaskId:      taskId,
// 			PlanId:      planId,
// 			ParentNo:    candidate.NodeId,
// 			SourceType:  candidate.SourceType,
// 			SectionPath: candidate.SectionPath,
// 			ParentText:  candidate.Text,
// 			CharCount:   len(candidate.Text),
// 			TokenCount:  len(candidate.Text) / 4, // 估算
// 			ChildCount:  len(candidate.ChildChunks),
// 			StartChunkNo: 0,
// 			EndChunkNo:   len(candidate.ChildChunks) - 1,
// 		}
//
// 		// 估算字符数和Token数
// 		for _, child := range candidate.ChildChunks {
// 			parentBlock.CharCount += len(child.Text)
// 			parentBlock.TokenCount += len(child.Text) / 4
// 		}
//
// 		parentBlockList = append(parentBlockList, parentBlock)
//
// 		// 创建子块实体
// 		for i, child := range candidate.ChildChunks {
// 			chunk := &entity.DocumentChunk{
// 				DocumentId:      documentId,
// 				TaskId:          taskId,
// 				PlanId:          planId,
// 				ParentBlockId:   parentBlockId,
// 				ChunkNo:         i + 1,
// 				SourceType:      child.SourceType,
// 				SectionPath:     child.SectionPath,
// 				StructureNodeId: child.NodeId,
// 				ChunkText:       child.Text,
// 				CharCount:       len(child.Text),
// 				TokenCount:      len(child.Text) / 4,
// 				VectorStatus:    vo.VectorStatusPending,
// 			}
// 			chunkList = append(chunkList, chunk)
// 		}
// 	}
//
// 	return parentBlockList, chunkList
// }
//
// // saveTaskLog 保存任务日志
// func (d *DocumentAsyncProcessingLogicImpl) saveTaskLog(ctx context.Context, taskId, documentId int64, stageType, eventType, logLevel, operatorType int, operatorId string, content string, detail map[string]interface{}) {
// 	taskLog := &entity.DocumentTaskLog{
// 		TaskId:       taskId,
// 		DocumentId:   documentId,
// 		StageType:    stageType,
// 		EventType:    eventType,
// 		LogLevel:     logLevel,
// 		OperatorType: operatorType,
// 		Content:      content,
// 		DetailJson:   utils.ToJsonString(detail),
// 	}
//
// 	if err := d.repo.InsertTaskLog(ctx, taskLog); err != nil {
// 		log.Printf("保存任务日志失败: taskId=%d, err=%v", taskId, err)
// 	}
// }
//
// // finishTaskSuccess 成功完成任务
// func (d *DocumentAsyncProcessingLogicImpl) finishTaskSuccess(ctx context.Context, task *entity.DocumentTask, currentStage, startTime int64) {
// 	endTime := time.Now().UnixMilli()
// 	task.TaskStatus = vo.TaskStatusSuccess
// 	task.CurrentStage = currentStage
// 	task.FinishTime = endTime
// 	task.CostMillis = endTime - startTime
//
// 	if err := d.repo.UpdateTask(ctx, task); err != nil {
// 		log.Printf("完成任务失败: taskId=%d, err=%v", task.ID, err)
// 	}
// }

func Warnf(format string, args ...any) {
	logx.Alert(fmt.Sprintf(format, args...))
}
