package process

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/analysis"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/index"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// AsyncProcessImpl 异步处理业务逻辑实现
//
//	HandleParseRoute → 解析路由（文件解析 + 结构节点 + 策略推荐）
//	HandleIndexBuild → 索引构建（切块流水线 + 向量化 + 落库 + GraphRAG + RAPTOR）
type AsyncProcessImpl struct {
	repo          adapter.DocumentRepository
	analysisChain *analysis.Chain
	indexChain    *index.BuildIndexChain
}

// NewAsyncProcessImpl 构造异步处理逻辑实例
func NewAsyncProcessImpl(repo adapter.DocumentRepository, analysisChain *analysis.Chain, indexChain *index.BuildIndexChain) *AsyncProcessImpl {
	return &AsyncProcessImpl{
		repo:          repo,
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
	//graphRagBuildResult := task.ReadGraphRagBuildResult()

	//// 检查是否需要直接失败
	//if graphRagBuildResult != nil && graphRagBuildResult.OuterTaskDisposition == enum.OuterTaskDispositionFailIndexTask {
	//	d.applyGraphFailureDisposition(ctx, document, task, plan.ID, graphRagBuildResult, nil)
	//	return nil // 直接失败，无需继续执行
	//}
	//
	//defer func() {
	//	if v := recover(); v != nil {
	//		var panicErr *vo.GraphRagBuildFailureException
	//		if errors.As(v, &panicErr) {
	//			d.handleGraphRagBuildFailure(ctx, document, task, plan, panicErr, startTime)
	//		}
	//	}
	//}()

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
