package gateway

import (
	"context"

	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/config"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// KnowledgeAdapter 知识库解析器实现
type KnowledgeAdapter struct {
	l        logic.KnowledgeBaseRetrievalLogic
	repo     adapter.KnowledgeRepository
	resolver *config.Resolver
	router   route.KnowledgeRouter
}

// NewKnowledgeAdapter 创建知识库解析器
func NewKnowledgeAdapter(l logic.KnowledgeBaseRetrievalLogic, repo adapter.KnowledgeRepository, router route.KnowledgeRouter, global config.GlobalConfigProvider) *KnowledgeAdapter {
	return &KnowledgeAdapter{
		l:        l,
		repo:     repo,
		router:   router,
		resolver: config.NewResolver(global),
	}
}

// DetermineKnowledgeScope 根据知识库选择模式解析检索范围
func (r *KnowledgeAdapter) DetermineKnowledgeScope(ctx context.Context, selectMode string, kbIds []string) (*cvo.KnowledgeBaseSelectionSnapshot, error) {
	snapshot, err := r.l.DetermineKnowledgeScope(ctx, selectMode, kbIds)
	return convert.ToKnowledgeBaseSelectionSnapshot(snapshot), err
}

func (r *KnowledgeAdapter) Route(ctx context.Context, input *conversation.KnowledgeRouteInput) (*cvo.KnowledgeRouteDecision, error) {
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

func (r *KnowledgeAdapter) Resolve(ctx context.Context, document *den.Document) *dvo.IndexingOptions {
	var options *vo.IndexingOptions
	if document == nil || document.KnowledgeBaseId == 0 {
		options = r.resolver.ResolveIndexingOptions(nil)
	} else {
		base, _ := r.repo.SelectKnowledgeBaseById(ctx, document.KnowledgeBaseId)
		options = r.resolver.ResolveIndexingOptions([]*entity.KnowledgeBase{base})
	}
	return convert.ToIndexingOptions(options)
}

func (r *KnowledgeAdapter) RequireEnabled(ctx context.Context, knowledgeBaseId int64) (*den.KnowledgeBase, error) {
	base, err := r.repo.SelectKnowledgeBaseById(ctx, knowledgeBaseId)
	if err != nil {
		return nil, err
	}
	return &den.KnowledgeBase{
		ID:       base.ID,
		BaseName: base.BaseName,
	}, nil
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
