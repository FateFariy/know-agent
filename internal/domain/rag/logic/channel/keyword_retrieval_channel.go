package channel

import (
	"context"

	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	kl "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/rag/logic"
	"github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// KeywordRetrievalChannel 关键词检索通道
type KeywordRetrievalChannel struct {
	documentKnowledgeLogic kl.DocumentKnowledgeLogic
	keywordTopK            int
}

var _ logic.RetrievalChannel = (*KeywordRetrievalChannel)(nil)

// NewKeywordRetrievalChannel 创建关键词检索通道
func NewKeywordRetrievalChannel(svcCtx *svc.ServiceContext, documentKnowledgeLogic kl.DocumentKnowledgeLogic) *KeywordRetrievalChannel {
	return &KeywordRetrievalChannel{
		documentKnowledgeLogic: documentKnowledgeLogic,
		keywordTopK:            svcCtx.Config.Chat.Rag.KeywordTopK,
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
func (c *KeywordRetrievalChannel) Retrieve(ctx context.Context, subQuestion string, plan *cvo.ConversationExecutionPlan) (*logic.RetrievalChannelResult, error) {
	documentRetrieve := klvo.NewDocumentRetrieve(subQuestion, plan, c.keywordTopK)

	docs, err := c.documentKnowledgeLogic.KeywordSearch(ctx, documentRetrieve)
	if err != nil {
		Warnf("关键词检索失败: subQuestion='%s', error=%v", subQuestion, err)
		return nil, err
	}

	return &logic.RetrievalChannelResult{
		ChannelName: c.ChannelName(),
		Documents:   docs,
	}, nil
}
