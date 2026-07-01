package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

type KnowledgeRepository interface {
	// Do 运行一个事务
	Do(ctx context.Context, fn func(ctx context.Context) error) error

	// SelectRetrievableDocuments 查询可检索的文档
	SelectRetrievableDocuments(ctx context.Context, documentIds ...int64) ([]*vo.KnowledgeDocument, error)

	// SelectParentBlocks 根据ID列表查询父级块
	SelectParentBlocks(ctx context.Context, parentBlockIDs []int64) ([]*entity.DocumentParentBlock, error)
}
