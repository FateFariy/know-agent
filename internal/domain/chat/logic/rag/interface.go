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
	Retrieve(ctx context.Context, plan *vo.RetrievalPlan) (*vo.RetrievalResult, error)
}

// Fusion 检索结果融合接口
type Fusion interface {
	// Fuse 对多通道检索结果执行加权融合，返回排序后的候选文档列表
	Fuse(ctx context.Context, results []*RetrievalChannelResult, plan *vo.RetrievalPlan) vo.DocumentChunks
}
