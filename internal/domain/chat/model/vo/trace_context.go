package vo

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/callbacks"
)

// CtxTraceKey 存储 ConversationTrace 的 ctx key
type CtxTraceKey struct{}

// TraceFromCtx 从 ctx 获取 ConversationTrace
func TraceFromCtx(ctx context.Context) *ConversationTrace {
	if v := ctx.Value(CtxTraceKey{}); v != nil {
		if t, ok := v.(*ConversationTrace); ok {
			return t
		}
	}
	return nil
}

// WithTrace 将 ConversationTrace 存入 ctx
func WithTrace(ctx context.Context, trace *ConversationTrace) context.Context {
	return context.WithValue(ctx, CtxTraceKey{}, trace)
}

// StageInput OnStart 的输入
type StageInput struct {
	SummaryText string
	Snapshot    any
}

// StageOutput OnEnd 的输入
type StageOutput = StageInput

// StageError 用于在 OnError 回调中传递错误信息
type StageError struct {
	Err         error
	SummaryText string
}

func (e *StageError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.SummaryText
}

func (e *StageError) Unwrap() error {
	return e.Err
}

// OnStart 构造 RunInfo 并调用 callbacks.OnStart
func OnStart(ctx context.Context, stageCode *ConversationTraceStage, executionMode string, input *StageInput) context.Context {
	runInfo := &callbacks.RunInfo{
		StageId:       utils.GetSnowflakeNextID(),
		StageCode:     stageCode,
		ExecutionMode: executionMode,
		StartTime:     time.Now(),
	}
	ctx = callbacks.EnsureRunInfo(ctx, runInfo)
	ctx = callbacks.OnStart(ctx, input)
	return ctx
}

// OnEnd 调用 callbacks.OnEnd
func OnEnd(ctx context.Context, output *StageOutput) context.Context {
	return callbacks.OnEnd(ctx, output)
}

// OnError 调用 callbacks.OnError
func OnError(ctx context.Context, summaryText string, err error) context.Context {
	return callbacks.OnError(ctx, &StageError{err, summaryText})
}
