package channel

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// VectorRetrievalChannel 向量检索通道
type VectorRetrievalChannel struct {
	retriever adapter.VectorRetriever
}

var _ rag.RetrievalChannel = (*VectorRetrievalChannel)(nil)

// NewVectorRetrievalChannel 创建向量检索通道
func NewVectorRetrievalChannel(svcCtx *svc.ServiceContext, retriever adapter.VectorRetriever) *VectorRetrievalChannel {
	return &VectorRetrievalChannel{
		retriever: retriever,
	}
}

// ChannelName 返回通道名称
func (c *VectorRetrievalChannel) ChannelName() string {
	return vo.RetrievalChannelVector
}

// Supports 判断是否支持该执行计划
func (c *VectorRetrievalChannel) Supports(plan *vo.ConversationExecutionPlan) bool {
	return plan.SelectedDocumentId != 0
}

// Retrieve 执行向量检索
// 流程：参数校验 → 构建描述符 map → 调用 Milvus 向量相似度查询（topK + 过滤）
func (c *VectorRetrievalChannel) Retrieve(ctx context.Context, query *vo.DocumentRetrieve) (*vo.RetrievalChannelResult, error) {
	if !query.ValidSearchable() {
		return nil, fmt.Errorf("invaild value")
	}

	docs, err := c.retriever.SearchByVector(ctx, query)
	if err != nil {
		logx.Errorf("向量检索失败: question='%s', error=%v", query.Question, err)
		return nil, err
	}

	return &vo.RetrievalChannelResult{
		ChannelName: c.ChannelName(),
		Documents:   docs,
	}, nil
}
