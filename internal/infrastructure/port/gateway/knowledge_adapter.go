package gateway

import (
	"context"

	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
)

// KnowledgeBaseResolverImpl 知识库解析器实现
type KnowledgeBaseResolverImpl struct {
	l logic.KnowledgeBaseRetrievalLogic
}

// NewKnowledgeBaseResolverImpl 创建知识库解析器
func NewKnowledgeBaseResolverImpl(l logic.KnowledgeBaseRetrievalLogic) *KnowledgeBaseResolverImpl {
	return &KnowledgeBaseResolverImpl{
		l: l,
	}
}

// Resolve 根据聊天模式和知识库选择模式解析检索范围
func (r *KnowledgeBaseResolverImpl) Resolve(ctx context.Context, chatMode, kbSelectionMode string,
	selectedKnowledgeBaseIds []string) (*vo.KnowledgeBaseSelectionSnapshot, error) {
	snapshot, err := r.l.Resolve(ctx, chatMode, kbSelectionMode, selectedKnowledgeBaseIds)
	return convert.ToKnowledgeBaseSelectionSnapshot(snapshot), err
}
