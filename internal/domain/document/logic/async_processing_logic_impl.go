package logic

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/parse"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

const (
	embeddingBatch = 100 // 默认向量化批大小
)

var (
	englishPattern = regexp.MustCompile(`[A-Za-z]`) // 匹配英文字母
)

// AsyncProcessingLogicImpl 异步处理业务逻辑实现
//
//	HandleParseRoute → 解析路由（文件解析 + 结构节点 + 策略推荐）
//	HandleIndexBuild → 索引构建（切块流水线 + 向量化 + 落库）
type AsyncProcessingLogicImpl struct {
	repo          adapter.DocumentRepository
	port          *adapter.DocumentPort
	registry      parse.Registry
	strategyLogic StrategyLogic
	structureNode StructureNodeLogic
	textPreProc   TextPreProcessLogic
}

// NewAsyncProcessingLogicImpl 构造异步处理逻辑实例
func NewAsyncProcessingLogicImpl(repo adapter.DocumentRepository, port *adapter.DocumentPort, registry parse.Registry,
	strategyLogic StrategyLogic, structureNode StructureNodeLogic, textPreProc TextPreProcessLogic) *AsyncProcessingLogicImpl {
	return &AsyncProcessingLogicImpl{
		repo:          repo,
		port:          port,
		registry:      registry,
		strategyLogic: strategyLogic,
		structureNode: structureNode,
		textPreProc:   textPreProc,
	}
}

