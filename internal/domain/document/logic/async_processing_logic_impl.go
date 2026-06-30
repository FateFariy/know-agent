package logic

import (
	"context"
	"encoding/json"
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
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

const (
	embeddingBatch  = 100 // 默认向量化批大小
	defaultLogLevel = "PG_VECTOR"
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
}

// NewAsyncProcessingLogic 构造异步处理逻辑实例
func NewAsyncProcessingLogic(repo adapter.DocumentRepository, port *adapter.DocumentPort,
	registry parse.Registry, strategyLogic StrategyLogic, structureNode StructureNodeLogic) *AsyncProcessingLogicImpl {
	return &AsyncProcessingLogicImpl{
		repo:          repo,
		port:          port,
		registry:      registry,
		strategyLogic: strategyLogic,
		structureNode: structureNode,
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
func (d *AsyncProcessingLogicImpl) HandleParseRoute(ctx context.Context, documentId, taskId int64) error {
	// 加载文档与任务
	document, err := d.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		Warnf("查询解析文档失败: documentId=%d, err=%v", documentId, err)
		return err
	}

	// 加载任务
	task, err := d.repo.SelectTaskById(ctx, taskId)
	if err != nil {
		Warnf("查询解析任务失败: taskId=%d, err=%v", taskId, err)
		return err
	}

	startTime := time.Now()
	defer func() {
		if v := recover(); v != nil {
			if panicErr, ok := v.(error); ok {
				d.handleParseFailure(ctx, document, task, panicErr.Error())
			}
		}
	}()

	fn := func(txCtx context.Context) error {
		documentTask := &entity.DocumentTask{
			ID:           taskId,
			TaskStatus:   vo.TaskStatusRunning,
			CurrentStage: vo.TaskStageContentParse,
			StartTime:    startTime,
		}
		if err = d.repo.UpdateTaskById(txCtx, documentTask); err != nil {
			Warnf("更新任务失败: taskId=%d, err=%v", taskId, err)
			return err
		}
		if err = d.repo.UpdateDocument(txCtx, &entity.Document{ID: documentId, ParseStatus: vo.ParseStatusParsing}); err != nil {
			Warnf("更新文档失败: documentId=%d, err=%v", documentId, err)
			return err
		}
		detail, _ := json.Marshal(map[string]any{"objectName": document.ObjectName})
		startLog := &entity.DocumentTaskLog{
			TaskId:       taskId,
			DocumentId:   documentId,
			StageType:    vo.TaskStageContentParse,
			EventType:    vo.TaskEventStart,
			LogLevel:     vo.LogLevelInfo,
			OperatorType: vo.OperatorTypeSystem,
			Content:      "开始解析文档内容",
			DetailJson:   string(detail),
		}
		return d.repo.InsertTaskLog(txCtx, startLog)
	}

	// 下载原始文件并解析
	fileBytes, err := d.port.DownloadObject(ctx, document.ObjectName)
	if err != nil {
		panic(err)
	}

	parsedText, err := d.parse(ctx, fileBytes, vo.FileTypeName(document.FileType))
	if err != nil {
		panic(err)
	}

	// 上传解析文本（供索引构建阶段直接下载）
	parseTextPath, err := d.port.UploadParsedText(ctx, documentId, parsedText)
	if err != nil {
		panic(err)
	}

	// 结构节点与文档属性（当前阶段不依赖，先跳过图谱画像）
	nodeCandidates := d.buildStructureNodeCandidates(parsedText)
	_, err = d.structureNode.ReplaceDocumentNodes(ctx, documentId, taskId, nodeCandidates)
	if err != nil {
		// 结构节点失败不阻塞主流程，降级为忽略结构线索
		Warnf("替换文档结构节点失败: documentId=%d, err=%v", documentId, err)
	}
	structureNodes, _ := d.structureNode.ListDocumentNodes(ctx, documentId, taskId)

	// 构建文档分析结果并生成策略推荐
	analysisResult := &vo.DocumentAnalysisResult{
		ParsedText:          parsedText,
		CharCount:           len(parsedText),
		TokenCount:          len(parsedText) / tokenRatio,
		StructureLevel:      vo.StructureLevelLow,
		ContentQualityLevel: vo.ContentQualityLevelMedium,
		ParagraphCount:      countParagraphs(parsedText),
		StructureNodes:      structureNodes,
	}

	planDraft, err := d.strategyLogic.RecommendStrategy(ctx, document, analysisResult)
	if err != nil {
		Warnf("策略推荐失败: documentId=%d, err=%v", documentId, err)
		d.handleParseFailure(ctx, document, task, err.Error())
		return err
	}

	// 持久化方案和步骤
	planId, err := d.persistRecommendation(ctx, document, task, planDraft)
	if err != nil {
		Warnf("持久化推荐方案失败: documentId=%d, err=%v", documentId, err)
		d.handleParseFailure(ctx, document, task, err.Error())
		return err
	}

	// 写回文档统计 + 标记解析成功
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
	document.StructureNodeCount = len(structureNodes)

	if err = d.repo.UpdateDocument(ctx, document); err != nil {
		Warnf("更新文档信息失败: documentId=%d, err=%v", documentId, err)
		return err
	}

	// 记录"文档解析完成" / "策略推荐完成"日志
	d.saveTaskLog(ctx, taskId, documentId, vo.TaskStageContentParse, vo.TaskEventComplete, vo.LogLevelInfo,
		0, "", "文档解析完成", map[string]any{
			"charCount":           analysisResult.CharCount,
			"tokenCount":          analysisResult.TokenCount,
			"structureLevel":      analysisResult.StructureLevel,
			"contentQualityLevel": analysisResult.ContentQualityLevel,
			"structureNodeCount":  len(structureNodes),
			"paragraphCount":      analysisResult.ParagraphCount,
		})

	d.saveTaskLog(ctx, taskId, documentId, vo.TaskStageStrategyRoute, vo.TaskEventRecommendStrategy, vo.LogLevelInfo,
		0, "", "系统已生成推荐策略", map[string]any{
			"planId":           planId,
			"strategySnapshot": planDraft.StrategySnapshot,
			"parentStepCount":  len(planDraft.ParentSteps),
			"childStepCount":   len(planDraft.ChildSteps),
		})

	// 收尾
	d.finishTaskSuccess(ctx, task, vo.TaskStageStrategyRoute, startTime)
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

// parse 根据文件类型查找解析器并解析原始字节内容为纯文本
func (d *AsyncProcessingLogicImpl) parse(ctx context.Context, bytes []byte, fileType string) (string, error) {
	if parser := d.registry.Get(fileType); parser != nil {
		return parser.Parse(ctx, bytes)
	}
	return "", errorx.ErrUnsupportedFileType
}

// persistRecommendation 保存策略推荐结果（方案 + 步骤 + 任务日志），返回方案 ID
func (d *AsyncProcessingLogicImpl) persistRecommendation(ctx context.Context,
	document *entity.Document, task *entity.DocumentTask,
	planDraft *vo.DocumentStrategyPlanDraft) (int64, error) {

	planId := utils.GetSnowflakeNextID()
	latestVersion, err := d.repo.SelectLatestPlanVersion(ctx, document.ID)
	if err != nil {
		latestVersion = 0
	}

	plan := &entity.DocumentStrategyPlan{
		ID:               planId,
		DocumentId:       document.ID,
		PlanVersion:      latestVersion + 1,
		PlanSource:       vo.PlanSourceSystemRecommend,
		PlanStatus:       vo.PlanStatusWaitConfirm,
		StrategyCount:    len(planDraft.ParentSteps) + len(planDraft.ChildSteps),
		StrategySnapshot: planDraft.StrategySnapshot,
		RecommendReason:  planDraft.RecommendReason,
	}
	if err := d.repo.InsertPlan(ctx, plan); err != nil {
		return 0, err
	}

	insertSteps := func(drafts []*vo.DocumentStrategyStepDraft, pipelineType string) {
		for i, draft := range drafts {
			step := &entity.DocumentStrategyStep{
				ID:              utils.GetSnowflakeNextID(),
				PlanId:          planId,
				DocumentId:      document.ID,
				PipelineType:    pipelineType,
				StepNo:          i + 1,
				StrategyType:    draft.StrategyType,
				StrategyRole:    draft.StrategyRole,
				SourceType:      draft.SourceType,
				ExecuteStatus:   vo.StrategyExecuteStatusWaitExecute,
				RecommendReason: draft.RecommendReason,
			}
			if err := d.repo.InsertStep(ctx, step); err != nil {
				Warnf("插入步骤失败: planId=%d, pipelineType=%s, err=%v", planId, pipelineType, err)
			}
		}
	}

	insertSteps(planDraft.ParentSteps, vo.PipelineTypeParent)
	insertSteps(planDraft.ChildSteps, vo.PipelineTypeChild)

	// 推进任务阶段到"策略路由"
	task.CurrentStage = vo.TaskStageStrategyRoute
	if err := d.repo.UpdateTaskById(ctx, task); err != nil {
		Warnf("更新任务阶段失败: taskId=%d, err=%v", task.ID, err)
	}

	return planId, nil
}

// buildStructureNodeCandidates 基于解析文本构建结构节点候选
// 当前使用极简实现：对"标题化"文本行进行切分，交由结构节点服务处理。
// 更精确的结构解析由上层 parse 注册表提供，这里仅兜底。
func (d *AsyncProcessingLogicImpl) buildStructureNodeCandidates(parsedText string) []*vo.DocumentStructureNodeCandidate {
	if strings.TrimSpace(parsedText) == "" {
		return nil
	}
	// 暂时不强制做结构化拆分——由切块阶段在 Strategy 内部自行决定结构，
	// 上游"推荐策略"会在缺少结构节点时自动降级为递归/语义切块。
	return nil
}

// buildParentChildEntities 将父块候选 + 子块候选转换为可落库的实体列表
func (d *AsyncProcessingLogicImpl) buildParentChildEntities(documentId, taskId, planId int64,
	candidates []*vo.ParentBlockCandidate) ([]*entity.DocumentParentBlock, []*entity.DocumentChunk) {

	parentBlocks := make([]*entity.DocumentParentBlock, 0, len(candidates))
	chunks := make([]*entity.DocumentChunk, 0)

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
					VectorStoreType:   vo.VectorStoreTypeMilvus,
				})
				parentBlock.ChildCount++
			}
		}
		parentBlock.EndChunkNo = globalChunkNo - 1
		parentBlocks = append(parentBlocks, parentBlock)
	}
	return parentBlocks, chunks
}

