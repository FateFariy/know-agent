package handler

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/api/knowledge"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
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
	return k.l.DeleteScope(ctx, strutil.Trim(req.ScopeCode))
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
	return k.l.DeleteTopic(ctx, strutil.Trim(req.TopicCode))
}

func (k *KnowledgeService) ListKnowledgeTopic(ctx context.Context, req *knowledge.KnowledgeTopicListReq) ([]*knowledge.KnowledgeTopicItem, error) {
	nodes, err := k.l.ListTopics(ctx, strutil.Trim(req.ScopeCode))
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

// ListTopicDocumentRelation 列表主题文档关联
func (k *KnowledgeService) ListTopicDocumentRelation(ctx context.Context, req *knowledge.TopicDocumentRelationListReq) ([]*knowledge.TopicDocumentRelationItem, error) {
	relations, err := k.l.ListTopicDocumentRelations(ctx, req.TopicCode)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeTopicDocumentRelationItemList(relations), nil
}

// SaveTopicDocumentRelation 保存主题文档关联
func (k *KnowledgeService) SaveTopicDocumentRelation(ctx context.Context, req *knowledge.TopicDocumentRelationSaveReq) (*knowledge.TopicDocumentRelationItem, error) {
	relation := convert.FromKnowledgeTopicDocumentRelationSaveReq(req)
	relation, err := k.l.SaveTopicDocumentRelation(ctx, relation)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeTopicDocumentRelationItem(relation), nil
}

// RemoveTopicDocumentRelation 移除主题文档关联
func (k *KnowledgeService) RemoveTopicDocumentRelation(ctx context.Context, req *knowledge.TopicDocumentRelationRemoveReq) (bool, error) {
	return k.l.RemoveTopicDocumentRelation(ctx, req.TopicCode, req.DocumentId)
}

// ==================== 路由追踪 ====================

func (k *KnowledgeService) QueryKnowledgeRouteTracePage(ctx context.Context, req *knowledge.KnowledgeRouteTracePageReq) (*knowledge.KnowledgeRouteTracePageResp, error) {
	traces, total, err := k.l.QueryRouteTracePage(ctx, req.ConversationId, req.Mode, req.RouteStatus, req.PageNo, req.PageSize)
	if err != nil {
		return nil, err
	}
	result := convert.ToKnowledgeRouteTraceItemList(traces)
	totalPages := (total + int64(req.PageSize) - 1) / int64(req.PageSize)
	return &knowledge.KnowledgeRouteTracePageResp{
		PageNo:     req.PageNo,
		PageSize:   req.PageSize,
		TotalSize:  total,
		TotalPages: int(totalPages),
		Records:    result,
	}, nil
}
