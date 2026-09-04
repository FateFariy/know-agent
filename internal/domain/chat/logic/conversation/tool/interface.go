package tool

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/route"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// Info 工具元信息
type Info struct {
	// Name 工具的唯一名称，需在同一工具集内唯一
	Name string

	// Description 描述工具的用途与调用时机，可直接作为模型提示
	Description string
}

// Tool 通用工具
type Tool[I, O any] interface {
	// Info 返回工具元信息
	Info(ctx context.Context) *Info

	// Invoke 调用工具，返回工具执行结果
	Invoke(ctx context.Context, args I) (O, error)
}

// Retriever RAG 检索引擎接口
type Retriever interface {
	Retrieve(ctx context.Context, plan *vo.RetrievalPlan) (*vo.RetrievalResult, error)
}

// DocumentRouter 文档路由器
type DocumentRouter interface {
	// Route 根据文档ID和问题进行文档内路由
	Route(ctx context.Context, input *route.NavigationInput) (*vo.DocumentNavigationDecision, error)
}