// finishTaskSuccess 标记任务执行完成并记录结束日志
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

// handleParseFailure "解析路由"失败时的收尾流程：标记文档解析失败 + 任务失败 + 失败日志
func (d *AsyncProcessingLogicImpl) handleParseFailure(ctx context.Context, document *entity.Document,
	task *entity.DocumentTask, errorMsg string) {

	document.ParseStatus = vo.ParseStatusParseFailed
	document.ParseErrorMsg = errorMsg
	if err := d.repo.UpdateDocument(ctx, document); err != nil {
		Warnf("更新文档解析失败状态失败: documentId=%d, err=%v", document.ID, err)
	}

	task.TaskStatus = vo.TaskStatusFailed
	task.CurrentStage = vo.TaskStageContentParse

	failDetail, _ := json.Marshal(map[string]any{"error": errorMsg})
	failLog := &entity.DocumentTaskLog{
		TaskId:       task.ID,
		DocumentId:   task.DocumentId,
		StageType:    vo.TaskStageContentParse,
		EventType:    vo.TaskEventFailed,
		LogLevel:     vo.LogLevelError,
		OperatorType: vo.OperatorTypeSystem,
		Content:      "任务执行失败",
		DetailJson:   string(failDetail),
	}
	if err := d.repo.UpdateDocumentAggregate(ctx, &aggregate.Document{
		Document: document,
		Task:     task,
		TaskLog:  failLog,
	}); err != nil {
		Warnf("保存解析失败日志失败: taskId=%d, err=%v", task.ID, err)
	}
}

