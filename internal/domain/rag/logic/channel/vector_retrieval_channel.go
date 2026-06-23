package channel

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	kl "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/rag/adapter"
	"github.com/swiftbit/know-agent/internal/domain/rag/logic"
	"github.com/swiftbit/know-agent/internal/svc"
)

// VectorRetrievalChannel 向量检索通道
type VectorRetrievalChannel struct {
	repo                   adapter.RagRepository
	documentKnowledgeLogic kl.DocumentKnowledgeLogic
	vectorTopK             int
}

var _ logic.RetrievalChannel = (*VectorRetrievalChannel)(nil)

// NewVectorRetrievalChannel 创建向量检索通道
func NewVectorRetrievalChannel(svcCtx *svc.ServiceContext, repo adapter.RagRepository, documentKnowledgeLogic kl.DocumentKnowledgeLogic) *VectorRetrievalChannel {
	return &VectorRetrievalChannel{
		repo:                   repo,
		documentKnowledgeLogic: documentKnowledgeLogic,
		vectorTopK:             svcCtx.Config.Chat.Rag.VectorTopK,
	}
}

// ChannelName 返回通道名称
func (c *VectorRetrievalChannel) ChannelName() string {
	return "vector"
}

// Supports 判断是否支持该执行计划
func (c *VectorRetrievalChannel) Supports(plan *cvo.ConversationExecutionPlan) bool {
	return plan.SelectedDocumentId != 0
}

// Retrieve 执行向量检索
func (c *VectorRetrievalChannel) Retrieve(ctx context.Context, subQuestion string, plan *cvo.ConversationExecutionPlan) (*logic.RetrievalChannelResult, error) {
	documentRetrieve := klvo.NewDocumentRetrieve(subQuestion, plan, c.vectorTopK)

	docs, err := c.documentKnowledgeLogic.VectorSearch(ctx, documentRetrieve)
	if err != nil {
		Warnf("向量检索失败: subQuestion='%s', error=%v", subQuestion, err)
		return nil, err
	}

	return &logic.RetrievalChannelResult{
		ChannelName: c.ChannelName(),
	}, nil
}

func Warnf(format string, v ...any) {
	logx.Alert(fmt.Sprintf(format, v...))
}
