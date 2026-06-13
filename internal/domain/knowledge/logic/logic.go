package logic

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/req"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// DocumentKnowledgeService 文档知识服务接口
type DocumentKnowledgeService interface {
	// ListRetrievableDocuments 获取可检索的文档列表
	ListRetrievableDocuments(ctx context.Context) ([]*vo.KnowledgeDocumentDescriptor, error)

	// VectorSearch 向量检索
	VectorSearch(ctx context.Context, request *req.DocumentRetrieveRequest) ([]*vo.SearchDocument, error)

	// KeywordSearch 关键词检索
	KeywordSearch(ctx context.Context, request *req.DocumentRetrieveRequest) ([]*vo.SearchDocument, error)

	// ElevateToParentBlocks 将子文档提升到父块级别
	ElevateToParentBlocks(ctx context.Context, childDocuments []*vo.SearchDocument, maxChars int) ([]*vo.SearchDocument, error)
}