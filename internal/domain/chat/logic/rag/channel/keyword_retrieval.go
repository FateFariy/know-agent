package channel

import (
	"context"
	"fmt"
	"slices"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// KeywordRetrievalChannel 关键词检索通道
type KeywordRetrievalChannel struct {
	retriever KeywordRetriever
}

// NewKeywordRetrievalChannel 创建关键词检索通道
func NewKeywordRetrievalChannel(svcCtx *svc.ServiceContext, retriever KeywordRetriever) *KeywordRetrievalChannel {
	return &KeywordRetrievalChannel{
		retriever: retriever,
	}
}

// Name 返回通道名称
func (c *KeywordRetrievalChannel) Name() string {
	return enum.RetrievalChannelKeyword
}

// Retrieve 执行关键词检索
func (c *KeywordRetrievalChannel) Retrieve(ctx context.Context, input *rag.ExecutionInput) (*rag.RetrievalChannelResult, error) {
	query, err := NewDocumentRetrieve(c.Name(), input)
	if err != nil {
		return nil, err
	}

	if !query.Validate() {
		return nil, fmt.Errorf("invaild value")
	}

	docs, err := c.retriever.SearchByKeyword(ctx, query)
	if err != nil {
		logx.Errorf("关键词检索失败: question='%s', error=%v", query.Question, err)
		return nil, err
	}

	channel, _ := input.RequireChannel(c.Name())
	topScore := 1.0
	if len(docs) > 1 {
		cmp := func(a, b *vo.DocumentChunk) int { return int(a.Score - b.Score) }
		topScore = slices.MaxFunc(docs, cmp).Score
	}
	if topScore <= 0 {
		return &rag.RetrievalChannelResult{
			Name:         c.Name(),
			RawDocuments: docs,
		}, nil
	}
	acceptedFloor := topScore * channel.RelativeScoreFloor
	accepted := utils.Filter(docs, func(doc *vo.DocumentChunk) bool {
		return doc != nil && doc.Score >= acceptedFloor
	})

	return &rag.RetrievalChannelResult{
		Name:              c.Name(),
		RawDocuments:      docs,
		AcceptedDocuments: accepted,
	}, nil
}
