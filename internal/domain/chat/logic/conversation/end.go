package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
)

type End struct {
	repo            adapter.ChatRepository
	distributedLock adapter.DistributedLock
}

var _ Stage = (*End)(nil)

func NewEnd(repo adapter.ChatRepository, distributedLock adapter.DistributedLock) *End {
	return &End{
		repo:            repo,
		distributedLock: distributedLock,
	}
}

// Name 阶段名称
func (e *End) Name() string {
	return "结束会话"
}

// Execute 执行逻辑
func (e *End) Execute(ctx context.Context, convCtx *Context) error {
	panic("unimplemented")
}

// // refreshDebugTraceRuntimeStats 刷新调试轨迹中的统计信息
// func (e *End) refreshDebugTraceRuntimeStats(convCtx *Context) {
// 	debugTrace := convCtx.DebugTrace.Load()
// 	if debugTrace == nil {
// 		return
// 	}
// 	modelUsageTraces := convCtx.Trace.SnapshotModelUsageTraces()
// 	debugTrace.ModelUsageTraces = modelUsageTraces
// 	debugTrace.LimitStats = &vo.ChatLimitStats{
// 		ModelCallsUsed:        len(modelUsageTraces),
// 		ToolCallsUsed:         len(convCtx.SnapshotUsedTools()),
// 		ModelCallsRunLimit:    c.options.maxModelCallsPerRun,
// 		ToolCallsRunLimit:     c.options.maxToolCallsPerRun,
// 		ModelCallsThreadLimit: c.options.maxModelCallsPerThread,
// 		ToolCallsThreadLimit:  c.options.maxToolCallsPerThread,
// 	}
// 	convCtx.DebugTrace.Store(debugTrace)
// }