// HandleParseRoute 处理解析路由任务
//
// 整体阶段：
//  1. 读取文档/任务，将任务标记为 RUNNING，当前阶段推进到 CONTENT_PARSE
//  2. 从对象存储下载原始文件并调用解析器提取纯文本
//  3. 将解析后的纯文本重新上传为 txt，便于后续索引构建直接复用
//  4. 用结构节点服务替换文档结构节点（便于结构切块策略依赖）
//  5. 基于解析结果调用策略服务生成推荐切块方案
//  6. 把推荐方案和步骤写入数据库，同步更新文档的解析状态/策略状态/统计信息
//  7. 以成功或失败状态收尾任务，并记录任务日志
func (d *AsyncProcessingLogicImpl) HandleParseRoute(ctx context.Context, documentId, taskId int64) (err error) {
	// 加载文档与任务实体
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

	// 记录开始时间并注册 panic recover → 失败时统一调用 handleParseFailure
	startTime := time.Now()
	defer func() {
		if v := recover(); v != nil {
			if panicErr, ok := v.(error); ok {
				d.handleParseFailure(ctx, document, task, panicErr.Error())
			}
		}
	}()

	// 事务性标记任务运行中 + 文档解析中，并写入"开始解析"日志
	markParseStartTx := func(txCtx context.Context) error {
		runningTask := &entity.DocumentTask{
			ID:           taskId,
			TaskStatus:   vo.TaskStatusRunning,
			CurrentStage: vo.TaskStageContentParse,
			StartTime:    startTime,
		}
		if err = d.repo.UpdateTaskById(txCtx, runningTask); err != nil {
			return err
		}
		if err = d.repo.UpdateDocument(txCtx, &entity.Document{ID: documentId, ParseStatus: vo.ParseStatusParsing}); err != nil {
			return err
		}
		// 写入"开始解析文档"日志，附带对象存储 key
		startDetail, _ := json.Marshal(map[string]any{"objectName": document.ObjectName})
		startLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    vo.TaskStageContentParse,
			EventType:    vo.TaskEventStart,
			LogLevel:     vo.LogLevelInfo,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "开始解析文档内容",
			DetailJson:   string(startDetail),
		}
		return d.repo.InsertTaskLog(txCtx, startLog)
	}
	if err = d.repo.Do(ctx, markParseStartTx); err != nil {
		panic(err)
	}

	// 从对象存储下载原始文件字节
	rawFileBytes, err := d.port.DownloadObject(ctx, document.ObjectName)
	if err != nil {
		panic(err)
	}
	// 调用文本预处理逻辑
	analysisResult, err := d.textPreProc.PreProcess(ctx, document.OriginalFileName, string(rawFileBytes), vo.FileTypeName(document.FileType))
	if err != nil {
		panic(err)
	}

	// 上传解析后的纯文本到对象存储，供"索引构建阶段"直接下载复用
	parsedTextPath, err := d.port.UploadParsedText(ctx, documentId, analysisResult.ParsedText)
	if err != nil {
		panic(err)
	}

	// 基于解析文本构建并写入文档结构节点（供结构切块策略使用）
	structureNodes, err := d.structureNode.ReplaceDocumentNodes(ctx, documentId, taskId, analysisResult.StructureNodes)
	if err != nil {
		panic(err)
	}

	if err = d.syncNavigationArtifacts(ctx, documentId, taskId, structureNodes); err != nil {
		panic(err)
	}

	// todo documentProfileService.generateProfile(documentId, analysisResult, structureNodes);

	// 写入"文档解析完成"日志（附带字符数/段落/结构节点数量等统计信息）
	parseFinishDetail, _ := json.Marshal(map[string]any{
		"charCount":           analysisResult.CharCount,
		"tokenCount":          analysisResult.TokenCount,
		"structureLevel":      analysisResult.StructureLevel,
		"contentQualityLevel": analysisResult.ContentQualityLevel,
		"structureNodeCount":  len(structureNodes),
		"paragraphCount":      analysisResult.ParagraphCount,
	})
	parseFinishLog := &entity.DocumentTaskLog{
		TaskId:       taskId,
		DocumentId:   documentId,
		StageType:    vo.TaskStageContentParse,
		EventType:    vo.TaskEventComplete,
		LogLevel:     vo.LogLevelInfo,
		OperatorType: vo.OperatorTypeSystem,
		Content:      "文档解析完成",
		DetailJson:   string(parseFinishDetail),
	}
	if err = d.repo.InsertTaskLog(ctx, parseFinishLog); err != nil {
		panic(err)
	}

	// 调用策略服务生成推荐切块方案草稿
	strategyPlanDraft, err := d.strategyLogic.RecommendStrategy(ctx, document, analysisResult)
	if err != nil {
		panic(err)
	}

	// 事务性持久化策略方案 → 回写文档统计/状态 → 收尾任务 → 写入"生成推荐策略"日志
	persistStrategyTx := func(txCtx context.Context) error {
		// 持久化方案和步骤（写入 document_plan 与 document_strategy_step）
		var planId int64
		planId, err = d.persistRecommendation(txCtx, document, task, strategyPlanDraft)
		if err != nil {
			return err
		}

		// 写回文档统计 + 标记解析成功/策略已推荐
		updatedDoc := &entity.Document{
			ID:                  documentId,
			ParseStatus:         vo.ParseStatusParseSuccess,
			StrategyStatus:      vo.StrategyStatusRecommended,
			CharCount:           analysisResult.CharCount,
			TokenCount:          analysisResult.TokenCount,
			StructureLevel:      analysisResult.StructureLevel,
			ContentQualityLevel: analysisResult.ContentQualityLevel,
			ParseTextPath:       parsedTextPath,
			ParseErrorMsg:       utils.Pointer(""),
			CurrentPlanId:       planId,
			LastParseTaskId:     taskId,
			StructureNodeCount:  len(structureNodes),
		}
		if err = d.repo.UpdateDocument(txCtx, updatedDoc); err != nil {
			return err
		}
		// 标记任务成功完成并收尾（写入耗时等）
		if err = d.finishTaskSuccess(txCtx, task, vo.TaskStageStrategyRoute, startTime); err != nil {
			return err
		}

		// 记录"系统已生成推荐策略"日志
		recommendDetail, _ := json.Marshal(map[string]any{
			"planId":             planId,
			"strategySnapshot":   strategyPlanDraft.StrategySnapshot,
			"parentStepCount":    len(strategyPlanDraft.ParentSteps),
			"childStepCount":     len(strategyPlanDraft.ChildSteps),
			"structureNodeCount": len(structureNodes),
			"recommendReason":    strategyPlanDraft.RecommendReason,
		})
		recommendLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    vo.TaskStageContentParse,
			EventType:    vo.TaskEventComplete,
			LogLevel:     vo.LogLevelInfo,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "系统已生成推荐策略",
			DetailJson:   string(recommendDetail),
		}
		return d.repo.InsertTaskLog(txCtx, recommendLog)
	}
	if err = d.repo.Do(ctx, persistStrategyTx); err != nil {
		panic(err)
	}
	return nil
}

