package conversation

import (
	"context"
)

// Stage 表示对话流程中的一个阶段
type Stage interface {
	// Name 返回阶段名称
	Name() string

	// Execute 执行阶段逻辑
	// ctx: 标准上下文，用于控制取消、超时和传递请求作用域的值
	// convCtx: 对话上下文，携带会话状态和业务数据
	// sink: 事件输出器，用于发送流式事件
	Execute(ctx context.Context, convCtx *Context) error
}

// ConditionalStage 条件执行阶段
type ConditionalStage interface {
	Stage

	// ShouldExecute 决定是否执行该阶段
	ShouldExecute(ctx context.Context, convCtx *Context) bool
}
