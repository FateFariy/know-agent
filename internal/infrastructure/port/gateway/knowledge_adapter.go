package gateway

import (
	"context"

	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route"
)

// KnowledgeBaseResolverImpl 知识库解析器实现
type KnowledgeBaseResolverImpl struct {
	l      logic.KnowledgeBaseRetrievalLogic
	router route.KnowledgeRouter
}

// NewKnowledgeBaseResolverImpl 创建知识库解析器
func NewKnowledgeBaseResolverImpl(l logic.KnowledgeBaseRetrievalLogic, router route.KnowledgeRouter) *KnowledgeBaseResolverImpl {
	return &KnowledgeBaseResolverImpl{
		l:      l,
		router: router,
	}
}

// DetermineKnowledgeScope 根据聊天模式和知识库选择模式解析检索范围
func (r *KnowledgeBaseResolverImpl) DetermineKnowledgeScope(ctx context.Context, chatMode, selectMode string, kbIds []string) (*vo.KnowledgeBaseSelectionSnapshot, error) {
	snapshot, err := r.l.DetermineKnowledgeScope(ctx, chatMode, selectMode, kbIds)
	return convert.ToKnowledgeBaseSelectionSnapshot(snapshot), err
}

func (r *KnowledgeBaseResolverImpl) Route(ctx context.Context, input *conversation.RouteInput) (*vo.KnowledgeRouteDecision, error) {
	routeCtx := ToRouteContext(input)
	decision, err := r.router.Route(ctx, routeCtx)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeRouteDecision(decision), nil
}

func (r *KnowledgeBaseResolverImpl) RecordShadowRoute(ctx context.Context, input *conversation.RouteInput) error {
	routeCtx := ToRouteContext(input)
	return r.router.RecordShadowRoute(ctx, routeCtx)
}

func ToRouteContext(input *conversation.RouteInput) *route.Context {
	routeCtx := route.NewRouteContext(input.Question, input.RewriteQuestion, input.SelectedKnowledgeBaseIds, input.AllowedDocumentIds)
	routeCtx.ConversationId = input.ConversationId
	routeCtx.ExchangeId = input.ExchangeId
	routeCtx.SelectedDocumentId = input.SelectedDocumentId
	routeCtx.KnowledgeBaseSelectionMode = input.KnowledgeBaseSelectionMode
	routeCtx.SelectedKnowledgeBaseNames = input.SelectedKnowledgeBaseNames
	return routeCtx
}
