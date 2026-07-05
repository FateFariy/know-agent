package executor

import (
	"context"
	"fmt"

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

// Execute 当路由产生多个候选文档且置信度不足时，交互在此处暂停并向用户确认知识范围。
//
// 执行流程：
//  1. 从 convCtx 加载执行计划（缺失时直接报错）
//  2. 启动澄清路由追踪阶段
//  3. 准备澄清回复、原因、候选项；原因写入调试轨迹
//  4. 发布思考事件 + 状态事件（原因非空时）
//  5. 提交追踪快照，最后通过 singleValueChan 返回澄清文本
func (e *ClarificationExecutor) Execute(ctx context.Context, convCtx *vo.ConversationContext) (<-chan string, error) {
	// 加载执行计划，缺失时直接报错
	plan := convCtx.ExecutionPlan.Load()
	if plan == nil {
		return nil, fmt.Errorf("invalid value")
	}

	// 启动澄清路由追踪阶段（以 Mode 名称标识执行路径）
	routeStage, err := e.tracer.StartStage(ctx, convCtx.Trace, vo.ConversationTraceStageRoute, e.Mode().Name(), "当前候选存在歧义，先返回澄清问题。", nil)
	if err != nil {
		return nil, err
	}

	// 从执行计划中取出澄清文本、原因与候选项；原因写入调试轨迹以便离线分析
	reply := utils.BlankToDefault(plan.ClarificationReply, "当前我无法稳定判断你想问哪份知识文档，请补充更具体的文档名、主题或关键词。")
	reason := plan.ClarificationReason
	options := plan.ClarificationOptions

	if debugTrace := convCtx.DebugTrace.Load(); debugTrace != nil {
		if strutil.IsNotBlank(reason) {
			debugTrace.AddRetrievalNotes(reason)
		}
	}

	// 向客户端流发布思考事件；原因非空时再追加一条状态事件
	if err = publishThinking(convCtx, "当前问题涉及多份候选文档，先向你确认知识范围。"); err != nil {
		return nil, err
	}
	if strutil.IsNotBlank(reason) {
		if err = publishStatus(convCtx, reason); err != nil {
			return nil, err
		}
	}

	// 提交追踪快照（包含回复、原因、候选项），通过 singleValueChan 返回澄清文本
	snapshot := map[string]any{
		"clarificationReply":   reply,
		"clarificationReason":  reason,
		"clarificationOptions": options,
	}
	if err = e.tracer.CompleteStage(ctx, routeStage, "已返回澄清问题。", snapshot); err != nil {
		return nil, err
	}
	return singleValueChan(reply), nil
}
