package rank

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

type EsSearchInput struct {
	RoutingText string
	EntityType  string
	Size        int
	KbIds       []int64
}

// RouteLexicalIndex 路由侧的词面索引能力
type RouteLexicalIndex interface {
	// Search 在指定实体类型下进行词面检索，返回命中 (entityCode/documentId, score) 列表
	Search(ctx context.Context, input *EsSearchInput) ([]*vo.RouteLexicalHit, error)
}
