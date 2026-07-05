package executor

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	ragvo "github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ReactAgentExecutor 开放式 Agent 执行器
// 当问题跨越单篇文档知识边界时，将问题交由 ReAct Agent 自行决定工具调用与回答组合
type ReactAgentExecutor struct {
	reactAgent ragvo.ReactAgentService
	tracer     *trace.ConversationTraceRecorder
}

// NewReactAgentExecutor 构造 ReAct Agent 执行器
func NewReactAgentExecutor(
	reactAgent ragvo.ReactAgentService,
	tracer *trace.ConversationTraceRecorder,
) *ReactAgentExecutor {
	return &ReactAgentExecutor{
		reactAgent: reactAgent,
		tracer:     tracer,
	}
}

var _ conversation.Executor = (*ReactAgentExecutor)(nil)

// Mode 返回 REACT_AGENT
func (e *ReactAgentExecutor) Mode() vo.ExecutionMode { return vo.ExecutionModeReactAgent }

// Execute 调用 ReAct Agent 流式输出
func (e *ReactAgentExecutor) Execute(ctx context.Context, convCtx *vo.ConversationContext) (<-chan string, error) {
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return nil, nil
	}

	publishThinking(convCtx, "问题涉及多方面信息，交由 ReAct Agent 综合回答。")

	agentStage, err := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageReActAgent,
		e.Mode().Name(), "ReAct Agent 正在思考与执行。", nil)

	streamCh, err := e.reactAgent.Stream(ctx, plan.OriginalQuestion)
	if err != nil {
		logx.Errorf("ReAct Agent 调用失败: conversationId=%s err=%v", convCtx.ConversationId, err)
		e.tracer.FailStage(ctx, agentStage, "ReAct Agent 执行失败。", err, nil)
		publishText(convCtx, utils.BlankToDefault(plan.NoEvidenceReply, defaultNoEvidenceReply))
		return nil, err
	}

	snapshot := map[string]any{
		"firstResponseTimeMs": convCtx.FirstResponseTimeMs.Load(),
		"answerLength":        convCtx.AnswerLength(),
	}
	_ = e.tracer.CompleteStage(ctx, agentStage, "ReAct Agent 回答完成。", snapshot)
	return streamCh, nil
}
