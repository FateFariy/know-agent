package handler

import (
	"context"
	"strconv"

	"github.com/swiftbit/know-agent/api/knowledge"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
)

// KnowledgeService 知识管理 HTTP 服务
type KnowledgeService struct {
	l logic.KnowledgeLogic
}

var _ knowledge.HTTPServer = (*KnowledgeService)(nil)

// NewKnowledgeService 构造函数
func NewKnowledgeService(l logic.KnowledgeLogic) *KnowledgeService {
	return &KnowledgeService{l: l}
}

// ==================== 知识范围 ====================

func (k *KnowledgeService) SaveKnowledgeScope(ctx context.Context, req *knowledge.KnowledgeScopeSaveReq) (*knowledge.KnowledgeScopeItem, error) {
	scopeNode := convert.FromKnowledgeScopeSaveReq(req)
	scopeNode, err := k.l.SaveScope(ctx, scopeNode)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeScopeItem(scopeNode), nil
}

func (k *KnowledgeService) DeleteKnowledgeScope(ctx context.Context, req *knowledge.KnowledgeScopeDeleteReq) (bool, error) {
	return k.l.DeleteScope(ctx, req.ScopeCode)
}

func (k *KnowledgeService) ListKnowledgeScope(ctx context.Context) ([]*knowledge.KnowledgeScopeItem, error) {
	nodes, err := k.l.ListScopes(ctx)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeScopeItemList(nodes), nil
}

// ==================== 主题 ====================

func (k *KnowledgeService) SaveKnowledgeTopic(ctx context.Context, req *knowledge.KnowledgeTopicSaveReq) (*knowledge.KnowledgeTopicItem, error) {
	topicNode := convert.FromKnowledgeTopicSaveReq(req)
	topicNode, err := k.l.SaveTopic(ctx, topicNode)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeTopicItem(topicNode), nil
}

func (k *KnowledgeService) DeleteKnowledgeTopic(ctx context.Context, req *knowledge.KnowledgeTopicDeleteReq) (bool, error) {
	return k.l.DeleteTopic(ctx, req.TopicCode)
}

func (k *KnowledgeService) ListKnowledgeTopic(ctx context.Context, req *knowledge.KnowledgeTopicListReq) ([]*knowledge.KnowledgeTopicItem, error) {
	nodes, err := k.l.ListTopics(ctx, req.ScopeCode)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeTopicItemList(nodes), nil
}

// ==================== 文档画像 ====================

func (k *KnowledgeService) GetDocumentProfile(ctx context.Context, req *knowledge.DocumentProfileDetailReq) (*knowledge.DocumentProfileResp, error) {
	profile, err := k.l.GetDocumentProfile(ctx, req.DocumentId)
	if err != nil {
		return nil, err
	}
	return convert.ToDocumentProfileResp(profile), nil
}

func (k *KnowledgeService) RegenerateDocumentProfile(ctx context.Context, req *knowledge.DocumentProfileRegenerateReq) (*knowledge.DocumentProfileResp, error) {
	profile, err := k.l.RegenerateDocumentProfile(ctx, req.DocumentId)
	if err != nil {
		return nil, err
	}
	return convert.ToDocumentProfileResp(profile), nil
}

func (k *KnowledgeService) BatchRegenerateDocumentProfile(ctx context.Context, req *knowledge.DocumentProfileBatchRegenerateReq) ([]*knowledge.DocumentProfileResp, error) {
	profiles, err := k.l.BatchRegenerateDocumentProfiles(ctx, req.DocumentIds)
	if err != nil {
		return nil, err
	}
	return convert.ToDocumentProfileItemList(profiles), nil
}

// ==================== 主题-文档关联 ====================

func (k *KnowledgeService) ListTopicDocumentRelation(ctx context.Context, req *knowledge.TopicDocumentRelationListReq) ([]*knowledge.TopicDocumentRelationItem, error) {
	relations, err := k.l.ListTopicDocumentRelations(ctx, req.TopicCode)
	if err != nil {
		return nil, err
	}
	result := make([]*knowledge.TopicDocumentRelationItem, 0, len(relations))
	for _, r := range relations {
		result = append(result, &knowledge.TopicDocumentRelationItem{
			TopicCode:          r.TopicCode,
			DocumentId:         r.DocumentId,
			DocumentName:       r.DocumentName,
			KnowledgeScopeCode: r.KnowledgeScopeCode,
			KnowledgeScopeName: r.KnowledgeScopeName,
			BusinessCategory:   r.BusinessCategory,
			DocumentTags:       r.DocumentTags,
			RelationScore:      r.RelationScore,
			RelationSource:     r.RelationSource,
			Reason:             r.Reason,
		})
	}
	return result, nil
}

