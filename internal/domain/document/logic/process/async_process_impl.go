package process

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/analysis"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

const (
	embeddingBatch = 100 // 默认向量化批大小
)

// AsyncProcessImpl 异步处理业务逻辑实现
//
//	HandleParseRoute → 解析路由（文件解析 + 结构节点 + 策略推荐）
//	HandleIndexBuild → 索引构建（切块流水线 + 向量化 + 落库 + GraphRAG + RAPTOR）
type AsyncProcessImpl struct {
	repo          adapter.DocumentRepository
	port          *adapter.DocumentPort
	analysisChain analysis.PhaseChain
	indexChain    index.PhaseChain
}

// NewAsyncProcessImpl 构造异步处理逻辑实例
func NewAsyncProcessImpl(repo adapter.DocumentRepository,
	port *adapter.DocumentPort,
	analysisChain analysis.PhaseChain,
	indexChain index.PhaseChain) *AsyncProcessImpl {
	return &AsyncProcessImpl{
		repo:          repo,
		port:          port,
		analysisChain: analysisChain,
		indexChain:    indexChain,
	}
}

// HandleParseRoute 处理解析路由任务
//
// 整体阶段：initialization → download → parse → upload → structure → strategy → finalization
func (d *AsyncProcessImpl) HandleParseRoute(ctx context.Context, documentId, taskId int64) (err error) {
	// 加载文档与任务实体
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return err
	}
	task, err := d.repo.SelectTaskById(ctx, taskId)
	if err != nil {
		return err
	}

	logx.Infof("开始异步解析文档，documentId=%d, taskId=%d, fileName=%s, fileType=%s, objectName=%s",
		documentId, taskId, document.OriginalFileName, enum.FileTypeName(document.FileType), document.ObjectName)

	// 记录开始时间 → 失败时统一调用 handleParseFailure
	startTime := time.Now()
	defer func() {
		if err != nil {
			d.handleParseFailure(ctx, document, task, err.Error())
		}
	}()

	// 构建上下文并执行责任链
	parseCtx := &analysis.Context{
		DocumentId: documentId,
		TaskId:     taskId,
		Document:   document,
		Task:       task,
		StartTime:  startTime,
	}

	if err = d.analysisChain.Run(ctx, parseCtx); err != nil {
		logx.Errorf("解析路由任务失败，documentId=%d, taskId=%d, err=%v", documentId, taskId, err)
		return err
	}
	logx.Infof("解析路由任务成功，documentId=%d, taskId=%d", documentId, taskId)

	return nil
}

// HandleIndexBuild 执行索引构建主流程（使用责任链模式）：准备 → 切块 → 向量化 → 关键词索引 → GraphRAG → RAPTOR → 完成
func (d *AsyncProcessImpl) HandleIndexBuild(ctx context.Context, documentId, taskId, planId int64) (err error) {
	// 加载任务相关实体
	task, err := d.repo.SelectTaskById(ctx, taskId)
	if err != nil {
		return err
	}
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return err
	}
	plan, err := d.repo.SelectPlanById(ctx, planId)
	if err != nil {
		return err
	}

	// 前置检查：如果任务已成功或失败，跳过重复执行
	if task.TaskStatus == enum.TaskStatusSuccess {
		logx.Infof("索引构建任务已成功，跳过重复执行，documentId=%d, taskId=%d, planId=%d",
			documentId, taskId, planId)
		return nil // 已完成，直接返回
	}
	if task.TaskStatus == enum.TaskStatusFailed {
		logx.Infof("索引构建任务已失败，跳过重复执行，documentId=%d, taskId=%d, planId=%d",
			documentId, taskId, planId)
		return nil // 已失败，直接返回
	}

	startTime := time.Now()

	// 读取已有的 GraphRAG 构建结果（用于断点恢复）
	graphRagBuildResult := task.ReadGraphRagBuildResult()

	// 检查是否需要直接失败
	if graphRagBuildResult != nil && graphRagBuildResult.OuterTaskDisposition == enum.OuterTaskDispositionFailIndexTask {
		d.applyGraphFailureDisposition(ctx, document, task, plan.ID, graphRagBuildResult, nil)
		return nil // 直接失败，无需继续执行
	}

	defer func() {
		if v := recover(); v != nil {
			var panicErr *vo.GraphRagBuildFailureException
			if errors.As(v, &panicErr) {
				d.handleGraphRagBuildFailure(ctx, document, task, plan, panicErr, startTime)
			}
		}
	}()

	// 构建上下文并执行责任链
	buildCtx := &index.Context{
		DocumentId: documentId,
		TaskId:     taskId,
		PlanId:     planId,
		Document:   document,
		Task:       task,
		Plan:       plan,
		StartTime:  startTime,
	}
	if err = d.indexChain.Run(ctx, buildCtx); err != nil {
		return err
	}
	logx.Infof("索引构建任务成功，documentId=%d, taskId=%d, planId=%d", documentId, taskId, planId)

	return nil
}

