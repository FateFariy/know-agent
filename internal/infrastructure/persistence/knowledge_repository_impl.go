package persistence

import (
	"context"

	"github.com/cloudwego/eino/components/retriever"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/data"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	"github.com/swiftbit/know-agent/internal/infrastructure/model"
	"github.com/swiftbit/know-agent/internal/svc"
)

type KnowledgeRepositoryImpl struct {
	retriever *retriever.Retriever
	*transactionManager
}

var _ adapter.KnowledgeRepository = (*KnowledgeRepositoryImpl)(nil)

func NewKnowledgeRepository(svcCtx *svc.ServiceContext) *KnowledgeRepositoryImpl {
	return &KnowledgeRepositoryImpl{
		transactionManager: &transactionManager{db: svcCtx.Db},
	}
}

// SelectAllDocuments 查询所有文档
func (k *KnowledgeRepositoryImpl) SelectAllDocuments(ctx context.Context) ([]*vo.KnowledgeDocument, error) {
	return k.SelectDocumentsByIDs(ctx, nil)
}

// SelectDocumentsByIDs 根据ID列表查询文档
func (k *KnowledgeRepositoryImpl) SelectDocumentsByIDs(ctx context.Context, documentIDs []int64) ([]*vo.KnowledgeDocument, error) {
	var documents []*vo.KnowledgeDocument
	query := k.dbWithContext(ctx).Model(&model.Document{})
	if len(documentIDs) > 0 {
		query = query.Where("id in ?", documentIDs)
	}
	if err := query.Find(&documents).Error; err != nil {
		return nil, err
	}
	return documents, nil
}

func (k *KnowledgeRepositoryImpl) SearchByVector(ctx context.Context, query string, documentIDs, taskIDs []int64, topK int, filters *vo.DocumentRetrieveFilters) ([]*data.EmbeddingChunk, error) {
	// TODO implement me
	panic("implement me")
}

func (k *KnowledgeRepositoryImpl) SearchByKeyword(ctx context.Context, query string, documentIDs, taskIDs []int64, topK int, filters *vo.DocumentRetrieveFilters) ([]*data.EmbeddingChunk, error) {
	// TODO implement me
	panic("implement me")
}

// SelectParentBlocks 根据ID列表查询父级块
func (k *KnowledgeRepositoryImpl) SelectParentBlocks(ctx context.Context, parentBlockIDs []int64) ([]*entity.DocumentParentBlock, error) {
	if len(parentBlockIDs) == 0 {
		return nil, nil
	}
	var parentBlocks []*entity.DocumentParentBlock
	if err := k.dbWithContext(ctx).Model(&model.DocumentParentBlock{}).
		Where("id in ?", parentBlockIDs).
		Order("parent_no ASC").
		Find(&parentBlocks).Error; err != nil {
		return nil, err
	}
	return parentBlocks, nil
}
