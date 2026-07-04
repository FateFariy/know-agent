package channel

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/model/vo"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// RetrievalChannel 检索通道接口
type RetrievalChannel interface {
	// ChannelName 检索通道名称
	ChannelName() string

	// Supports 是否支持该执行计划
	Supports(plan *cvo.ConversationExecutionPlan) bool

	// Retrieve 根据子问题检索
	Retrieve(ctx context.Context, query *vo.DocumentRetrieve) (*vo.RetrievalChannelResult, error)
}

type baseRetrievalChannel struct {
	repo adapter.RagRepository
}

// getDocumentsMap 获取文档描述符到 documentId 的映射
func (c *baseRetrievalChannel) getDocumentsMap(ctx context.Context, documentIds []int64) (map[int64]*vo.KnowledgeDocument, error) {
	documents, err := c.repo.SelectRetrievableDocuments(ctx, documentIds...)
	if err != nil {
		return nil, err
	}
	descriptorMap := utils.SliceToMapBy(documents, func(t *vo.KnowledgeDocument) (int64, *vo.KnowledgeDocument) {
		return t.DocumentId, t
	})
	return descriptorMap, nil
}

func Warnf(format string, v ...any) {
	logx.Alert(fmt.Sprintf(format, v...))
}