// HandleIndexBuildLegacy 原始的索引构建流程（保留用于对比参考）
// 切块流水线 → 父子块落库 → 向量化 → 关键词索引 → GraphRAG 构建 → RAPTOR 构建 → 状态收尾
func (d *AsyncProcessImpl) HandleIndexBuildLegacy(ctx context.Context, documentId, taskId, planId int64) (err error) {
	// 加载任务相关实体，失败直接返回，交由调度层观察
	task, err := d.repo.SelectTaskById(ctx, taskId)
	if err != nil {
		return err
	}
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return err
	}
	plan, err := d.repo.SelectPlanById(ctx, planId)
	if err != nil {
		return err
	}

	// 前置检查：如果任务已成功或失败，跳过重复执行
	if task.TaskStatus == enum.TaskStatusSuccess {
		logx.Infof("索引构建任务已成功，跳过重复执行，documentId=%d, taskId=%d, planId=%d", documentId, taskId, planId)
		return nil
	}
	if task.TaskStatus == enum.TaskStatusFailed {
		logx.Infof("索引构建任务已失败，跳过重复执行，documentId=%d, taskId=%d, planId=%d", documentId, taskId, planId)
		return nil
	}

	// 读取已有的 GraphRAG 构建结果（用于断点恢复）
	graphRagBuildResult := d.readGraphRagBuildResult(task)
	if graphRagBuildResult != nil && graphRagBuildResult.OuterTaskDisposition == enum.OuterTaskDispositionFailIndexTask {
		d.applyGraphFailureDisposition(ctx, document, task, plan.ID, graphRagBuildResult, nil)
		return nil
	}

	// 记录起始时间用于耗时统计；defer recover 统一捕获 panic 为失败状态
	startTime := time.Now()
	buildStartedNanos := time.Now()
	defer func() {
		if v := recover(); v != nil {
			if panicErr, ok := v.(*vo.GraphRagBuildFailureException); ok {
				d.handleGraphRagBuildFailure(ctx, document, task, plan, panicErr, startTime)
			} else if err2, ok := v.(error); ok {
				d.handleIndexBuildFailure(ctx, document, task, plan, err2.Error())
			}
		}
	}()

	logx.Infof("开始执行索引构建任务，documentId=%d, taskId=%d, planId=%d", documentId, taskId, planId)

	// 查询策略步骤列表
	pipelineSteps, err := d.repo.SelectStepListByPlanId(ctx, planId)
	if err != nil {
		return err
	}
	logx.Infof("索引构建策略步骤读取完成，documentId=%d, taskId=%d, planId=%d, stepCount=%d",
		documentId, taskId, planId, len(pipelineSteps))

	// 事务性推进任务状态到"切块执行中"
	markBuildingTx := func(txCtx context.Context) error {
		// 文档状态
		if err = d.repo.UpdateDocumentById(txCtx, &entity.Document{ID: document.ID, IndexStatus: enum.IndexStatusBuilding}); err != nil {
			return err
		}
		// 策略步骤标记执行中
		if err = d.repo.UpdateStepExecuteStatus(txCtx, plan.ID, enum.StrategyExecuteStatusExecuting); err != nil {
			return err
		}
		// 记录开始执行切块日志
		chunkStartDetail, _ := json.Marshal(map[string]any{"strategySnapshot": plan.StrategySnapshot})
		chunkStartLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    enum.TaskStageChunkExecute,
			EventType:    enum.TaskEventStart,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "开始执行切块流水线",
			DetailJson:   string(chunkStartDetail),
		}
		if err = d.repo.InsertTaskLog(txCtx, chunkStartLog); err != nil {
			return err
		}
		// 推进任务阶段为"切块执行中"
		return d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID:           taskId,
			TaskStatus:   enum.TaskStatusRunning,
			CurrentStage: enum.TaskStageChunkExecute,
			StartTime:    utils.Pointer(time.Now()),
		})
	}
	if err = d.repo.Do(ctx, markBuildingTx); err != nil {
		panic(err)
	}

	// 检查是否需要从已提交 GraphRAG 结果恢复
	resumeCommittedGraph := d.isCommittedGraph(graphRagBuildResult)
	var parentCandidates []*vo.ParentChunkCandidate
	var childChunks []*entity.DocumentChunk
	var parentBlocks []*entity.DocumentParentChunk

	if resumeCommittedGraph {
		// 从已提交的 GraphRAG outcome 恢复
		parentBlocks = []*entity.DocumentParentChunk{}
		childChunks, err = d.listFrozenSourceChunks(ctx, documentId, taskId)
		if err != nil {
			panic(err)
		}
		graphRagBuildResult = d.repairCrossDocumentProjection(ctx, document, documentId, taskId, graphRagBuildResult)
		logx.Infof("从已提交 GraphRAG outcome 恢复索引任务: documentId=%d, taskId=%d", documentId, taskId)
	} else {
		// 下载解析文本（已在解析路由阶段上传）
		parsedText, err := d.port.DownloadText(ctx, document.ParseTextPath)
		if err != nil {
			panic(err)
		}

		// 按步骤执行切块流水线，产出父-子块候选
		chunkStartedNanos := time.Now()
		parentCandidates, err = d.coordinator.BuildParentBlocks(ctx, document, pipelineSteps, parsedText)
		if err != nil {
			panic(err)
		}
		chunkCostMillis := time.Since(chunkStartedNanos).Milliseconds()
		logx.Infof("切块流水线执行完成，documentId=%d, taskId=%d, parentCount=%d, childCount=%d, costMillis=%d",
			documentId, taskId, len(parentCandidates), d.countChildCandidates(parentCandidates), chunkCostMillis)

		// 事务性标记切块完成 + 推进到切块后处理阶段
		markChunkCompleteTx := func(txCtx context.Context) error {
			// 策略步骤状态 -> 执行成功
			if err = d.repo.UpdateStepExecuteStatus(txCtx, plan.ID, enum.StrategyExecuteStatusExecuteSuccess); err != nil {
				return err
			}
			chunkEndDetail, _ := json.Marshal(map[string]any{
				"parentCount": len(parentCandidates),
				"childCount":  d.countChildCandidates(parentCandidates),
				"costMillis":  chunkCostMillis,
			})
			chunkEndLog := &entity.DocumentTaskLog{
				TaskId:       taskId,
				DocumentId:   documentId,
				StageType:    enum.TaskStageChunkExecute,
				EventType:    enum.TaskEventComplete,
				LogLevel:     enum.LogLevelInfo,
				OperatorType: enum.OperatorTypeSystem,
				Content:      "切块执行完成",
				DetailJson:   string(chunkEndDetail),
			}
			if err = d.repo.InsertTaskLog(txCtx, chunkEndLog); err != nil {
				return err
			}
			// 推进任务阶段到"切块后处理"
			return d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: taskId, CurrentStage: enum.TaskStageChunkPostProcess})
		}
		if err = d.repo.Do(ctx, markChunkCompleteTx); err != nil {
			panic(err)
		}

		// 清理候选并构造持久化实体（父块 + 子块）
		processStartedNanos := time.Now()
		finalCandidates := d.cleanupParentCandidates(parentCandidates)
		parentBlocks, childChunks = d.buildParentChildEntities(documentId, taskId, planId, finalCandidates)
		processCostMillis := time.Since(processStartedNanos).Milliseconds()
		logx.Infof("切块后处理完成，documentId=%d, taskId=%d, parentCount=%d, childCount=%d, costMillis=%d",
			documentId, taskId, len(finalCandidates), d.countChildCandidates(finalCandidates), processCostMillis)

		// 事务性批量落库 + 推进到向量化阶段
		persistBlocksTx := func(txCtx context.Context) error {
			// 批量写入父块
			if err = d.repo.InsertParentBlockBatch(txCtx, parentBlocks); err != nil {
				return err
			}
			// 批量写入子块
			if err = d.repo.InsertChunkBatch(txCtx, childChunks); err != nil {
				return err
			}
			// 记录"切块后处理完成"日志
			chunkPostDetail, _ := json.Marshal(map[string]any{
				"parentCount": len(finalCandidates),
				"childCount":  d.countChildCandidates(finalCandidates),
				"costMillis":  processCostMillis,
			})
			chunkPostLog := &entity.DocumentTaskLog{
				TaskId:       taskId,
				DocumentId:   documentId,
				StageType:    enum.TaskStageChunkPostProcess,
				EventType:    enum.TaskEventComplete,
				LogLevel:     enum.LogLevelInfo,
				OperatorType: enum.OperatorTypeSystem,
				Content:      "切块后处理完成",
				DetailJson:   string(chunkPostDetail),
			}
			if err = d.repo.InsertTaskLog(txCtx, chunkPostLog); err != nil {
				return err
			}
			// 推进任务阶段到"向量化"
			return d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: taskId, CurrentStage: enum.TaskStageVectorize})
		}
		if err = d.repo.Do(ctx, persistBlocksTx); err != nil {
			panic(err)
		}
	}

	// ========== 向量化阶段 ==========
	vectorSize := len(childChunks)
	vectorBatch := (vectorSize + embeddingBatch - 1) / embeddingBatch

	// 记录"开始执行向量化"日志
	markVectorStartTx := func(txCtx context.Context) error {
		vectorStartDetail, _ := json.Marshal(map[string]any{
			"chunkCount":          vectorSize,
			"embeddingBatchSize":  embeddingBatch,
			"embeddingBatchCount": vectorBatch,
			"vectorStoreType":     enum.VectorStoreTypeMilvus,
			"parentCount":         len(parentBlocks),
		})
		vectorStartLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    enum.TaskStageVectorize,
			EventType:    enum.TaskEventStart,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "开始执行向量化",
			DetailJson:   string(vectorStartDetail),
		}
		return d.repo.InsertTaskLog(txCtx, vectorStartLog)
	}
	if err = d.repo.Do(ctx, markVectorStartTx); err != nil {
		panic(err)
	}

	// 批量向量化
	vectorStartedNanos := time.Now()
	if err = d.port.BuildVectors(ctx, childChunks); err != nil {
		panic(err)
	}
	vectorCostMillis := time.Since(vectorStartedNanos).Milliseconds()

	// 回写向量状态
	for _, chunk := range childChunks {
		if err = d.repo.UpdateChunkByTaskId(ctx, chunk); err != nil {
			panic(err)
		}
	}

	// 记录"向量化完成"日志
	markVectorCompleteTx := func(txCtx context.Context) error {
		vectorEndDetail, _ := json.Marshal(map[string]any{
			"chunkCount":          vectorSize,
			"embeddingBatchSize":  embeddingBatch,
			"embeddingBatchCount": vectorBatch,
			"vectorStoreType":     enum.VectorStoreTypeMilvus,
			"parentCount":         len(parentBlocks),
			"vectorCostMillis":    vectorCostMillis,
		})
		vectorEndLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    enum.TaskStageVectorize,
			EventType:    enum.TaskEventComplete,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "向量化完成",
			DetailJson:   string(vectorEndDetail),
		}
		return d.repo.InsertTaskLog(txCtx, vectorEndLog)
	}
	if err = d.repo.Do(ctx, markVectorCompleteTx); err != nil {
		panic(err)
	}

	// ========== 关键词索引阶段 ==========
	markKeywordIndexTx := func(txCtx context.Context) error {
		if err = d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: taskId, CurrentStage: enum.TaskStageKeywordIndex}); err != nil {
			return err
		}
		keywordStartDetail, _ := json.Marshal(map[string]any{"chunkCount": vectorSize})
		keywordStartLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    enum.TaskStageKeywordIndex,
			EventType:    enum.TaskEventStart,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "开始构建关键词索引",
			DetailJson:   string(keywordStartDetail),
		}
		return d.repo.InsertTaskLog(txCtx, keywordStartLog)
	}
	if err = d.repo.Do(ctx, markKeywordIndexTx); err != nil {
		panic(err)
	}

	keywordStartedNanos := time.Now()
	if err = d.port.BuildIndexes(ctx, childChunks); err != nil {
		panic(err)
	}
	keywordCostMillis := time.Since(keywordStartedNanos).Milliseconds()

	markKeywordCompleteTx := func(txCtx context.Context) error {
		keywordEndDetail, _ := json.Marshal(map[string]any{
			"chunkCount": vectorSize,
			"costMillis": keywordCostMillis,
		})
		keywordEndLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    enum.TaskStageKeywordIndex,
			EventType:    enum.TaskEventComplete,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "关键词索引完成",
			DetailJson:   string(keywordEndDetail),
		}
		return d.repo.InsertTaskLog(txCtx, keywordEndLog)
	}
	if err = d.repo.Do(ctx, markKeywordCompleteTx); err != nil {
		panic(err)
	}

	// ========== GraphRAG 构建阶段 ==========
	if !resumeCommittedGraph {
		markGraphRagStartTx := func(txCtx context.Context) error {
			if err = d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: taskId, CurrentStage: enum.TaskStageGraphRag}); err != nil {
				return err
			}
			graphStartDetail, _ := json.Marshal(map[string]any{
				"chunkCount":  vectorSize,
				"parentCount": len(parentBlocks),
			})
			graphStartLog := &entity.DocumentTaskLog{
				TaskId:       taskId,
				DocumentId:   documentId,
				StageType:    enum.TaskStageGraphRag,
				EventType:    enum.TaskEventStart,
				LogLevel:     enum.LogLevelInfo,
				OperatorType: enum.OperatorTypeSystem,
				Content:      "开始构建 GraphRAG 实体关系图谱",
				DetailJson:   string(graphStartDetail),
			}
			return d.repo.InsertTaskLog(txCtx, graphStartLog)
		}
		if err = d.repo.Do(ctx, markGraphRagStartTx); err != nil {
			panic(err)
		}

		graphRagStartedNanos := time.Now()
		graphRagBuildResult, err = d.graphRagBuilder.RebuildDocumentGraph(ctx, documentId, taskId, childChunks)
		if err != nil {
			panic(&vo.GraphRagBuildFailureException{Result: graphRagBuildResult, Err: err})
		}
		graphRagCostMillis := time.Since(graphRagStartedNanos).Milliseconds()
		logx.Infof("GraphRAG 构建阶段完成，documentId=%d, taskId=%d, entityCount=%d, relationCount=%d, costMillis=%d",
			documentId, taskId, graphRagBuildResult.EntityCount, graphRagBuildResult.RelationCount, graphRagCostMillis)

		markGraphRagCompleteTx := func(txCtx context.Context) error {
			graphEndDetail, _ := json.Marshal(map[string]any{
				"entityCount":   graphRagBuildResult.EntityCount,
				"relationCount": graphRagBuildResult.RelationCount,
				"costMillis":    graphRagCostMillis,
			})
			graphEndLog := &entity.DocumentTaskLog{
				TaskId:       taskId,
				DocumentId:   documentId,
				StageType:    enum.TaskStageGraphRag,
				EventType:    enum.TaskEventComplete,
				LogLevel:     enum.LogLevelInfo,
				OperatorType: enum.OperatorTypeSystem,
				Content:      "GraphRAG 实体关系图谱构建完成",
				DetailJson:   string(graphEndDetail),
			}
			return d.repo.InsertTaskLog(txCtx, graphEndLog)
		}
		if err = d.repo.Do(ctx, markGraphRagCompleteTx); err != nil {
			panic(err)
		}
	}

	// ========== GraphRAG 结果处理 ==========
	graphFinalization := d.finalizeGraphRagOutcome(ctx, document, documentId, taskId, planId, task, childChunks, graphRagBuildResult, resumeCommittedGraph)
	graphRagBuildResult = graphFinalization.Result
	var graphTypedChunkList []vo.TypedChunk
	if graphFinalization.TypedChunks != nil {
		graphTypedChunkList = graphFinalization.TypedChunks
	}

	if graphRagBuildResult.OuterTaskDisposition == enum.OuterTaskDispositionRepairRequired {
		// 需要修复，保持 RUNNING 状态
		if err = d.repo.UpdateTaskById(ctx, &entity.DocumentTask{ID: taskId, TaskStatus: enum.TaskStatusRunning, CurrentStage: enum.TaskStageGraphTypedIndex}); err != nil {
			panic(err)
		}
		logx.Warnf("GraphRAG post-commit component requires repair; BUILD_INDEX remains RUNNING: documentId=%d, taskId=%d", documentId, taskId)
		return nil
	}
	if graphRagBuildResult.OuterTaskDisposition == enum.OuterTaskDispositionFailIndexTask {
		d.applyGraphFailureDisposition(ctx, document, task, plan.ID, graphRagBuildResult, nil)
		return nil
	}

	// ========== RAPTOR 构建阶段 ==========
	markRaptorStartTx := func(txCtx context.Context) error {
		if err = d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: taskId, CurrentStage: enum.TaskStageRaptor}); err != nil {
			return err
		}
		raptorStartDetail, _ := json.Marshal(map[string]any{
			"chunkCount":  vectorSize,
			"parentCount": len(parentBlocks),
		})
		raptorStartLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    enum.TaskStageRaptor,
			EventType:    enum.TaskEventStart,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "开始构建 RAPTOR 层级摘要树",
			DetailJson:   string(raptorStartDetail),
		}
		return d.repo.InsertTaskLog(txCtx, raptorStartLog)
	}
	if err = d.repo.Do(ctx, markRaptorStartTx); err != nil {
		panic(err)
	}

	raptorStartedNanos := time.Now()
	raptorBuildResult, err := d.raptorBuilder.RebuildDocumentTree(ctx, documentId, taskId, childChunks)
	if err != nil {
		panic(err)
	}
	raptorCostMillis := time.Since(raptorStartedNanos).Milliseconds()
	logx.Infof("RAPTOR 构建阶段完成，documentId=%d, taskId=%d, nodeCount=%d, levelCount=%d, costMillis=%d",
		documentId, taskId, raptorBuildResult.NodeCount, raptorBuildResult.LevelCount, raptorCostMillis)

	markRaptorCompleteTx := func(txCtx context.Context) error {
		raptorEndDetail, _ := json.Marshal(map[string]any{
			"nodeCount":   raptorBuildResult.NodeCount,
			"levelCount":  raptorBuildResult.LevelCount,
			"sourceCount": raptorBuildResult.SourceChunkCount,
			"costMillis":  raptorCostMillis,
		})
		raptorEndLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    enum.TaskStageRaptor,
			EventType:    enum.TaskEventComplete,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "RAPTOR 层级摘要树构建完成",
			DetailJson:   string(raptorEndDetail),
		}
		return d.repo.InsertTaskLog(txCtx, raptorEndLog)
	}
	if err = d.repo.Do(ctx, markRaptorCompleteTx); err != nil {
		panic(err)
	}

	// ========== 事务性最终状态更新（任务/方案/文档），并写入索引构建完成日志 ==========
	totalCostMillis := time.Since(buildStartedNanos).Milliseconds()
	finalizeTx := func(txCtx context.Context) error {
		// 任务阶段推进到"存储完成"
		if err = d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: taskId, CurrentStage: enum.TaskStageStoreComplete}); err != nil {
			return err
		}
		// 方案状态标记为已执行
		if err = d.repo.UpdatePlanById(txCtx, &entity.DocumentStrategyPlan{ID: planId, PlanStatus: enum.PlanStatusExecuted}); err != nil {
			return err
		}
		// 文档索引状态更新为构建成功
		if err = d.repo.UpdateDocumentById(txCtx, &entity.Document{ID: documentId, IndexStatus: enum.IndexStatusBuildSuccess, LastIndexTaskId: taskId}); err != nil {
			return err
		}
		// 写入成功耗时/统计日志
		if err = d.finishTaskSuccess(txCtx, task, enum.TaskStageStoreComplete, startTime); err != nil {
			panic(err)
		}
		// 索引构建完成日志
		buildCompleteDetail, _ := json.Marshal(map[string]any{
			"parentBlockCount":     len(parentBlocks),
			"chunkCount":           len(childChunks),
			"graphTypedChunkCount": len(graphTypedChunkList),
			"costMillis":           totalCostMillis,
		})
		buildCompleteLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    enum.TaskStageStoreComplete,
			EventType:    enum.TaskEventComplete,
			LogLevel:     enum.LogLevelInfo,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "索引构建完成",
			DetailJson:   string(buildCompleteDetail),
		}
		return d.repo.InsertTaskLog(txCtx, buildCompleteLog)
	}
	if err = d.repo.Do(ctx, finalizeTx); err != nil {
		panic(err)
	}
	logx.Infof("索引构建任务执行完成，documentId=%d, taskId=%d, planId=%d, parentCount=%d, chunkCount=%d, costMillis=%d",
		documentId, taskId, planId, len(parentBlocks), len(childChunks), totalCostMillis)
	return nil
}

