package gateway

import (
	"context"

	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route"
)

// KnowledgeAdapter 知识库解析器实现
type KnowledgeAdapter struct {
	l      logic.KnowledgeBaseRetrievalLogic
	router route.KnowledgeRouter
}

// NewKnowledgeAdapter 创建知识库解析器
func NewKnowledgeAdapter(l logic.KnowledgeBaseRetrievalLogic, router route.KnowledgeRouter) *KnowledgeAdapter {
	return &KnowledgeAdapter{
		l:      l,
		router: router,
	}
}

// DetermineKnowledgeScope 根据聊天模式和知识库选择模式解析检索范围
func (r *KnowledgeAdapter) DetermineKnowledgeScope(ctx context.Context, chatMode, selectMode string, kbIds []string) (*vo.KnowledgeBaseSelectionSnapshot, error) {
	snapshot, err := r.l.DetermineKnowledgeScope(ctx, chatMode, selectMode, kbIds)
	return convert.ToKnowledgeBaseSelectionSnapshot(snapshot), err
}

func (r *KnowledgeAdapter) Route(ctx context.Context, input *conversation.KnowledgeRouteInput) (*vo.KnowledgeRouteDecision, error) {
	routeCtx := ToRouteContext(input)
	decision, err := r.router.Route(ctx, routeCtx)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeRouteDecision(decision), nil
}

func (r *KnowledgeAdapter) RecordShadowRoute(ctx context.Context, input *conversation.KnowledgeRouteInput) error {
	routeCtx := ToRouteContext(input)
	return r.router.RecordShadowRoute(ctx, routeCtx)
}

func ToRouteContext(input *conversation.KnowledgeRouteInput) *route.Context {
	routeCtx := route.NewRouteContext(input.Question, input.RewriteQuestion, input.SelectedKnowledgeBaseIds, input.AllowedDocumentIds)
	routeCtx.ConversationId = input.ConversationId
	routeCtx.ExchangeId = input.ExchangeId
	routeCtx.SelectedDocumentId = input.SelectedDocumentId
	routeCtx.KnowledgeBaseSelectionMode = input.KnowledgeBaseSelectionMode
	routeCtx.SelectedKnowledgeBaseNames = input.SelectedKnowledgeBaseNames
	return routeCtx
}
