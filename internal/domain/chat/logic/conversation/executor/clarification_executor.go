package executor

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/trace"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ClarificationExecutor 路由歧义澄清执行器
// 当路由阶段判定候选文档存在歧义时，直接返回澄清话术并记录澄清原因
type ClarificationExecutor struct {
	tracer *trace.ConversationTraceRecorder
}

// NewClarificationExecutor 构造澄清执行器
func NewClarificationExecutor(tracer *trace.ConversationTraceRecorder) *ClarificationExecutor {
	return &ClarificationExecutor{tracer: tracer}
}

var _ conversation.Executor = (*ClarificationExecutor)(nil)

// Mode 返回 CLARIFICATION
func (e *ClarificationExecutor) Mode() vo.ExecutionMode {
	return vo.ExecutionModeClarification
}

// Execute 下发澄清思考事件 + 澄清话术
func (e *ClarificationExecutor) Execute(ctx context.Context, convCtx *vo.ConversationContext) (<-chan string, error) {
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return nil, nil
	}

	reply := utils.BlankToDefault(plan.ClarificationReply, "当前我无法稳定判断你想问哪份知识文档，请补充更具体的文档名、主题或关键词。")
	reason := plan.ClarificationReason
	options := plan.ClarificationOptions

	// 记录原因到调试轨迹
	if debugTrace := convCtx.DebugTrace.Load(); debugTrace != nil {
		if strutil.IsNotBlank(reason) {
			debugTrace.AddRetrievalNote(reason)
		}
	}

	publishThinking(convCtx, reply)
	if strutil.IsNotBlank(reason) {
		publishStatus(convCtx, reason)
		if len(options) > 0 {
			publishRecommendations(convCtx, options)
		}
	}

	// 记录路由阶段追踪
	routeStage, err := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRoute, e.Mode().Name(), "当前候选存在歧义，先返回澄清问题。", nil)
	if err != nil {
		return nil, err
	}

	snapshot := map[string]any{
		"clarificationReply":   reply,
		"clarificationReason":  reason,
		"clarificationOptions": options,
	}
	return nil, e.tracer.CompleteStage(ctx, routeStage, "已返回澄清问题。", snapshot)
}