// buildParentChildEntities 将父块候选转换为可落库的"父块实体 + 子块实体"双列表
// 关键信息：
//   - 每个父块维护 StartChunkNo / EndChunkNo（用于快速定位其覆盖的子块区间）
//   - 子块的 ChunkNo 在函数内全局递增
//   - 任何父块至少会得到 1 个兜底子块（由上层 buildParentBlocks 保证）
func (d *AsyncProcessImpl) buildParentChildEntities(documentId, taskId, planId int64,
	candidates []*vo.ParentChunkCandidate) ([]*entity.DocumentParentChunk, []*entity.DocumentChunk) {

	parentBlocks := make([]*entity.DocumentParentChunk, 0, len(candidates))
	chunks := make([]*entity.DocumentChunk, 0)

	// 全局子块编号：从 0 开始，遇到有效子块时递增并写入 ChunkNo
	globalChunkNo := 0
	for parentIdx, candidate := range candidates {
		parentBlock := &entity.DocumentParentChunk{
			ID:                utils.GetSnowflakeNextID(),
			DocumentId:        documentId,
			TaskId:            taskId,
			PlanId:            planId,
			ParentNo:          parentIdx + 1,
			SourceType:        candidate.SourceType,
			SectionPath:       candidate.SectionPath,
			StructureNodeId:   candidate.StructureNodeId,
			StructureNodeType: candidate.StructureNodeType,
			CanonicalPath:     candidate.CanonicalPath,
			ItemIndex:         candidate.ItemIndex,
			ParentText:        candidate.Text,
			CharCount:         utils.Len(candidate.Text),
			TokenCount:        utils.EstimateTokens(candidate.Text),
			StartChunkNo:      globalChunkNo,
		}

		// 遍历 ChildChunks：非空文本的子块才会被写入
		for _, child := range candidate.ChildChunks {
			if child != nil && strutil.IsNotBlank(child.Text) {
				globalChunkNo++
				chunks = append(chunks, &entity.DocumentChunk{
					ID:                utils.GetSnowflakeNextID(),
					DocumentId:        documentId,
					TaskId:            taskId,
					PlanId:            planId,
					ParentChunkId:     parentBlock.ID,
					ChunkNo:           globalChunkNo,
					SourceType:        child.SourceType,
					SectionPath:       utils.BlankToDefault(child.SectionPath, candidate.SectionPath),
					StructureNodeId:   child.StructureNodeId,
					StructureNodeType: child.StructureNodeType,
					CanonicalPath:     child.CanonicalPath,
					ItemIndex:         child.ItemIndex,
					ChunkText:         child.Text,
					CharCount:         utils.Len(child.Text),
					TokenCount:        utils.EstimateTokens(child.Text),
					VectorStatus:      enum.VectorStatusWaitVector,
				})
				parentBlock.ChildCount++
			}
		}
		// 更新当前父块的末尾 ChunkNo
		parentBlock.EndChunkNo = globalChunkNo - 1
		parentBlocks = append(parentBlocks, parentBlock)
	}
	return parentBlocks, chunks
}