// handleIndexBuildFailure "索引构建"失败时的收尾流程
func (d *AsyncProcessingLogicImpl) handleIndexBuildFailure(ctx context.Context, document *entity.Document, task *entity.DocumentTask, plan *entity.DocumentStrategyPlan, errorMsg string) {
	logx.Error("索引构建失败: documentId=%d, taskId=%d, planId=%d, err=%v", document.ID, task.ID, plan.ID, errorMsg)
	fn := func(txCtx context.Context) error {
		if err := d.repo.UpdateDocument(txCtx, &entity.Document{ID: document.ID, IndexStatus: vo.IndexStatusBuildFailed}); err != nil {
			return err
		}
		chunk := &entity.DocumentChunk{TaskId: task.ID, VectorStatus: vo.VectorStatusVectorFailed, VectorStoreType: vo.VectorStoreTypeMilvus}
		if err := d.repo.UpdateChunkByTaskId(txCtx, chunk); err != nil {
			return err
		}
		if err := d.repo.UpdateStepExecuteStatus(txCtx, plan.ID, vo.StrategyExecuteStatusExecuteFailed); err != nil {
			return err
		}
		if err := d.failTask(txCtx, task, errorMsg); err != nil {
			return err
		}
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
	if err := d.repo.Do(ctx, fn); err != nil {
		Warnf("保存索引构建失败日志失败: taskId=%d, err=%v", task.ID, err)
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
	var chineseCount int
	var englishCount int

	// 统计中文字符数量
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		}
	}

	// 统计英文单词数量：按空白分割，单词包含至少一个英文字母则计数+1
	words := strings.Fields(text)
	for _, word := range words {
		for _, r := range word {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				englishCount++
				break
			}
		}
	}

	// 非中英文字符按 4 字符折算 1 Token
	baseToken := max(1, (utf8.RuneCountInString(text)-chineseCount-englishCount)/4)

	return chineseCount + englishCount + baseToken
}

//
// // countParagraphs 按空行粗略估算段落数（用于"内容质量"判断的辅助信号）
// func countParagraphs(text string) int {
// 	if strings.TrimSpace(text) == "" {
// 		return 0
// 	}
// 	lines := strings.Split(text, "\n")
// 	count := 0
// 	inParagraph := false
// 	for _, line := range lines {
// 		trimmed := strings.TrimSpace(line)
// 		if trimmed == "" {
// 			inParagraph = false
// 			continue
// 		}
// 		if !inParagraph {
// 			count++
// 			inParagraph = true
// 		}
// 	}
// 	return count
// }

// // Warnf 统一的告警日志入口
// func Warnf(format string, args ...any) {
// 	logx.Alert(fmt.Sprintf(format, args...))
// }
