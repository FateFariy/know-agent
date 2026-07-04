package channel

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// KeywordRetrievalChannel 关键词检索通道
type KeywordRetrievalChannel struct {
	keywordDB adapter.KeywordDB
}

var _ RetrievalChannel = (*KeywordRetrievalChannel)(nil)

// NewKeywordRetrievalChannel 创建关键词检索通道
func NewKeywordRetrievalChannel(svcCtx *svc.ServiceContext, keywordDB adapter.KeywordDB) *KeywordRetrievalChannel {
	return &KeywordRetrievalChannel{
		keywordDB: keywordDB,
	}
}

// ChannelName 返回通道名称
func (c *KeywordRetrievalChannel) ChannelName() string {
	return vo.RetrievalChannelKeyword
}

// Supports 判断是否支持该执行计划
func (c *KeywordRetrievalChannel) Supports(plan *vo.ConversationExecutionPlan) bool {
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

	return &vo.RetrievalChannelResult{
		ChannelName: c.ChannelName(),
		Documents:   docs,
	}, nil
}