// finishTaskSuccess 将任务标记为成功状态并写入耗时信息（毫秒），清空错误字段
func (d *AsyncProcessImpl) finishTaskSuccess(ctx context.Context, task *entity.DocumentTask, currentStage int, startTime time.Time) error {
	return d.repo.UpdateTaskById(ctx, &entity.DocumentTask{
		ID:           task.ID,
		TaskStatus:   enum.TaskStatusSuccess,
		CurrentStage: currentStage,
		FinishTime:   utils.Pointer(time.Now()),
		CostMillis:   time.Since(startTime).Milliseconds(),
		ErrorCode:    utils.Pointer(""),
		ErrorMsg:     utils.Pointer(""),
	})
}

// handleParseFailure 异步任务"解析路由"阶段失败时的统一收尾流程：先记录错误日志，再在事务内将文档状态、任务状态、失败详情与失败日志一次落库。
func (d *AsyncProcessImpl) handleParseFailure(ctx context.Context, document *entity.Document, task *entity.DocumentTask, errorMsg string) {
	logx.Errorf("异步解析文档失败，documentId=%d, taskId=%d, exception=%v", document.ID, task.ID, errorMsg)
	task.CurrentStage = utils.DefaultIfZero(task.CurrentStage, enum.TaskStageContentParse)
	task.TaskStatus = enum.TaskStatusFailed

	parseFailTx := func(txCtx context.Context) error {
		// 文档：标记为解析失败，并保留失败原因
		if err := d.repo.UpdateDocumentById(txCtx, &entity.Document{
			ID:            document.ID,
			ParseStatus:   enum.ParseStatusParseFailed,
			ParseErrorMsg: utils.Pointer(errorMsg),
		}); err != nil {
			return err
		}
		// 任务：标记为失败并停留在 CONTENT_PARSE
		if err := d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID:           task.ID,
			TaskStatus:   enum.TaskStatusFailed,
			CurrentStage: task.CurrentStage,
		}); err != nil {
			return err
		}
		// 通用失败收尾（写入耗时/错误码/错误消息）
		if err := d.failTask(txCtx, task, errorMsg); err != nil {
			return err
		}
		// 写入失败事件日志
		stageName := enum.TaskStageName(task.CurrentStage)
		failDetail, _ := json.Marshal(map[string]any{
			"error":            errorMsg,
			"currentStage":     task.CurrentStage,
			"currentStageName": stageName,
			"costMillis":       task.CostMillis,
		})
		failLog := &entity.DocumentTaskLog{
			TaskId:       task.ID,
			DocumentId:   task.DocumentId,
			StageType:    task.CurrentStage,
			EventType:    enum.TaskEventFailed,
			LogLevel:     enum.LogLevelError,
			OperatorType: enum.OperatorTypeSystem,
			Content:      fmt.Sprintf("文档解析失败，当前阶段：%s,耗时：%dms", stageName, task.CostMillis),
			DetailJson:   string(failDetail),
		}
		_ = d.repo.InsertTaskLog(txCtx, failLog)
		return nil
	}
	if err := d.repo.Do(ctx, parseFailTx); err != nil {
		logx.Warnf("解析失败时收尾失败: taskId=%d, err=%v", task.ID, err)
	}
}

