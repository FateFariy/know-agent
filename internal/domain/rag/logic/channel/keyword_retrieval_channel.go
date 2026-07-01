package channel

import (
	"context"

	"github.com/swiftbit/know-agent/common/utils"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/rag/adapter"
	"github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// KeywordRetrievalChannel 关键词检索通道
type KeywordRetrievalChannel struct {
	repo      adapter.RagRepository
	keywordDB adapter.KeywordDB
}

var _ RetrievalChannel = (*KeywordRetrievalChannel)(nil)

// NewKeywordRetrievalChannel 创建关键词检索通道
func NewKeywordRetrievalChannel(svcCtx *svc.ServiceContext, repo adapter.RagRepository, keywordDB adapter.KeywordDB) *KeywordRetrievalChannel {
	return &KeywordRetrievalChannel{
		repo:      repo,
		keywordDB: keywordDB,
	}
}

// ChannelName 返回通道名称
func (c *KeywordRetrievalChannel) ChannelName() string {
	return vo.RetrievalChannelKeyword
}

// Supports 判断是否支持该执行计划
func (c *KeywordRetrievalChannel) Supports(plan *cvo.ConversationExecutionPlan) bool {
	return plan.SelectedDocumentId != 0
}

// Retrieve 执行关键词检索
func (c *KeywordRetrievalChannel) Retrieve(ctx context.Context, query *vo.DocumentRetrieve) (*vo.RetrievalChannelResult, error) {
	if !query.ValidSearchable() {
		return nil, nil
	}

	docs, err := c.keywordDB.SearchByKeyword(ctx, query)
	if err != nil {
		Warnf("关键词检索失败: question='%s', error=%v", query.Question, err)
		return nil, err
	}
	knowledgeMap, err := c.getDocumentsMap(ctx, query.DocumentIds)
	if err != nil {
		return nil, err
	}

	for _, document := range docs {
		document.FillKnowledge(knowledgeMap[document.DocumentId])
	}

	return &vo.RetrievalChannelResult{
		ChannelName: c.ChannelName(),
		Documents:   docs,
	}, nil
}

// getDocumentsMap 获取文档描述符到 documentId 的映射
func (c *KeywordRetrievalChannel) getDocumentsMap(ctx context.Context, documentIds []int64) (map[int64]*vo.KnowledgeDocument, error) {
	documents, err := c.repo.SelectRetrievableDocuments(ctx, documentIds...)
	if err != nil {
		return nil, err
	}
	descriptorMap := utils.SliceToMapBy(documents, func(t *vo.KnowledgeDocument) (int64, *vo.KnowledgeDocument) {
		return t.DocumentId, t
	})
	return descriptorMap, nil
}