// HandleIndexBuild 执行索引构建主流程：切块流水线 → 父子块落库 → 向量化 → 状态收尾
func (d *AsyncProcessingLogicImpl) HandleIndexBuild(ctx context.Context, documentId, taskId, planId int64) (err error) {
	// 加载任务相关实体，失败直接返回，交由调度层观察
	task, err := d.repo.SelectTaskById(ctx, taskId)
	if err != nil {
		Warnf("查询索引任务失败, taskId=%d, err=%v", taskId, err)
		return err
	}
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		Warnf("查询索引任务文档失败, documentId=%d, err=%v", documentId, err)
		return err
	}
	plan, err := d.repo.SelectPlanById(ctx, planId)
	if err != nil {
		Warnf("查询索引任务方案失败, planId=%d, err=%v", planId, err)
		return err
	}
	pipelineSteps, err := d.repo.SelectStepListByPlanId(ctx, planId)
	if err != nil {
		Warnf("查询索引任务步骤失败, planId=%d, err=%v", planId, err)
		return err
	}

	// 记录起始时间用于耗时统计；defer recover 统一捕获 panic 为失败状态
	startTime := time.Now()
	defer func() {
		if v := recover(); v != nil {
			if panicErr, ok := v.(error); ok {
				d.handleIndexBuildFailure(ctx, document, task, plan, panicErr.Error())
			}
		}
	}()

	// 事务性推进任务状态到"切块执行中"
	markBuildingTx := func(txCtx context.Context) error {
		// 文档状态
		if err = d.repo.UpdateDocument(txCtx, &entity.Document{ID: document.ID, IndexStatus: vo.IndexStatusBuilding}); err != nil {
			return err
		}
		// 策略步骤标记执行中
		if err = d.repo.UpdateStepExecuteStatus(txCtx, plan.ID, vo.StrategyExecuteStatusExecuting); err != nil {
			return err
		}
		// 记录开始执行切块日志
		chunkStartDetail, _ := json.Marshal(map[string]any{"strategySnapshot": plan.StrategySnapshot})
		chunkStartLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    vo.TaskStageChunkExecute,
			EventType:    vo.TaskEventStart,
			LogLevel:     vo.LogLevelInfo,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "开始执行切块流水线",
			DetailJson:   string(chunkStartDetail),
		}
		if err = d.repo.InsertTaskLog(txCtx, chunkStartLog); err != nil {
			return err
		}
		// 推进任务阶段为"切块执行中"
		return d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID:           taskId,
			TaskStatus:   vo.TaskStatusRunning,
			CurrentStage: vo.TaskStageChunkExecute,
			StartTime:    time.Now(),
		})
	}
	if err = d.repo.Do(ctx, markBuildingTx); err != nil {
		panic(err)
	}

	// 下载解析文本（已在解析路由阶段上传）
	parsedText, err := d.port.DownloadText(ctx, document.ParseTextPath)
	if err != nil {
		panic(err)
	}

	// 按步骤执行切块流水线，产出父-子块候选
	parentCandidates, err := d.strategyLogic.BuildParentBlocks(ctx, document, pipelineSteps, parsedText)
	if err != nil {
		panic(err)
	}

	// 事务性标记切块完成 + 推进到切块后处理阶段
	markChunkCompleteTx := func(txCtx context.Context) error {
		// 策略步骤状态 -> 执行成功
		if err = d.repo.UpdateStepExecuteStatus(txCtx, plan.ID, vo.StrategyExecuteStatusExecuteSuccess); err != nil {
			return err
		}
		chunkEndDetail, _ := json.Marshal(map[string]any{
			"parentCount": len(parentCandidates),
			"childCount":  d.countChildCandidates(parentCandidates),
		})
		chunkEndLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    vo.TaskStageChunkExecute,
			EventType:    vo.TaskEventComplete,
			LogLevel:     vo.LogLevelInfo,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "切块执行完成",
			DetailJson:   string(chunkEndDetail),
		}
		if err = d.repo.InsertTaskLog(txCtx, chunkEndLog); err != nil {
			return err
		}
		// 推进任务阶段到"切块后处理"
		return d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: taskId, CurrentStage: vo.TaskStageChunkPostProcess})
	}
	if err = d.repo.Do(ctx, markChunkCompleteTx); err != nil {
		panic(err)
	}

	// 清理候选并构造持久化实体（父块 + 子块）
	finalCandidates := d.cleanupParentCandidates(parentCandidates)
	parentBlocks, childChunks := d.buildParentChildEntities(documentId, taskId, planId, finalCandidates)

	// 事务性批量落库 + 推进到向量化阶段
	persistBlocksTx := func(txCtx context.Context) error {
		// 批量写入父块
		if err = d.repo.InsertParentBlockBatch(txCtx, parentBlocks); err != nil {
			Warnf("插入父块失败: documentId=%d, err=%v", documentId, err)
			return err
		}
		// 批量写入子块
		if err = d.repo.InsertChunkBatch(txCtx, childChunks); err != nil {
			Warnf("插入块失败: documentId=%d, err=%v", documentId, err)
			return err
		}
		// 记录"切块后处理完成"日志
		chunkPostDetail, _ := json.Marshal(map[string]any{
			"parentCount": len(finalCandidates),
			"childCount":  d.countChildCandidates(finalCandidates),
		})
		chunkPostLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    vo.TaskStageChunkPostProcess,
			EventType:    vo.TaskEventComplete,
			LogLevel:     vo.LogLevelInfo,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "切块后处理完成",
			DetailJson:   string(chunkPostDetail),
		}
		if err = d.repo.InsertTaskLog(txCtx, chunkPostLog); err != nil {
			return err
		}
		// 推进任务阶段到"向量化"
		return d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: taskId, CurrentStage: vo.TaskStageVectorize})
	}
	if err = d.repo.Do(ctx, persistBlocksTx); err != nil {
		panic(err)
	}

	// 记录"开始执行向量化"日志（单独事务，便于追踪状态）
	vectorSize := len(childChunks)
	vectorBatch := (vectorSize + embeddingBatch - 1) / embeddingBatch

	markVectorStartTx := func(txCtx context.Context) error {
		vectorStartDetail, _ := json.Marshal(map[string]any{
			"chunkCount":          vectorSize,
			"embeddingBatchSize":  embeddingBatch,
			"embeddingBatchCount": vectorBatch,
			"vectorStoreType":     vo.VectorStoreTypeMilvus,
			"parentCount":         len(parentBlocks),
		})
		vectorStartLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    vo.TaskStageVectorize,
			EventType:    vo.TaskEventStart,
			LogLevel:     vo.LogLevelInfo,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "开始执行向量化",
			DetailJson:   string(vectorStartDetail),
		}
		return d.repo.InsertTaskLog(txCtx, vectorStartLog)
	}
	if err = d.repo.Do(ctx, markVectorStartTx); err != nil {
		panic(err)
	}

	// 批量向量化
	if err = d.port.Vectorize(ctx, childChunks); err != nil {
		panic(err)
	}
	// 批量关键词索引
	if err = d.port.IndexChunks(ctx, childChunks); err != nil {
		panic(err)
	}
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
			"vectorStoreType":     vo.VectorStoreTypeMilvus,
			"parentCount":         len(parentBlocks),
		})
		vectorEndLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    vo.TaskStageVectorize,
			EventType:    vo.TaskEventComplete,
			LogLevel:     vo.LogLevelInfo,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "向量化完成",
			DetailJson:   string(vectorEndDetail),
		}
		return d.repo.InsertTaskLog(txCtx, vectorEndLog)
	}
	if err = d.repo.Do(ctx, markVectorCompleteTx); err != nil {
		panic(err)
	}

	// 事务性最终状态更新（任务/方案/文档），并写入索引构建完成日志
	finalizeTx := func(txCtx context.Context) error {
		// 任务阶段推进到"存储完成"
		if err = d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{ID: taskId, CurrentStage: vo.TaskStageStoreComplete}); err != nil {
			return err
		}
		// 方案状态标记为已执行
		if err = d.repo.UpdatePlanStatus(txCtx, planId, vo.PlanStatusExecuted); err != nil {
			return err
		}
		// 文档索引状态更新为构建成功
		if err = d.repo.UpdateDocument(txCtx, &entity.Document{ID: documentId, IndexStatus: vo.IndexStatusBuildSuccess, LastIndexTaskId: taskId}); err != nil {
			return err
		}
		// 写入成功耗时/统计日志
		if err = d.finishTaskSuccess(txCtx, task, vo.TaskStageStoreComplete, startTime); err != nil {
			panic(err)
		}
		// 索引构建完成日志
		buildCompleteDetail, _ := json.Marshal(map[string]any{
			"parentBlockCount": len(parentBlocks),
			"chunkCount":       len(childChunks),
		})
		buildCompleteLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    vo.TaskStageVectorize,
			EventType:    vo.TaskEventComplete,
			LogLevel:     vo.LogLevelInfo,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "索引构建完成",
			DetailJson:   string(buildCompleteDetail),
		}
		return d.repo.InsertTaskLog(txCtx, buildCompleteLog)
	}
	if err = d.repo.Do(ctx, finalizeTx); err != nil {
		panic(err)
	}
	return nil
}