// handleIndexBuildFailure "索引构建"失败时的统一收尾：事务性地将文档 IndexStatus、chunk 向量状态、step 执行状态、任务失败信息与日志一次落库。
func (d *AsyncProcessImpl) handleIndexBuildFailure(ctx context.Context, document *entity.Document, task *entity.DocumentTask, plan *entity.DocumentStrategyPlan, errorMsg string) {
	logx.Errorf("索引构建失败: documentId=%d, taskId=%d, planId=%d, err=%v", document.ID, task.ID, plan.ID, errorMsg)
	indexBuildFailTx := func(txCtx context.Context) error {
		// 文档：索引构建失败
		if err := d.repo.UpdateDocumentById(txCtx, &entity.Document{ID: document.ID, IndexStatus: enum.IndexStatusBuildFailed}); err != nil {
			return err
		}
		// chunk：按任务 ID 批量将向量状态置为失败（Milvus 为默认向量库类型）
		failedChunkMarker := &entity.DocumentChunk{
			TaskId:          task.ID,
			VectorStatus:    enum.VectorStatusVectorFailed,
			VectorStoreType: enum.VectorStoreTypeMilvus,
		}
		if err := d.repo.UpdateChunkByTaskId(txCtx, failedChunkMarker); err != nil {
			return err
		}
		// 标记当前计划所有步骤为失败
		if err := d.repo.UpdateStepExecuteStatus(txCtx, plan.ID, enum.StrategyExecuteStatusExecuteFailed); err != nil {
			return err
		}
		// 通用任务失败收尾（耗时/错误码等）
		if err := d.failTask(txCtx, task, errorMsg); err != nil {
			return err
		}
		// 写入"索引构建失败"日志
		failDetail, _ := json.Marshal(map[string]any{"error": errorMsg})
		failLog := &entity.DocumentTaskLog{
			TaskId:       task.ID,
			DocumentId:   task.DocumentId,
			StageType:    task.CurrentStage,
			EventType:    enum.TaskEventFailed,
			LogLevel:     enum.LogLevelError,
			OperatorType: enum.OperatorTypeSystem,
			Content:      "索引构建失败",
			DetailJson:   string(failDetail),
		}
		return d.repo.InsertTaskLog(txCtx, failLog)
	}
	if err := d.repo.Do(ctx, indexBuildFailTx); err != nil {
		logx.Warnf("索引构建失败时收尾失败: taskId=%d, err=%v", task.ID, err)
	}
}

