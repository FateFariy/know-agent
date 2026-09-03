package conversation

import (
	"context"
)

// ToolInfo 工具元信息
type ToolInfo struct {
	// Name 工具的唯一名称，需在同一工具集内唯一
	Name string

	// Description 描述工具的用途与调用时机，可直接作为模型提示
	Description string
}

// Tool 通用工具
type Tool[I, O any] interface {
	// Info 返回工具元信息
	Info(ctx context.Context) *ToolInfo

	// Invoke 调用工具，返回工具执行结果
	Invoke(ctx context.Context, args I) (O, error)
}