// todo 待实现
func (d *AsyncProcessingLogicImpl) syncNavigationArtifacts(ctx context.Context, documentId, parseTaskId int64, structureNodes []*entity.DocumentStructureNode) error {
	return nil
}

// persistRecommendation 持久化策略推荐结果：写入方案 + 批量写入父/子流水线步骤 + 推进任务阶段
func (d *AsyncProcessingLogicImpl) persistRecommendation(ctx context.Context, document *entity.Document,
	task *entity.DocumentTask, planDraft *vo.DocumentStrategyPlanDraft) (int64, error) {
	// 分配计划 ID 并读取当前最新版本号（用于版本自增）
	planId := utils.GetSnowflakeNextID()
	latestVersion, err := d.repo.SelectLatestPlanVersion(ctx, document.ID)
	if err != nil {
		return 0, err
	}

	// 构造并插入计划主体
	strategyPlan := &entity.DocumentStrategyPlan{
		ID:               planId,
		DocumentId:       document.ID,
		PlanVersion:      latestVersion + 1,
		PlanSource:       vo.PlanSourceSystemRecommend,
		PlanStatus:       vo.PlanStatusWaitConfirm,
		StrategyCount:    len(planDraft.ParentSteps) + len(planDraft.ChildSteps),
		StrategySnapshot: planDraft.StrategySnapshot,
		RecommendReason:  planDraft.RecommendReason,
	}
	if err = d.repo.InsertPlan(ctx, strategyPlan); err != nil {
		return 0, err
	}

	// 按流水线类型将草稿步骤批量转成实体并落库
	insertPipelineSteps := func(drafts []*vo.DocumentStrategyStepDraft, pipelineType string) {
		steps := make([]*entity.DocumentStrategyStep, 0, len(drafts))
		for orderIdx, draft := range drafts {
			steps = append(steps, &entity.DocumentStrategyStep{
				ID:              utils.GetSnowflakeNextID(),
				PlanId:          planId,
				DocumentId:      document.ID,
				PipelineType:    pipelineType,
				StepNo:          orderIdx + 1,
				StrategyType:    draft.StrategyType,
				StrategyRole:    draft.StrategyRole,
				SourceType:      draft.SourceType,
				ExecuteStatus:   vo.StrategyExecuteStatusWaitExecute,
				RecommendReason: draft.RecommendReason,
			})
		}
		if err = d.repo.InsertStepBatch(ctx, steps); err != nil {
			Warnf("插入步骤失败: planId=%d, pipelineType=%s, err=%v", planId, pipelineType, err)
		}
	}

	// 顺序写入父块与子块流水线步骤
	insertPipelineSteps(planDraft.ParentSteps, vo.PipelineTypeParent)
	insertPipelineSteps(planDraft.ChildSteps, vo.PipelineTypeChild)

	// 推进任务阶段到"策略路由"
	task.CurrentStage = vo.TaskStageStrategyRoute
	if err = d.repo.UpdateTaskById(ctx, task); err != nil {
		return 0, err
	}

	return planId, nil
}