// failTask 标记任务失败
func (d *AsyncProcessImpl) failTask(txCtx context.Context, task *entity.DocumentTask, errorMsg string) error {
	task.TaskStatus = enum.TaskStatusFailed
	task.FinishTime = utils.Pointer(time.Now())
	task.ErrorCode = utils.Pointer("TASK_FAILED")
	task.ErrorMsg = utils.Pointer(errorMsg)
	if task.StartTime == nil {
		task.CostMillis = time.Since(*task.StartTime).Milliseconds()
	}
	return d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{
		ID:           task.ID,
		TaskStatus:   task.TaskStatus,
		CurrentStage: task.CurrentStage,
		StartTime:    task.StartTime,
		FinishTime:   task.FinishTime,
		ErrorCode:    task.ErrorCode,
		ErrorMsg:     task.ErrorMsg,
		CostMillis:   task.CostMillis,
	})
}

// countChildCandidates 计算子块候选数
func (d *AsyncProcessImpl) countChildCandidates(parentBlockCandidateList []*vo.ParentChunkCandidate) int {
	count := 0
	for _, candidate := range parentBlockCandidateList {
		for _, child := range candidate.ChildChunks {
			if child != nil && strutil.IsNotBlank(child.Text) {
				count++
			}
		}
	}
	return count
}

// cleanupParentCandidates 过滤"文本为空"或"无子块"的父块候选
func (d *AsyncProcessImpl) cleanupParentCandidates(candidates []*vo.ParentChunkCandidate) []*vo.ParentChunkCandidate {
	return slice.Filter(candidates, func(_ int, item *vo.ParentChunkCandidate) bool {
		fn := func(child *vo.ChunkCandidate) bool { return child != nil && strutil.IsNotBlank(child.Text) }
		return item != nil && strutil.IsNotBlank(item.Text) && slices.ContainsFunc(item.ChildChunks, fn)
	})
}

