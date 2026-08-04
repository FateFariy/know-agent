package rag

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// Retriever RAG 检索引擎接口
type Retriever interface {
	Retrieve(ctx context.Context, plan *vo.ConversationExecutionPlan) (*vo.RagRetrievalContext, error)
}