// buildParentChildEntities 将父块候选转换为可落库的"父块实体 + 子块实体"双列表
// 关键信息：
//   - 每个父块维护 StartChunkNo / EndChunkNo（用于快速定位其覆盖的子块区间）
//   - 子块的 ChunkNo 在函数内全局递增
//   - 任何父块至少会得到 1 个兜底子块（由上层 BuildParentBlocks 保证）
func (d *AsyncProcessingLogicImpl) buildParentChildEntities(documentId, taskId, planId int64,
	candidates []*vo.ParentBlockCandidate) ([]*entity.DocumentParentBlock, []*entity.DocumentChunk) {

	parentBlocks := make([]*entity.DocumentParentBlock, 0, len(candidates))
	chunks := make([]*entity.DocumentChunk, 0)

	// 全局子块编号：从 1 开始，遇到有效子块时递增并写入 ChunkNo
	globalChunkNo := 1
	for parentIdx, candidate := range candidates {
		parentBlock := &entity.DocumentParentBlock{
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
			CharCount:         utf8.RuneCountInString(candidate.Text),
			TokenCount:        d.estimateTokenCount(candidate.Text),
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
					ParentBlockId:     parentBlock.ID,
					ChunkNo:           globalChunkNo,
					SourceType:        child.SourceType,
					SectionPath:       utils.BlankToDefault(child.SectionPath, candidate.SectionPath),
					StructureNodeId:   child.StructureNodeId,
					StructureNodeType: child.StructureNodeType,
					CanonicalPath:     child.CanonicalPath,
					ItemIndex:         child.ItemIndex,
					ChunkText:         child.Text,
					CharCount:         utf8.RuneCountInString(child.Text),
					TokenCount:        d.estimateTokenCount(child.Text),
					VectorStatus:      vo.VectorStatusWaitVector,
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
func (d *AsyncProcessingLogicImpl) finishTaskSuccess(ctx context.Context, task *entity.DocumentTask, currentStage int, startTime time.Time) error {
	return d.repo.UpdateTaskById(ctx, &entity.DocumentTask{
		ID:           task.ID,
		TaskStatus:   vo.TaskStatusSuccess,
		CurrentStage: currentStage,
		FinishTime:   time.Now(),
		CostMillis:   int64(time.Since(startTime) / time.Millisecond),
		ErrorCode:    utils.Pointer(""),
		ErrorMsg:     utils.Pointer(""),
	})
}

// handleParseFailure 异步任务"解析路由"阶段失败时的统一收尾流程：先记录错误日志，再在事务内将文档状态、任务状态、失败详情与失败日志一次落库。
func (d *AsyncProcessingLogicImpl) handleParseFailure(ctx context.Context, document *entity.Document, task *entity.DocumentTask, errorMsg string) {
	logx.Errorf("异步解析文档失败，documentId=%d, taskId=%d, exception=%v", document.ID, task.ID, errorMsg)
	parseFailTx := func(txCtx context.Context) error {
		// 文档：标记为解析失败，并保留失败原因
		if err := d.repo.UpdateDocument(txCtx, &entity.Document{
			ID: document.ID, ParseStatus: vo.ParseStatusParseFailed, ParseErrorMsg: utils.Pointer(errorMsg),
		}); err != nil {
			return err
		}
		// 任务：标记为失败并停留在 CONTENT_PARSE
		if err := d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{
			ID: task.ID, TaskStatus: vo.TaskStatusFailed, CurrentStage: vo.TaskStageContentParse,
		}); err != nil {
			return err
		}
		// 通用失败收尾（写入耗时/错误码/错误消息）
		if err := d.failTask(txCtx, task, errorMsg); err != nil {
			return err
		}
		// 写入失败事件日志
		failDetail, _ := json.Marshal(map[string]any{"error": errorMsg})
		failLog := &entity.DocumentTaskLog{
			TaskId:       task.ID,
			DocumentId:   task.DocumentId,
			StageType:    vo.TaskStageContentParse,
			EventType:    vo.TaskEventFailed,
			LogLevel:     vo.LogLevelError,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "文档解析失败",
			DetailJson:   string(failDetail),
		}
		return d.repo.InsertTaskLog(txCtx, failLog)
	}
	if err := d.repo.Do(ctx, parseFailTx); err != nil {
		Warnf("解析失败时收尾失败: taskId=%d, err=%v", task.ID, err)
	}
}

// handleIndexBuildFailure "索引构建"失败时的统一收尾：事务性地将文档 IndexStatus、chunk 向量状态、step 执行状态、任务失败信息与日志一次落库。
func (d *AsyncProcessingLogicImpl) handleIndexBuildFailure(ctx context.Context, document *entity.Document, task *entity.DocumentTask, plan *entity.DocumentStrategyPlan, errorMsg string) {
	logx.Errorf("索引构建失败: documentId=%d, taskId=%d, planId=%d, err=%v", document.ID, task.ID, plan.ID, errorMsg)
	indexBuildFailTx := func(txCtx context.Context) error {
		// 文档：索引构建失败
		if err := d.repo.UpdateDocument(txCtx, &entity.Document{ID: document.ID, IndexStatus: vo.IndexStatusBuildFailed}); err != nil {
			return err
		}
		// chunk：按任务 ID 批量将向量状态置为失败（Milvus 为默认向量库类型）
		failedChunkMarker := &entity.DocumentChunk{
			TaskId:          task.ID,
			VectorStatus:    vo.VectorStatusVectorFailed,
			VectorStoreType: vo.VectorStoreTypeMilvus,
		}
		if err := d.repo.UpdateChunkByTaskId(txCtx, failedChunkMarker); err != nil {
			return err
		}
		// 标记当前计划所有步骤为失败
		if err := d.repo.UpdateStepExecuteStatus(txCtx, plan.ID, vo.StrategyExecuteStatusExecuteFailed); err != nil {
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
			EventType:    vo.TaskEventFailed,
			LogLevel:     vo.LogLevelError,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "索引构建失败",
			DetailJson:   string(failDetail),
		}
		return d.repo.InsertTaskLog(txCtx, failLog)
	}
	if err := d.repo.Do(ctx, indexBuildFailTx); err != nil {
		Warnf("索引构建失败时收尾失败: taskId=%d, err=%v", task.ID, err)
	}
}

// failTask 标记任务失败
func (d *AsyncProcessingLogicImpl) failTask(txCtx context.Context, task *entity.DocumentTask, errorMsg string) error {
	return d.repo.UpdateTaskById(txCtx, &entity.DocumentTask{
		ID:           task.ID,
		TaskStatus:   vo.TaskStatusFailed,
		CurrentStage: task.CurrentStage,
		FinishTime:   time.Now(),
		CostMillis:   int64(time.Since(task.StartTime) / time.Millisecond),
		ErrorCode:    utils.Pointer("TASK_FAILED"),
		ErrorMsg:     utils.Pointer(errorMsg),
	})
}

// countChildCandidates 计算子块候选数
func (d *AsyncProcessingLogicImpl) countChildCandidates(parentBlockCandidateList []*vo.ParentBlockCandidate) int {
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
func (d *AsyncProcessingLogicImpl) cleanupParentCandidates(candidates []*vo.ParentBlockCandidate) []*vo.ParentBlockCandidate {
	return slice.Filter(candidates, func(_ int, item *vo.ParentBlockCandidate) bool {
		fn := func(child *vo.ChunkCandidate) bool { return child != nil && strutil.IsNotBlank(child.Text) }
		return item != nil && strutil.IsNotBlank(item.Text) && slices.ContainsFunc(item.ChildChunks, fn)
	})
}

// estimateTokenCount 估算文本 Token 数量
func (d *AsyncProcessingLogicImpl) estimateTokenCount(text string) int {
	englishCount, chineseCount := 0, 0

	// 统计英文单词数量
	for _, word := range strings.Fields(text) {
		if englishPattern.MatchString(word) {
			englishCount++
		}
	}

	// 统计中文字符数量
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		}
	}

	// 非中英文字符按 4 字符折算 1 Token
	baseToken := max(1, (utf8.RuneCountInString(text)-chineseCount-englishCount)/4)

	return chineseCount + englishCount + baseToken
}

// Warnf 统一的告警日志入口
// func Warnf(format string, args ...any) {
// 	logx.Alert(fmt.Sprintf(format, args...))
// }
