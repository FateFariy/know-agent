package executor

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// singleValueChan 将给定字符串包装为一个已关闭的带缓冲只读 channel，便于与流式管道拼接
func singleValueChan(content string) <-chan string {
	ch := make(chan string, 1)
	defer close(ch)
	ch <- content
	return ch
}

// Executor 对话执行器接口，根据执行模式负责生成最终回答。
type Executor interface {
	// Mode 返回当前执行器对应的执行模式
	Mode() enum.ExecutionMode

	// Execute 执行回答生成逻辑
	Execute(ctx context.Context, convCtx *conversation.Context) (<-chan string, error)
}
