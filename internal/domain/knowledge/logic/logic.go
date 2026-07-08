package logic

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// KnowledgeRouteLogic 知识路由服务接口
type KnowledgeRouteLogic interface {
	// Route 根据问题进行知识路由
	Route(ctx context.Context, question, rewriteQuestion string) (*vo.KnowledgeRouteDecision, error)

	// RecordAutoRoute 记录自动路由结果
	RecordAutoRoute(ctx context.Context, exchangeId int64, conversationId, question, rewriteQuestion string, decision *vo.KnowledgeRouteDecision) error

	// RecordShadowRoute 记录影子路由结果
	RecordShadowRoute(ctx context.Context, exchangeId, documentId int64, conversationId, question, rewriteQuestion string) error
}

// KnowledgeLogic 知识管理服务接口
type KnowledgeLogic interface {
	// SaveScope 保存/更新知识范围节点
	SaveScope(ctx context.Context, scopeNode *entity.KnowledgeScopeNode) (*entity.KnowledgeScopeNode, error)

	// DeleteScope 删除知识范围节点
	DeleteScope(ctx context.Context, scopeCode string) (bool, error)

	// ListScopes 查询知识范围列表
	ListScopes(ctx context.Context) ([]*entity.KnowledgeScopeNode, error)

	// SaveTopic 保存/更新主题节点
	SaveTopic(ctx context.Context, topicNode *entity.KnowledgeTopicNode) (*entity.KnowledgeTopicNode, error)

	// DeleteTopic 删除主题节点
	DeleteTopic(ctx context.Context, topicCode string) (bool, error)

	// ListTopics 查询主题列表（支持按 scopeCode 过滤）
	ListTopics(ctx context.Context, scopeCode string) ([]*entity.KnowledgeTopicNode, error)

	// GetDocumentProfile 获取文档画像
	GetDocumentProfile(ctx context.Context, documentId int64) (*entity.KnowledgeDocumentProfile, error)

	// RegenerateDocumentProfile 重新生成文档画像
	RegenerateDocumentProfile(ctx context.Context, documentId int64) (*entity.KnowledgeDocumentProfile, error)

	// BatchRegenerateDocumentProfiles 批量重新生成文档画像
	BatchRegenerateDocumentProfiles(ctx context.Context, documentIds []int64) ([]*entity.KnowledgeDocumentProfile, error)

	// ListTopicDocumentRelations 查询主题文档关联
	ListTopicDocumentRelations(ctx context.Context, topicCode string) ([]TopicDocumentRelationVo, error)

	// SaveTopicDocumentRelation 保存主题文档关联
	SaveTopicDocumentRelation(ctx context.Context, topicCode string, documentId int64, relationScore float64, relationSource, reason string) (*entity.KnowledgeTopicDocumentRelation, error)

	// RemoveTopicDocumentRelation 移除主题文档关联
	RemoveTopicDocumentRelation(ctx context.Context, topicCode string, documentId int64) (bool, error)

	// QueryRouteTracePage 分页查询知识路由追踪
	QueryRouteTracePage(ctx context.Context, conversationId, mode, routeStatus string, pageNo, pageSize int32) ([]*entity.KnowledgeRouteTrace, int64, error)
}

func Warnf(format string, v ...any) {
	logx.Alert(fmt.Sprintf(format, v...))
}
