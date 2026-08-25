package gateway

import (
	"context"
	"strconv"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

type DocumentAdapterForKnowledge struct {
	repo adapter.DocumentRepository
}

// NewDocumentAdapterForKnowledge 创建文档适配器
func NewDocumentAdapterForKnowledge(repo adapter.DocumentRepository) *DocumentAdapterForKnowledge {
	return &DocumentAdapterForKnowledge{
		repo: repo,
	}
}

// CountDocumentsByKbIds 按知识库ID列表统计文档数量（返回 map[kbId]count）
func (a *DocumentAdapterForKnowledge) CountDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error) {
	return a.repo.CountDocumentsByKbIds(ctx, kbIds)
}

// CountRetrievableDocumentsByKbIds 按知识库ID列表统计可检索文档数量（返回 map[kbId]count）
func (a *DocumentAdapterForKnowledge) CountRetrievableDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error) {
	return a.repo.CountRetrievableDocumentsByKbIds(ctx, kbIds)
}

// FindRetrievableByKbIds 根据知识库ID列表查询可检索的文档元数据
func (a *DocumentAdapterForKnowledge) FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*vo.DocumentMetadata, error) {
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

func (a *DocumentAdapterForKnowledge) FindRetrieveDocumentByIds(ctx context.Context, ids ...int64) ([]*vo.DocumentMetadata, error) {
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

func (a *DocumentAdapterForKnowledge) FindDocumentProfileByDocIds(ctx context.Context, docIds []int64) ([]*vo.DocumentProfile, error) {
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

// FindDocumentProfiles 查询所有文档画像元数据
func (a *DocumentAdapterForKnowledge) FindDocumentProfiles(ctx context.Context) ([]*vo.DocumentProfile, error) {
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

type DocumentAdapterForChat struct {
	repo adapter.DocumentRepository
}

func NewDocumentAdapterForChat(repo adapter.DocumentRepository) *DocumentAdapterForChat {
	return &DocumentAdapterForChat{
		repo: repo,
	}
}

func (a *DocumentAdapterForChat) FindRetrievableByKbIds(ctx context.Context, kbIds []int64) ([]*cvo.DocumentMetadata, error) {
	documentMetadata, err := a.repo.SelectRetrievableDocumentsByIds(ctx, kbIds...)
	if err != nil {
		return nil, err
	}
	result := make([]*cvo.DocumentMetadata, 0, len(documentMetadata))
	for _, metadata := range documentMetadata {
		result = append(result, &cvo.DocumentMetadata{
			DocumentId:        metadata.DocumentId,
			DocumentName:      metadata.DocumentName,
			KnowledgeBaseId:   metadata.KnowledgeBaseId,
			KnowledgeBaseName: metadata.KnowledgeBaseName,
			LastIndexTaskId:   metadata.LastIndexTaskId,
		})
	}
	return result, nil
}

func (a *DocumentAdapterForChat) FindRetrieveDocumentByIds(ctx context.Context, ids ...int64) ([]*cvo.DocumentMetadata, error) {
	documentMetadata, err := a.repo.SelectRetrievableDocumentsByIds(ctx, ids...)
	if err != nil {
		return nil, err
	}
	result := make([]*cvo.DocumentMetadata, 0, len(documentMetadata))
	for _, metadata := range documentMetadata {
		result = append(result, &cvo.DocumentMetadata{
			DocumentId:        metadata.DocumentId,
			DocumentName:      metadata.DocumentName,
			KnowledgeBaseId:   metadata.KnowledgeBaseId,
			KnowledgeBaseName: metadata.KnowledgeBaseName,
			LastIndexTaskId:   metadata.LastIndexTaskId,
		})
	}
	return result, nil
}

func (a *DocumentAdapterForChat) FindParentChunks(ctx context.Context, ids []int64) ([]*cvo.DocumentChunk, error) {
	documentChunks, err := a.repo.SelectParentChunkListByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]*cvo.DocumentChunk, 0, len(documentChunks))
	for _, chunk := range documentChunks {
		result = append(result, &cvo.DocumentChunk{
			ID:                strconv.FormatInt(chunk.ID, 10),
			Content:           chunk.ParentText,
			SourceType:        chunk.SourceTypeName,
			TaskId:            chunk.TaskId,
			DocumentId:        chunk.DocumentId,
			ChunkNo:           chunk.ParentNo,
			SectionPath:       chunk.SectionPath,
			StructureNodeId:   chunk.StructureNodeId,
			StructureNodeType: chunk.StructureNodeType,
			CanonicalPath:     chunk.CanonicalPath,
			ItemIndex:         chunk.ItemIndex,
		})
	}
	return result, nil
}

func (a *DocumentAdapterForChat) ListSections(ctx context.Context, documentId int64) ([]*cvo.GraphSection, error) {
	nodes, err := a.repo.SelectStructureNodeListByDocumentId(ctx, documentId)
	if err != nil {
		return nil, err
	}
	return utils.FilterMapUniqueLimit(nodes, -1, func(r *entity.StructureNode) (int64, *cvo.GraphSection, bool) {
		return r.ID, toGraphSection(r), r != nil && r.NodeType == enum.NodeTypeSection
	}), nil
}

func (a *DocumentAdapterForChat) FindSectionById(ctx context.Context, documentId, nodeId int64) (*cvo.GraphSection, error) {
	if documentId == 0 || nodeId == 0 {
		return nil, nil
	}
	sections, err := a.ListSections(ctx, documentId)
	if err != nil {
		return nil, err
	}
	for _, s := range sections {
		if s.NodeId == nodeId {
			return s, nil
		}
	}
	return nil, nil
}

func (a *DocumentAdapterForChat) FindSectionByCode(ctx context.Context, documentId int64, sectionCode string) (*cvo.GraphSection, error) {
	if documentId == 0 || utils.IsBlank(sectionCode) {
		return nil, nil
	}
	sections, err := a.ListSections(ctx, documentId)
	if err != nil || len(sections) == 0 {
		return nil, err
	}
	for _, s := range sections {
		if strings.EqualFold(strings.TrimSpace(s.NodeCode), strings.TrimSpace(sectionCode)) {
			return s, nil
		}
	}
	return nil, nil
}

func toGraphSection(r *entity.StructureNode) *cvo.GraphSection {
	if r == nil {
		return nil
	}
	return &cvo.GraphSection{
		NodeId:            r.ID,
		DocumentId:        r.DocumentId,
		ParseTaskId:       r.ParseTaskId,
		NodeNo:            r.NodeNo,
		Depth:             r.Depth,
		ParentNodeId:      r.ParentNodeId,
		PrevSiblingNodeId: r.PrevSiblingNodeId,
		NextSiblingNodeId: r.NextSiblingNodeId,
		NodeCode:          r.NodeCode,
		Title:             r.Title,
		AnchorText:        r.AnchorText,
		SectionPath:       r.SectionPath,
		CanonicalPath:     r.CanonicalPath,
		ContentText:       r.ContentText,
	}
}