// ============================================================
// GraphRAG 辅助方法
// ============================================================

// isCommittedGraph 检查图谱是否已提交
func (d *AsyncProcessImpl) isCommittedGraph(result *vo.GraphRagBuildResult) bool {
	return result != nil && result.KgCommitted && result.GraphPersistenceOutcome != "" && result.GraphPersistenceOutcome != enum.GraphPersistenceOutcomeFailed
}

// applyGraphFailureDisposition 应用图谱失败处置
func (d *AsyncProcessImpl) applyGraphFailureDisposition(ctx context.Context, document *entity.Document,
	task *entity.DocumentTask, planId int64, result *vo.GraphRagBuildResult, cause error) {

	task.CurrentStage = utils.DefaultIfZero(task.CurrentStage, enum.TaskStageGraphRag)
	if cause == nil {
		msg := "GraphRAG build failed."
		if result != nil && result.GraphPersistenceReason != "" {
			msg = result.GraphPersistenceReason
		}
		cause = errors.New(msg)
	}

	markFailureTx := func(txCtx context.Context) error {
		// 文档：索引构建失败
		document.IndexStatus = enum.IndexStatusBuildFailed
		if err := d.repo.UpdateDocumentById(txCtx, &entity.Document{
			ID:          document.ID,
			IndexStatus: document.IndexStatus,
		}); err != nil {
			return err
		}
		// 标记当前计划所有步骤为失败
		if err := d.repo.UpdateStepExecuteStatus(txCtx, planId, enum.StrategyExecuteStatusExecuteFailed); err != nil {
			return err
		}
		// 通用任务失败收尾
		return d.failTask(txCtx, task, cause.Error())
	}
	if err := d.repo.Do(ctx, markFailureTx); err != nil {
		logx.Warnf("图谱失败时收尾失败: taskId=%d, err=%v", task.ID, err)
	}
}

// handleGraphRagBuildFailure 处理 GraphRAG 构建失败
func (d *AsyncProcessImpl) handleGraphRagBuildFailure(ctx context.Context, document *entity.Document,
	task *entity.DocumentTask, plan *entity.DocumentStrategyPlan, exception *vo.GraphRagBuildFailureException, startTime time.Time) {
	logx.Errorf("GraphRAG 构建异常: documentId=%d, taskId=%d, planId=%d, err=%v", document.ID, task.ID, plan.ID, exception)

	failureResult := exception.Result
	if failureResult == nil {
		failureResult = &vo.GraphRagBuildResult{
			GraphPersistenceOutcome: enum.GraphPersistenceOutcomeFailed,
			KgCommitted:             false,
		}
	}

	// 计算最终结果
	terminalResult := d.graphRagOutcomePolicy.FinalizeOuterDisposition(
		failureResult,
		enum.ComponentOutcomeNotApplicable,
		enum.ObservationProjectionOutcomeSuccess,
	)

	// 尝试标记检查点
	if err := d.graphRagBuildCheckpoint.MarkOutcome(ctx, document.ID, task.ID, terminalResult, 0, 1); err != nil {
		logx.Warnf("GraphRAG 检查点标记失败: documentId=%d, taskId=%d, err=%v", document.ID, task.ID, err)
		// 降级为观察失败
		terminalResult = d.graphRagOutcomePolicy.FinalizeOuterDisposition(
			failureResult,
			enum.ComponentOutcomeNotApplicable,
			enum.ObservationProjectionOutcomeFailed,
		)
	}

	d.applyGraphFailureDisposition(ctx, document, task, plan.ID, terminalResult, exception)
}

// listFrozenSourceChunks 列出已冻结的源块（用于断点恢复）
func (d *AsyncProcessImpl) listFrozenSourceChunks(ctx context.Context, documentId, taskId int64) ([]*entity.DocumentChunk, error) {
	// TODO: 实现从数据库查询非 GRAPH_RAG 来源的 chunk
	return []*entity.DocumentChunk{}, nil
}

// listFrozenTypedChunks 列出已冻结的类型化块
func (d *AsyncProcessImpl) listFrozenTypedChunks(ctx context.Context, documentId, taskId int64) ([]*entity.DocumentChunk, error) {
	// TODO: 实现从数据库查询 GRAPH_RAG 来源的 chunk
	return []*entity.DocumentChunk{}, nil
}

