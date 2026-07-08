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
