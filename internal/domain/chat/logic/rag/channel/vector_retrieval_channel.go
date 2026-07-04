package channel

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/model/vo"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// VectorRetrievalChannel 向量检索通道
type VectorRetrievalChannel struct {
	vectorDB adapter.VectorDB
	baseRetrievalChannel
}

var _ RetrievalChannel = (*VectorRetrievalChannel)(nil)

// NewVectorRetrievalChannel 创建向量检索通道
func NewVectorRetrievalChannel(svcCtx *svc.ServiceContext, repo adapter.RagRepository, vectorDB adapter.VectorDB) *VectorRetrievalChannel {
	return &VectorRetrievalChannel{
		baseRetrievalChannel: baseRetrievalChannel{repo: repo},
		vectorDB:             vectorDB,
	}
}

// ChannelName 返回通道名称
func (c *VectorRetrievalChannel) ChannelName() string {
	return vo.RetrievalChannelVector
}

// Supports 判断是否支持该执行计划
func (c *VectorRetrievalChannel) Supports(plan *cvo.ConversationExecutionPlan) bool {
	return plan.SelectedDocumentId != 0
}

// Retrieve 执行向量检索
// 流程：参数校验 → 构建描述符 map → 调用 Milvus 向量相似度查询（topK + 过滤）
func (c *VectorRetrievalChannel) Retrieve(ctx context.Context, query *vo.DocumentRetrieve) (*vo.RetrievalChannelResult, error) {
	if !query.ValidSearchable() {
		return nil, nil
	}

	knowledgeMap, err := c.getDocumentsMap(ctx, query.DocumentIds)
	if err != nil {
		return nil, err
	}

	docs, err := c.vectorDB.SearchByVector(ctx, query)
	if err != nil {
		Warnf("向量检索失败: question='%s', error=%v", query.Question, err)
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
