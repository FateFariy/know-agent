package conversation

import (
	"context"
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rewrite"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// QuestionRewriteStage 问题改写阶段
// 负责对所有非 OpenChat 模式的问题进行改写，生成检索友好的问题表达和子问题拆分。
// 仅当 RAG 开启时执行，OpenChat 模式跳过。
type QuestionRewriteStage struct {
	rewriter        rewrite.QueryRewriter
	rewriteEnabled  bool
	ragEnabled      bool
	maxSubQuestions int
	temperature     float32
	topP            float32
	thinking        bool
}

var _ Stage = (*QuestionRewriteStage)(nil)

func NewQuestionRewriteStage(rewriter rewrite.QueryRewriter, rewriteEnabled, ragEnabled bool,
	maxSubQuestions int, temperature, topP float32, thinking bool) *QuestionRewriteStage {
	return &QuestionRewriteStage{
		rewriter:        rewriter,
		rewriteEnabled:  rewriteEnabled,
		ragEnabled:      ragEnabled,
		maxSubQuestions: maxSubQuestions,
		temperature:     temperature,
		topP:            topP,
		thinking:        thinking,
	}
}

// Name 阶段名称
func (q *QuestionRewriteStage) Name() string {
	return enum.ConversationTraceStageRewrite.Name
}

// Execute 执行问题改写
func (q *QuestionRewriteStage) Execute(ctx context.Context, convCtx *Context) error {
	// OpenChat 模式不需要问题改写
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

	question := strutil.Trim(convCtx.Question)
	historySummary := execPlan.HistorySummary

	// 启动改写追踪阶段
	ctx = vo.OnStart(ctx, enum.ConversationTraceStageRewrite, enum.ExecutionModeRetrieval.String(),
		&vo.StageInput{SummaryText: "正在生成检索友好的问题表达。",
			Snapshot: buildRewriteStageSnapshot(question, historySummary, nil, q.rewriteEnabled, q.temperature, q.topP, q.thinking)})

	// 调用改写逻辑（原始问题 + 历史摘要 → 改写问题 + 子问题）
	rewriteResult, err := q.rewriter.Rewrite(ctx, question, historySummary)
	if err != nil {
		ctx = vo.OnError(ctx, "问题改写失败。", err)
		return err
	}

	// 提交改写追踪（包含改写结果快照以便离线分析）
	ctx = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "问题改写完成。",
		Snapshot: buildRewriteStageSnapshot(question, historySummary, rewriteResult, q.rewriteEnabled, q.temperature, q.topP, q.thinking)})

	// 对改写结果做兜底处理
	//  - RewrittenQuestion 为空时回退到原始问题
	//  - SubQuestions 为空时使用改写问题作为单元素列表
	rewriteResult.RewrittenQuestion = utils.BlankToDefault(rewriteResult.RewrittenQuestion, question)
	if len(rewriteResult.SubQuestions) == 0 {
		rewriteResult.SubQuestions = []string{rewriteResult.RewrittenQuestion}
	}

	// 更新执行计划
	execPlan.RewriteQuestion = rewriteResult.RewrittenQuestion
	execPlan.RewriteSubQuestions = rewriteResult.SubQuestions
	execPlan.RetrievalQuestion = rewriteResult.RewrittenQuestion
	execPlan.RetrievalSubQuestions = rewriteResult.SubQuestions
	convCtx.SetExecutePlan(execPlan)

	logx.Infof("问题改写完成: question='%s', rewritten='%s', subQuestions=%v",
		question, rewriteResult.RewrittenQuestion, rewriteResult.SubQuestions)
	return nil
}

// ShouldExecute 决定是否执行该阶段（实现 ConditionalStage 接口）
func (q *QuestionRewriteStage) ShouldExecute(ctx context.Context, convCtx *Context) bool {
	return convCtx.ChatMode != enum.ChatQueryModeOpenChat
}

// buildRewriteStageSnapshot 构建改写阶段的统一快照，供追踪的 StartStage/Fail/Complete 三处复用。
//
// 快照字段：
//   - 原始问题、历史摘要（始终输出）
//   - 若 rewriteResult 非空，则输出改写后的问题、子问题列表、模型原始输出
//   - 改写开关、temperature、topP、thinking 等模型参数，用于离线分析
func buildRewriteStageSnapshot(question, historySummary string, rewriteResult *vo.QuestionRewriteResult,
	rewriteEnabled bool, temperature, topP float32, thinking bool) map[string]any {
	snapshot := make(map[string]any)
	snapshot["originalQuestion"] = strutil.Trim(question)
	snapshot["historyContext"] = strutil.Trim(historySummary)

	// 仅当改写结果存在时追加输出相关字段（避免在 StartStage 阶段填充空值）
	if rewriteResult != nil {
		snapshot["rewriteQuestion"] = strutil.Trim(rewriteResult.RewrittenQuestion)
		snapshot["subQuestions"] = rewriteResult.SubQuestions
		snapshot["rawModelOutput"] = strutil.Trim(rewriteResult.RawModelOutput)
	}
	// 追加当前配置的改写参数（便于离线分析参数影响）
	snapshot["rewriteOverrideEnabled"] = rewriteEnabled
	snapshot["rewriteTemperature"] = temperature
	snapshot["rewriteTopP"] = topP
	snapshot["rewriteThinking"] = thinking
	return snapshot
}
