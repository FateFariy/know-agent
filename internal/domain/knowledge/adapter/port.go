package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// Embedder 文本嵌入模型
type Embedder interface {
	// EmbedStrings 文本向量化
	EmbedStrings(ctx context.Context, texts ...string) ([][]float64, error)
}

// RouteLexicalIndex 路由侧的词面索引能力
type RouteLexicalIndex interface {
	// Search 在指定实体类型下进行词面检索，返回命中 (entityCode/documentId, score) 列表
	Search(ctx context.Context, routingText string, entityType string, size int) ([]*vo.RouteLexicalHit, error)
}

type DocumentGateway interface {
	// CountRetrievableDocumentsByKbIds 按知识库ID列表统计可检索文档数量（返回 map[kbId]count）
	CountRetrievableDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error)

	// FindRetrievableByKbIds 根据知识库ID列表查询可检索的文档元数据
	FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*vo.DocumentMetadata, error)

	// FindRetrieveDocumentByIds 根据ID列表获取可检索的文档元数据
	FindRetrieveDocumentByIds(ctx context.Context, ids []int64) ([]*vo.DocumentMetadata, error)

	// FindDocumentProfileByDocIds 根据文档ID列表获取文档属性
	FindDocumentProfileByDocIds(ctx context.Context, docIds []int64) ([]*vo.DocumentProfile, error)
}
