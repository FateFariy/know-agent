package logic

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/req"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// DocumentKnowledgeLogicImpl 文档知识服务
type DocumentKnowledgeLogicImpl struct {
}

// NewDocumentKnowledgeService 创建文档知识服务实例
func NewDocumentKnowledgeService() *DocumentKnowledgeLogicImpl {
	return &DocumentKnowledgeLogicImpl{}
}

// ListRetrievableDocuments 获取可检索的文档列表
func (s *DocumentKnowledgeLogicImpl) ListRetrievableDocuments(ctx context.Context) ([]*vo.KnowledgeDocumentDescriptor, error) {
	return []*vo.KnowledgeDocumentDescriptor{
		{
			DocumentId:         1,
			DocumentName:       "产品手册.pdf",
			LastIndexTaskId:    1001,
			KnowledgeScopeCode: "PRODUCT",
			KnowledgeScopeName: "产品知识",
			BusinessCategory:   "产品",
			DocumentTags:       "产品,手册",
		},
		{
			DocumentId:         2,
			DocumentName:       "技术文档.docx",
			LastIndexTaskId:    1002,
			KnowledgeScopeCode: "TECH",
			KnowledgeScopeName: "技术知识",
			BusinessCategory:   "技术",
			DocumentTags:       "技术,开发",
		},
	}, nil
}

// VectorSearch 向量检索
func (s *DocumentKnowledgeLogicImpl) VectorSearch(ctx context.Context, request *req.DocumentRetrieveRequest) ([]*vo.SearchDocument, error) {
	return []*vo.SearchDocument{
		{
			ID:      "doc-001",
			Content: "这是向量检索到的相关文档内容...",
			Meta: map[string]interface{}{
				"documentId":   1,
				"documentName": "产品手册.pdf",
				"chunkId":      10,
				"sectionPath":  "/章节1/小节A",
			},
			Score: 0.92,
		},
	}, nil
}

// KeywordSearch 关键词检索
func (s *DocumentKnowledgeLogicImpl) KeywordSearch(ctx context.Context, request *req.DocumentRetrieveRequest) ([]*vo.SearchDocument, error) {
	return []*vo.SearchDocument{
		{
			ID:      "doc-002",
			Content: "这是关键词检索到的相关文档内容...",
			Meta: map[string]interface{}{
				"documentId":   2,
				"documentName": "技术文档.docx",
				"chunkId":      25,
				"sectionPath":  "/技术说明/实现细节",
			},
			Score: 0.85,
		},
	}, nil
}

// ElevateToParentBlocks 将子文档提升到父块级别
func (s *DocumentKnowledgeLogicImpl) ElevateToParentBlocks(ctx context.Context, childDocuments []*vo.SearchDocument, maxChars int) ([]*vo.SearchDocument, error) {
	result := make([]*vo.SearchDocument, 0, len(childDocuments))
	for _, doc := range childDocuments {
		elevatedContent := doc.Content
		if len(elevatedContent) > maxChars {
			elevatedContent = elevatedContent[:maxChars] + "..."
		}
		result = append(result, &vo.SearchDocument{
			ID:      doc.ID,
			Content: elevatedContent,
			Meta:    doc.Meta,
			Score:   doc.Score,
		})
	}
	return result, nil
}
