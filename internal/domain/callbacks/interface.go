package callbacks

import (
	"context"
	"time"
)

// RunInfo 描述当前被拦截的组件身份
type RunInfo struct {
	StageId       int64     // 阶段ID
	Payload       any       // 业务负载（如 *ConversationTraceStage 或 *vo.ModelCallMeta）
	ExecutionMode string    // 执行模式
	StartTime     time.Time // 开始时间
	Component     string    // 组件标识，用于 Handler 判别（如 "trace"、"model_usage"）
}

// Timing 定义拦截的时机
type Timing = int

const (
	TimingOnStart Timing = iota
	TimingOnEnd
	TimingOnError
)

// Handler 用户实现的拦截器接口
type Handler interface {
	OnStart(ctx context.Context, info *RunInfo, input any) context.Context
	OnEnd(ctx context.Context, info *RunInfo, output any) context.Context
	OnError(ctx context.Context, info *RunInfo, err error) context.Context
}
