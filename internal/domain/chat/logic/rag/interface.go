package rag

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// Retrieval 检索通道接口
type Retrieval interface {
	// Name 检索通道名称
	Name() string

	// Retrieve 根据子问题检索
	Retrieve(ctx context.Context, input *ExecutionInput) (*RetrievalChannelResult, error)
}

// Retriever RAG 检索引擎接口
type Retriever interface {
	Retrieve(ctx context.Context, plan *vo.ConversationExecutionPlan) (*vo.RetrievalResult, error)
}
