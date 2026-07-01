package logic

import (
	"context"

	vo2 "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
)

// DocumentKnowledgeLogic 文档知识服务
type DocumentKnowledgeLogic interface {
	// ListRetrievableDocuments 获取可检索的文档列表
	ListRetrievableDocuments(ctx context.Context) ([]*vo2.KnowledgeDocument, error)

	// VectorSearch 向量检索
	VectorSearch(ctx context.Context, request *vo2.DocumentRetrieve) ([]*vo2.DocumentChunk, error)

	// KeywordSearch 关键词检索
	KeywordSearch(ctx context.Context, request *vo2.DocumentRetrieve) ([]*vo2.DocumentChunk, error)

	// ElevateToParentBlocks 将子文档提升到父块级别
	ElevateToParentBlocks(ctx context.Context, childDocuments []*vo2.DocumentChunk, maxChars int) ([]*vo2.DocumentChunk, error)
}
