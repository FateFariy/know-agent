package logic

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// DocumentKnowledgeLogic 文档知识服务
type DocumentKnowledgeLogic interface {
	// ListRetrievableDocuments 获取可检索的文档列表
	ListRetrievableDocuments(ctx context.Context) ([]*vo.KnowledgeDocument, error)

	// VectorSearch 向量检索
	VectorSearch(ctx context.Context, request *vo.DocumentRetrieve) ([]*vo.Document, error)

	// KeywordSearch 关键词检索
	KeywordSearch(ctx context.Context, request *vo.DocumentRetrieve) ([]*vo.Document, error)

	// ElevateToParentBlocks 将子文档提升到父块级别
	ElevateToParentBlocks(ctx context.Context, childDocuments []*vo.Document, maxChars int) ([]*vo.Document, error)
}
