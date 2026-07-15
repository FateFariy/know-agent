package handler

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/api/knowledge"
	"github.com/swiftbit/know-agent/common/utils"
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

func (k *KnowledgeService) SaveKnowledgeScope(ctx context.Context, req *knowledge.KnowledgeScopeSaveReq) (*knowledge.KnowledgeScopeResp, error) {
	scopeNode := convert.FromKnowledgeScopeSaveReq(req)
	scopeNode, err := k.l.SaveScope(ctx, scopeNode)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeScopeResp(scopeNode), nil
}

func (k *KnowledgeService) DeleteKnowledgeScope(ctx context.Context, req *knowledge.KnowledgeScopeDeleteReq) (bool, error) {
	return k.l.DeleteScope(ctx, strutil.Trim(req.ScopeCode))
}

func (k *KnowledgeService) ListKnowledgeScope(ctx context.Context) ([]*knowledge.KnowledgeScopeResp, error) {
	nodes, err := k.l.ListScopes(ctx)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeScopeRespList(nodes), nil
}

// ==================== 主题 ====================

func (k *KnowledgeService) SaveKnowledgeTopic(ctx context.Context, req *knowledge.KnowledgeTopicSaveReq) (*knowledge.KnowledgeTopicResp, error) {
	topicNode := convert.FromKnowledgeTopicSaveReq(req)
	topicNode, err := k.l.SaveTopic(ctx, topicNode)
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeTopicResp(topicNode), nil
}

func (k *KnowledgeService) DeleteKnowledgeTopic(ctx context.Context, req *knowledge.KnowledgeTopicDeleteReq) (bool, error) {
	return k.l.DeleteTopic(ctx, strutil.Trim(req.TopicCode))
}

func (k *KnowledgeService) ListKnowledgeTopic(ctx context.Context, req *knowledge.KnowledgeTopicListReq) ([]*knowledge.KnowledgeTopicResp, error) {
	nodes, err := k.l.ListTopics(ctx, strutil.Trim(req.ScopeCode))
	if err != nil {
		return nil, err
	}
	return convert.ToKnowledgeTopicRespList(nodes), nil
}

// ==================== 主题-文档关联 ====================

// ListTopicDocumentRelation 列表主题文档关联
func (k *KnowledgeService) ListTopicDocumentRelation(ctx context.Context, req *knowledge.TopicDocumentRelationListReq) ([]*knowledge.TopicDocumentRelationResp, error) {
	relations, err := k.l.ListTopicDocumentRelations(ctx, req.TopicCode)
	if err != nil {
		return nil, err
	}
	return convert.ToTopicDocumentRelationRespList(relations), nil
}

// SaveTopicDocumentRelation 保存主题文档关联
func (k *KnowledgeService) SaveTopicDocumentRelation(ctx context.Context, req *knowledge.TopicDocumentRelationSaveReq) (*knowledge.TopicDocumentRelationResp, error) {
	relation := convert.FromKnowledgeTopicDocumentRelationSaveReq(req)
	relation, err := k.l.SaveTopicDocumentRelation(ctx, relation)
	if err != nil {
		return nil, err
	}
	return convert.ToTopicDocumentRelationResp(relation), nil
}

// RemoveTopicDocumentRelation 移除主题文档关联
func (k *KnowledgeService) RemoveTopicDocumentRelation(ctx context.Context, req *knowledge.TopicDocumentRelationRemoveReq) (bool, error) {
	return k.l.RemoveTopicDocumentRelation(ctx, req.TopicCode, utils.StringToInt64(req.DocumentId))
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
		Total:      total,
		TotalPages: int(totalPages),
		Records:    result,
	}, nil
}
