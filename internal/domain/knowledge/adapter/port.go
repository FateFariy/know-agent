package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// Embedder 文本嵌入模型
type Embedder interface {
	// Embedding 文本向量化
	Embedding(ctx context.Context, texts ...string) ([][]float64, error)
}

type DocumentGateway interface {
	// CountDocumentsByKbIds 按知识库ID列表统计文档数量（返回 map[kbId]count）
	CountDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error)

	// CountRetrievableDocumentsByKbIds 按知识库ID列表统计可检索文档数量（返回 map[kbId]count）
	CountRetrievableDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error)

	// FindRetrievableByKbIds 根据知识库ID列表查询可检索的文档元数据
	FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*vo.DocumentMetadata, error)

	// FindRetrieveDocumentByIds 根据ID列表获取可检索的文档元数据
	FindRetrieveDocumentByIds(ctx context.Context, ids ...int64) ([]*vo.DocumentMetadata, error)

	// FindDocumentProfileByDocIds 根据文档ID列表获取文档属性
	FindDocumentProfileByDocIds(ctx context.Context, docIds []int64) ([]*vo.DocumentProfile, error)

	// FindDocumentProfiles 获取所有文档属性
	FindDocumentProfiles(ctx context.Context) ([]*vo.DocumentProfile, error)
}
