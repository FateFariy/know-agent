package channel

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// VectorRetrievalChannel 向量检索通道
type VectorRetrievalChannel struct {
	retriever VectorRetriever
}

// NewVectorRetrievalChannel 创建向量检索通道
func NewVectorRetrievalChannel(svcCtx *svc.ServiceContext, retriever VectorRetriever) *VectorRetrievalChannel {
	return &VectorRetrievalChannel{
		retriever: retriever,
	}
}

// Name 返回通道名称
func (c *VectorRetrievalChannel) Name() string {
	return enum.RetrievalChannelVector
}

// Retrieve 执行向量检索
// 流程：参数校验 → 构建描述符 map → 调用 Milvus 向量相似度查询（topK + 过滤）
func (c *VectorRetrievalChannel) Retrieve(ctx context.Context, input *rag.ExecutionInput) (*rag.RetrievalChannelResult, error) {
	query, err := rag.NewDocumentRetrieve(input, c.Name())
	if err != nil {
		return nil, err
	}

	if !query.Validate() {
		return nil, fmt.Errorf("invaild value")
	}

	docs, err := c.retriever.SearchByVector(ctx, query)
	if err != nil {
		logx.Errorf("向量检索失败: question='%s', error=%v", query.Question, err)
		return nil, err
	}

	channel, _ := input.RequireChannel(c.Name())
	accepted := utils.Filter(docs, func(doc *vo.DocumentChunk) bool {
		return doc != nil && doc.Score >= channel.MinimumScore
	})

	return &rag.RetrievalChannelResult{
		Name:              c.Name(),
		RawDocuments:      docs,
		AcceptedDocuments: accepted,
	}, nil
}
