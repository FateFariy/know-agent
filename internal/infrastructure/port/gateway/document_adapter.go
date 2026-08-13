package gateway

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

type DocumentAdapter struct {
	repo adapter.DocumentRepository
}

// NewDocumentAdapter 创建文档适配器
func NewDocumentAdapter(repo adapter.DocumentRepository) *DocumentAdapter {
	return &DocumentAdapter{
		repo: repo,
	}
}

// CountRetrievableDocumentsByKbIds 按知识库ID列表统计可检索文档数量（返回 map[kbId]count）
func (a *DocumentAdapter) CountRetrievableDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error) {
	return a.repo.CountRetrievableDocumentsByKbIds(ctx, kbIds)
}

// FindRetrievableByKbIds 根据知识库ID列表查询可检索的文档元数据
func (a *DocumentAdapter) FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*vo.DocumentMetadata, error) {
	documentMetadata, err := a.repo.SelectRetrievableDocumentsByIds(ctx, kbIds...)
	if err != nil {
		return nil, err
	}
	result := make([]*vo.DocumentMetadata, 0, len(documentMetadata))
	for _, metadata := range documentMetadata {
		result = append(result, &vo.DocumentMetadata{
			DocumentId:        metadata.DocumentId,
			DocumentName:      metadata.DocumentName,
			KnowledgeBaseId:   metadata.KnowledgeBaseId,
			KnowledgeBaseName: metadata.KnowledgeBaseName,
			LastIndexTaskId:   metadata.LastIndexTaskId,
		})
	}
	return result, nil
}

func (a *DocumentAdapter) FindRetrieveDocumentByIds(ctx context.Context, ids []int64) ([]*vo.DocumentMetadata, error) {
	documentMetadata, err := a.repo.SelectRetrievableDocumentsByIds(ctx, ids...)
	if err != nil {
		return nil, err
	}
	result := make([]*vo.DocumentMetadata, 0, len(documentMetadata))
	for _, metadata := range documentMetadata {
		result = append(result, &vo.DocumentMetadata{
			DocumentId:        metadata.DocumentId,
			DocumentName:      metadata.DocumentName,
			KnowledgeBaseId:   metadata.KnowledgeBaseId,
			KnowledgeBaseName: metadata.KnowledgeBaseName,
			LastIndexTaskId:   metadata.LastIndexTaskId,
		})
	}
	return result, nil
}

func (a *DocumentAdapter) FindDocumentProfileByDocIds(ctx context.Context, docIds []int64) ([]*vo.DocumentProfile, error) {
	documentProfile, err := a.repo.SelectDocumentProfilesByDocIds(ctx, docIds)
	if err != nil {
		return nil, err
	}
	result := make([]*vo.DocumentProfile, 0, len(documentProfile))
	for _, profile := range documentProfile {
		result = append(result, &vo.DocumentProfile{
			DocumentId:       profile.DocumentId,
			DocumentSummary:  profile.DocumentSummary,
			CoreTopics:       profile.CoreTopics,
			ExampleQuestions: profile.ExampleQuestions,
		})
	}
	return result, nil
}

// FindDocumentProfiles 查询所有文档元数据
func (a *DocumentAdapter) FindDocumentProfiles(ctx context.Context) ([]*vo.DocumentProfile, error) {
	documentProfiles, err := a.repo.SelectDocumentProfiles(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*vo.DocumentProfile, 0, len(documentProfiles))
	for _, profile := range documentProfiles {
		result = append(result, &vo.DocumentProfile{
			DocumentId:       profile.DocumentId,
			DocumentSummary:  profile.DocumentSummary,
			CoreTopics:       profile.CoreTopics,
			ExampleQuestions: profile.ExampleQuestions,
		})
	}
	return result, nil
}
