package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// KnowledgeRepository 知识存储库接口
type KnowledgeRepository interface {
	// ListDocuments 获取文档列表
	ListDocuments(ctx context.Context) ([]*vo.KnowledgeDocumentDescriptor, error)

	// SearchByVector 向量检索
	SearchByVector(ctx context.Context, query string, topK int, scoreThreshold float64) ([]*vo.SearchDocument, error)

	// SearchByKeyword 关键词检索
	SearchByKeyword(ctx context.Context, query string, topK int) ([]*vo.SearchDocument, error)

	// GetParentBlock 获取父块内容
	GetParentBlock(ctx context.Context, documentID string, maxChars int) (*vo.SearchDocument, error)
}
