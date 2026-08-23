package conversation

import (
	"context"
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// QueryRewriteStage 问题改写阶段
// 负责对所有非 OpenChat 模式的问题进行改写，生成检索友好的问题表达和子问题拆分。
// 仅当 RAG 开启时执行，OpenChat 模式跳过。
type QueryRewriteStage struct {
	rewriter        QueryRewriter
	enabled         bool
	ragEnabled      bool
	maxSubQuestions int
	temperature     float32
	topP            float32
	thinking        bool
}

var _ Stage = (*QueryRewriteStage)(nil)

func NewQueryRewriteStage(svcCtx *svc.ServiceContext, rewriter QueryRewriter) *QueryRewriteStage {
	return &QueryRewriteStage{
		rewriter:        rewriter,
		enabled:         svcCtx.Config.Chat.Rewrite.Enabled,
		ragEnabled:      svcCtx.Config.Chat.Rag.Enabled,
		maxSubQuestions: svcCtx.Config.Chat.Rewrite.MaxSubQuestions,
		temperature:     svcCtx.Config.Chat.Rewrite.Temperature,
		thinking:        svcCtx.Config.Chat.Rewrite.Thinking,
		topP:            svcCtx.Config.Chat.Rewrite.TopP,
	}
}

// Name 阶段名称
func (q *QueryRewriteStage) Name() string {
	return enum.ConversationTraceStageRewrite.Name
}

// Execute 执行问题改写
func (q *QueryRewriteStage) Execute(ctx context.Context, convCtx *Context) error {
	// OpenChat 模式不需要问题改写, 且未选择任何库时也不执行
	if convCtx.ChatMode == enum.ChatQueryModeOpenChat {
		return nil
	}

	// RAG 未启用时返回错误
	if !q.ragEnabled {
		return fmt.Errorf("当前文档问答模式未启用，请先开启聊天侧 RAG 编排")
	}

	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return nil
	}
	question := utils.Trim(convCtx.Question)
	historySummary := execPlan.HistorySummary

	// 启动改写追踪阶段
	input := &vo.StageInput{
		SummaryText: "正在生成检索友好的问题表达。",
		Snapshot:    q.buildRewriteStageSnapshot(question, historySummary, nil),
	}
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRewrite, enum.ExecutionModeRetrieval.String(), input)

	// 调用改写逻辑（原始问题 + 历史摘要 → 改写问题 + 子问题）
	rewriteResult, err := q.rewriter.Rewrite(ctx, question, historySummary)
	if err != nil {
		ctx = vo.OnError(ctx, "问题改写失败。", err)
		return err
	}

	// 更新执行计划
	execPlan.RewriteQuestion = utils.BlankToDefault(rewriteResult.RewrittenQuestion, question)
	execPlan.RewriteSubQuestions = rewriteResult.SubQuestions
	if len(rewriteResult.SubQuestions) == 0 {
		execPlan.RewriteSubQuestions = []string{rewriteResult.RewrittenQuestion}
	}
	convCtx.SetExecutePlan(execPlan)

	// 提交改写追踪（包含改写结果快照以便离线分析）
	output := &vo.StageOutput{
		SummaryText: "问题改写完成。",
		Snapshot:    q.buildRewriteStageSnapshot(question, historySummary, rewriteResult),
	}
	ctx = vo.OnEnd(ctx, output)

	logx.Infof("问题改写完成: question='%s', rewritten='%s', subQuestions=%v",
		question, rewriteResult.RewrittenQuestion, rewriteResult.SubQuestions)

	return nil
}

// buildRewriteStageSnapshot 构建改写阶段的统一快照
func (q *QueryRewriteStage) buildRewriteStageSnapshot(question, historySummary string, rewriteResult *vo.QuestionRewriteResult) map[string]any {
	snapshot := make(map[string]any)
	snapshot["originalQuestion"] = strutil.Trim(question)
	snapshot["historyContext"] = strutil.Trim(historySummary)

	// 仅当改写结果存在时追加输出相关字段
	if rewriteResult != nil {
		snapshot["rewriteQuestion"] = strutil.Trim(rewriteResult.RewrittenQuestion)
		snapshot["subQuestions"] = rewriteResult.SubQuestions
		snapshot["rawModelOutput"] = strutil.Trim(rewriteResult.RawModelOutput)
	}
	// 追加当前配置的改写参数（便于离线分析参数影响）
	snapshot["rewriteOverrideEnabled"] = q.enabled
	snapshot["rewriteTemperature"] = q.temperature
	snapshot["rewriteTopP"] = q.topP
	snapshot["rewriteThinking"] = q.thinking
	return snapshot
}