// finalizeGraphRagOutcome 最终化 GraphRAG 结果
func (d *AsyncProcessImpl) finalizeGraphRagOutcome(ctx context.Context, document *entity.Document,
	documentId, taskId, planId int64, task *entity.DocumentTask, sourceChunks []*entity.DocumentChunk,
	buildResult *vo.GraphRagBuildResult, resumeCommittedGraph bool) *vo.GraphRagFinalization {

	if buildResult == nil || buildResult.GraphPersistenceOutcome == "" {
		panic(&vo.GraphRagBuildFailureException{Err: errors.New("GraphRAG build did not return an explicit save outcome.")})
	}

	var typedChunks []vo.TypedChunk
	typedOutcome := enum.ComponentOutcomeNotApplicable

	if !buildResult.KgCommitted || buildResult.GraphPersistenceOutcome == enum.GraphPersistenceOutcomeFailed {
		typedOutcome = enum.ComponentOutcomeNotApplicable
	} else {
		existingTypedChunks, err := d.listFrozenTypedChunks(ctx, documentId, taskId)
		if err != nil {
			panic(err)
		}

		// 转换为 TypedChunk 接口切片
		existingTypedInterface := make([]vo.TypedChunk, len(existingTypedChunks))
		for i, chunk := range existingTypedChunks {
			existingTypedInterface[i] = chunk
		}

		graphEmpty := buildResult.GraphPersistenceOutcome == enum.GraphPersistenceOutcomeEmpty
		reuseSuccessfulTyped := resumeCommittedGraph &&
			buildResult.TypedIndexOutcome == enum.ComponentOutcomeSuccess &&
			len(existingTypedChunks) > 0
		reuseEmptyTyped := resumeCommittedGraph &&
			graphEmpty &&
			buildResult.TypedIndexOutcome == enum.ComponentOutcomeNotApplicable &&
			len(existingTypedChunks) == 0

		if reuseSuccessfulTyped {
			typedChunks = existingTypedInterface
			typedOutcome = enum.ComponentOutcomeSuccess
		} else if reuseEmptyTyped {
			typedOutcome = enum.ComponentOutcomeNotApplicable
		} else {
			// 执行类型化索引替换
			if err := d.repo.UpdateTaskById(ctx, &entity.DocumentTask{
				ID:           taskId,
				CurrentStage: enum.TaskStageGraphTypedIndex,
			}); err != nil {
				panic(err)
			}

			replaced, err := d.graphRagBuilder.ReplaceTypedIndex(ctx, documentId, taskId, planId, sourceChunks, d.nextChunkNo(sourceChunks))
			if err != nil {
				logx.Warnf("GraphRAG typed projection failed; preserving committed KG: documentId=%d, taskId=%d, message=%v",
					documentId, taskId, err)
				typedChunks = []vo.TypedChunk{}
				typedOutcome = enum.ComponentOutcomeFailed
			} else {
				if replaced == nil {
					typedChunks = []vo.TypedChunk{}
				} else {
					typedChunks = make([]vo.TypedChunk, len(replaced))
					for i, chunk := range replaced {
						typedChunks[i] = chunk
					}
				}
				if graphEmpty && len(typedChunks) == 0 {
					typedOutcome = enum.ComponentOutcomeNotApplicable
				} else if len(typedChunks) == 0 {
					typedOutcome = enum.ComponentOutcomeFailed
				} else {
					typedOutcome = enum.ComponentOutcomeSuccess
				}
			}
		}
	}

	// 计算候选最终结果
	candidate := d.graphRagOutcomePolicy.FinalizeOuterDisposition(buildResult, typedOutcome, enum.ObservationProjectionOutcomeSuccess)
	if candidate.OuterTaskDisposition == enum.OuterTaskDispositionRepairRequired {
		candidate = d.withdrawPendingCrossDocumentProjection(ctx, document, taskId, candidate)
	}

	// 标记检查点
	if err := d.graphRagBuildCheckpoint.MarkOutcome(ctx, documentId, taskId, candidate, d.resultAttempt(candidate), d.resultMaxAttempts(candidate)); err != nil {
		logx.Warnf("GraphRAG final outcome projection failed; BUILD_INDEX remains repairable: documentId=%d, taskId=%d, message=%v",
			documentId, taskId, err)
		failedObservation := d.graphRagOutcomePolicy.FinalizeOuterDisposition(buildResult, typedOutcome, enum.ObservationProjectionOutcomeFailed)
		failedObservation = d.withdrawPendingCrossDocumentProjection(ctx, document, taskId, failedObservation)
		return &vo.GraphRagFinalization{Result: failedObservation, TypedChunks: typedChunks}
	}

	return &vo.GraphRagFinalization{Result: candidate, TypedChunks: typedChunks}
}

// repairCrossDocumentProjection 修复跨文档投影
func (d *AsyncProcessImpl) repairCrossDocumentProjection(ctx context.Context, document *entity.Document,
	documentId, taskId int64, buildResult *vo.GraphRagBuildResult) *vo.GraphRagBuildResult {
	alreadyActive := document != nil && document.LastIndexTaskId == taskId
	if alreadyActive && buildResult.CrossDocumentIndexOutcome == enum.ComponentOutcomeSuccess {
		return buildResult
	}
	if err := d.crossDocumentIndexer.RebuildAll(ctx, documentId, taskId); err != nil {
		logx.Warnf("GraphRAG cross-document repair failed: documentId=%d, taskId=%d, message=%v", documentId, taskId, err)
		return d.graphRagOutcomePolicy.WithCrossDocumentOutcome(buildResult, enum.ComponentOutcomeFailed)
	}
	return d.graphRagOutcomePolicy.WithCrossDocumentOutcome(buildResult, enum.ComponentOutcomeSuccess)
}

// withdrawPendingCrossDocumentProjection 撤回待处理的跨文档投影
func (d *AsyncProcessImpl) withdrawPendingCrossDocumentProjection(ctx context.Context, document *entity.Document,
	taskId int64, buildResult *vo.GraphRagBuildResult) *vo.GraphRagBuildResult {
	if buildResult == nil ||
		buildResult.CrossDocumentIndexOutcome != enum.ComponentOutcomeSuccess ||
		document == nil ||
		document.LastIndexTaskId == taskId {
		return buildResult
	}
	if err := d.crossDocumentIndexer.RebuildAll(ctx, 0, 0); err != nil {
		logx.Errorf("GraphRAG pending cross-document projection withdrawal failed; task cannot publish current I: documentId=%d, taskId=%d, message=%v",
			document.ID, taskId, err)
	} else {
		logx.Infof("GraphRAG pending cross-document projection withdrawn to active document pointers: documentId=%d, taskId=%d",
			document.ID, taskId)
	}
	return d.graphRagOutcomePolicy.WithCrossDocumentOutcome(buildResult, enum.ComponentOutcomeFailed)
}

// resultAttempt 获取尝试次数
func (d *AsyncProcessImpl) resultAttempt(result *vo.GraphRagBuildResult) int {
	if result == nil {
		return 0
	}
	return max(0, result.Attempt)
}

// resultMaxAttempts 获取最大尝试次数
func (d *AsyncProcessImpl) resultMaxAttempts(result *vo.GraphRagBuildResult) int {
	if result == nil {
		return 1
	}
	return max(1, result.MaxAttempts)
}

// nextChunkNo 获取下一个块编号
func (d *AsyncProcessImpl) nextChunkNo(chunks []*entity.DocumentChunk) int {
	if len(chunks) == 0 {
		return 1
	}
	maxNo := 0
	for _, chunk := range chunks {
		if chunk != nil && chunk.ChunkNo > maxNo {
			maxNo = chunk.ChunkNo
		}
	}
	return maxNo + 1
}