func (k *KnowledgeService) SaveTopicDocumentRelation(ctx context.Context, req *knowledge.TopicDocumentRelationSaveReq) (*knowledge.TopicDocumentRelationItem, error) {
	rel, err := k.l.SaveTopicDocumentRelation(ctx, req.TopicCode, req.DocumentId, req.RelationScore, req.RelationSource, req.Reason)
	if err != nil {
		return nil, err
	}
	return &knowledge.TopicDocumentRelationItem{
		TopicCode:      rel.TopicCode,
		DocumentId:     rel.DocumentId,
		RelationScore:  rel.RelationScore,
		RelationSource: rel.RelationSource,
		Reason:         rel.Reason,
	}, nil
}

func (k *KnowledgeService) RemoveTopicDocumentRelation(ctx context.Context, req *knowledge.TopicDocumentRelationRemoveReq) (bool, error) {
	return k.l.RemoveTopicDocumentRelation(ctx, req.TopicCode, req.DocumentId)
}

// ==================== 路由追踪 ====================

func (k *KnowledgeService) QueryKnowledgeRouteTracePage(ctx context.Context, req *knowledge.KnowledgeRouteTracePageReq) (*knowledge.KnowledgeRouteTracePageResp, error) {
	traces, total, err := k.l.QueryRouteTracePage(ctx, req.ConversationId, req.Mode, req.RouteStatus, req.PageNo, req.PageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*knowledge.KnowledgeRouteTraceItem, 0, len(traces))
	for _, t := range traces {
		items = append(items, toKnowledgeRouteTraceItem(t))
	}
	totalPages := int32((total + int64(req.PageSize) - 1) / int64(req.PageSize))
	return &knowledge.KnowledgeRouteTracePageResp{
		PageNo:     req.PageNo,
		PageSize:   req.PageSize,
		TotalSize:  total,
		TotalPages: totalPages,
		Records:    items,
	}, nil
}

// ==================== 转换函数 ====================

func toKnowledgeScopeItem(n *entity.KnowledgeScopeNode) *knowledge.KnowledgeScopeItem {
	if n == nil {
		return nil
	}
	return &knowledge.KnowledgeScopeItem{
		Id:              int64ToString(n.ID),
		ScopeCode:       n.ScopeCode,
		ScopeName:       n.ScopeName,
		ParentScopeCode: n.ParentScopeCode,
		Description:     n.Description,
		Aliases:         n.Aliases,
		Examples:        n.Examples,
		SortOrder:       int32(n.SortOrder),
	}
}

func toKnowledgeTopicItem(n *entity.KnowledgeTopicNode) *knowledge.KnowledgeTopicItem {
	if n == nil {
		return nil
	}
	return &knowledge.KnowledgeTopicItem{
		Id:                  int64ToString(n.ID),
		TopicCode:           n.TopicCode,
		TopicName:           n.TopicName,
		ScopeCode:           n.ScopeCode,
		Description:         n.Description,
		Aliases:             n.Aliases,
		Examples:            n.Examples,
		AnswerShape:         n.AnswerShape,
		ExecutionPreference: n.ExecutionPreference,
		SortOrder:           int32(n.SortOrder),
	}
}

func toDocumentProfileResp(p *entity.KnowledgeDocumentProfile) *knowledge.DocumentProfileResp {
	if p == nil {
		return nil
	}
	return &knowledge.DocumentProfileResp{
		DocumentId:           p.DocumentId,
		DocumentSummary:      p.DocumentSummary,
		DocumentType:         p.DocumentType,
		CoreTopics:           p.CoreTopics,
		ExampleQuestions:     p.ExampleQuestions,
		GraphFriendly:        logic.RouteStatusName(p.ProfileStatus),
		SupportsGraphOutline: logic.RouteStatusName(p.ProfileStatus),
		SupportsItemLookup:   logic.RouteStatusName(p.ProfileStatus),
		SupportsGraphAssist:  logic.RouteStatusName(p.ProfileStatus),
		ProfileSource:        "MANUAL",
		ProfileStatus:        logic.RouteStatusName(p.ProfileStatus),
	}
}

func toKnowledgeRouteTraceItem(t *entity.KnowledgeRouteTrace) *knowledge.KnowledgeRouteTraceItem {
	if t == nil {
		return nil
	}
	return &knowledge.KnowledgeRouteTraceItem{
		Id:                  t.ID,
		ConversationId:      t.ConversationId,
		ExchangeId:          t.ExchangeId,
		Question:            t.Question,
		RewriteQuestion:     t.RewriteQuestion,
		Mode:                t.Mode,
		TopScopesJson:       t.TopScopesJson,
		TopTopicsJson:       t.TopTopicsJson,
		TopDocumentsJson:    t.TopDocumentsJson,
		SelectedDocumentId:  t.SelectedDocumentId,
		HitSelectedDocument: logic.RouteStatusHitFlag(t.HitSelectedDocument),
		Confidence:          float64ToPctString(t.Confidence),
		RouteStatus:         logic.RouteStatusName(t.RouteStatus),
		ErrorMsg:            t.ErrorMsg,
	}
}

// ========= 通用工具 =========

func int64ToString(v int64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatInt(v, 10)
}

func float64ToPctString(v float64) string {
	if v <= 0 {
		return ""
	}
	if v <= 1 {
		return strconv.FormatFloat(v*100, 'f', 2, 64) + "%"
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}
