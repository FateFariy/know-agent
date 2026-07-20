package persistence

import (
	"context"
	"errors"

	"github.com/duke-git/lancet/v2/strutil"
	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/infrastructure/model"
	"github.com/swiftbit/know-agent/internal/svc"
)

// KnowledgeRepositoryImpl 文档知识仓储实现
type KnowledgeRepositoryImpl struct {
	*transactionManager
}

var _ adapter.KnowledgeRepository = (*KnowledgeRepositoryImpl)(nil)

// NewKnowledgeRepository 构造函数
func NewKnowledgeRepository(svcCtx *svc.ServiceContext) *KnowledgeRepositoryImpl {
	return &KnowledgeRepositoryImpl{
		transactionManager: &transactionManager{db: svcCtx.Db},
	}
}

// ============ 知识范围节点 ============

// SelectKnowledgeScopeNodes 查询所有知识范围节点
func (k *KnowledgeRepositoryImpl) SelectKnowledgeScopeNodes(ctx context.Context) ([]*entity.KnowledgeScopeNode, error) {
	var nodes []*entity.KnowledgeScopeNode
	if err := k.dbWithContext(ctx).Model(&model.KnowledgeScopeNode{}).
		Order("sort_order ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// UpsertKnowledgeScopeNode 插入或更新知识范围节点
func (k *KnowledgeRepositoryImpl) UpsertKnowledgeScopeNode(ctx context.Context, node *entity.KnowledgeScopeNode) error {
	nodeModel := convert.ToKnowledgeScopeNodeModel(node)
	if err := k.dbWithContext(ctx).Model(&model.KnowledgeScopeNode{}).Where("scope_code = ?", node.ScopeCode).First(node).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return k.dbWithContext(ctx).Create(nodeModel).Error
	}
	nodeModel.ID = node.ID
	return k.dbWithContext(ctx).Updates(nodeModel).Error
}

// DeleteKnowledgeScopeNode 删除知识范围节点
func (k *KnowledgeRepositoryImpl) DeleteKnowledgeScopeNode(ctx context.Context, scopeCode string) error {
	if strutil.IsBlank(scopeCode) {
		return nil
	}
	return k.dbWithContext(ctx).Where("scope_code = ?", scopeCode).Delete(&model.KnowledgeScopeNode{}).Error
}

// ============ 主题节点 ============

// SelectKnowledgeTopicNodes 查询所有主题节点
func (k *KnowledgeRepositoryImpl) SelectKnowledgeTopicNodes(ctx context.Context) ([]*entity.KnowledgeTopicNode, error) {
	return k.SelectKnowledgeTopicNodesByScopeCode(ctx, "")
}

// SelectKnowledgeTopicNodesByScopeCode 根据知识范围节点查询所有主题节点
func (k *KnowledgeRepositoryImpl) SelectKnowledgeTopicNodesByScopeCode(ctx context.Context, scopeCode string) ([]*entity.KnowledgeTopicNode, error) {
	var nodes []*entity.KnowledgeTopicNode
	builder := k.dbWithContext(ctx).Model(&model.KnowledgeTopicNode{}).Order("sort_order ASC")
	if strutil.IsNotBlank(scopeCode) {
		builder = builder.Where("scope_code = ?", scopeCode)
	}
	if err := builder.Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// UpsertKnowledgeTopicNode 插入或更新主题节点
func (k *KnowledgeRepositoryImpl) UpsertKnowledgeTopicNode(ctx context.Context, node *entity.KnowledgeTopicNode) error {
	nodeModel := convert.ToKnowledgeTopicNodeModel(node)
	if err := k.dbWithContext(ctx).Model(&model.KnowledgeTopicNode{}).
		Where("topic_code = ?", node.TopicCode).
		First(node).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return k.dbWithContext(ctx).Create(nodeModel).Error
	}
	nodeModel.ID = node.ID
	return k.dbWithContext(ctx).Updates(nodeModel).Error
}

// DeleteKnowledgeTopicNode 删除主题节点
func (k *KnowledgeRepositoryImpl) DeleteKnowledgeTopicNode(ctx context.Context, topicCode string) error {
	if strutil.IsBlank(topicCode) {
		return nil
	}
	return k.dbWithContext(ctx).Model(&model.KnowledgeTopicNode{}).Where("topic_code = ?", topicCode).Delete(nil).Error
}

// ============ 主题-文档关系 ============

// SelectTopicDocumentRelations 查询所有主题-文档关系
func (k *KnowledgeRepositoryImpl) SelectTopicDocumentRelations(ctx context.Context) ([]*entity.KnowledgeTopicDocumentRelation, error) {
	var relations []*entity.KnowledgeTopicDocumentRelation
	if err := k.dbWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{}).Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}

// SelectTopicDocumentRelationsByTopicCode 根据主题编码查询主题-文档关系
func (k *KnowledgeRepositoryImpl) SelectTopicDocumentRelationsByTopicCode(ctx context.Context, topicCode string) ([]*entity.KnowledgeTopicDocumentRelation, error) {
	var relations []*entity.KnowledgeTopicDocumentRelation
	query := k.dbWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{})
	if strutil.IsNotBlank(topicCode) {
		query = query.Where("topic_code = ?", topicCode)
	}
	if err := query.Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}

// UpsertTopicDocumentRelation 插入或更新主题-文档关系
func (k *KnowledgeRepositoryImpl) UpsertTopicDocumentRelation(ctx context.Context, relation *entity.KnowledgeTopicDocumentRelation) error {
	relModel := convert.ToKnowledgeTopicDocumentRelationModel(relation)
	if err := k.dbWithContext(ctx).Model(&model.KnowledgeTopicDocumentRelation{}).
		Where("topic_code = ? AND document_id = ?", relation.TopicCode, relation.DocumentId).
		First(relation).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return k.dbWithContext(ctx).Create(relModel).Error
	}
	relModel.ID = relation.ID
	return k.dbWithContext(ctx).Updates(relModel).Error
}

// DeleteTopicDocumentRelation 删除主题-文档关系
func (k *KnowledgeRepositoryImpl) DeleteTopicDocumentRelation(ctx context.Context, topicCode string, documentId int64) error {
	return k.dbWithContext(ctx).Where("topic_code = ? AND document_id = ?", topicCode, documentId).
		Delete(&model.KnowledgeTopicDocumentRelation{}).Error
}

// ============ 路由跟踪 ============

// InsertKnowledgeRouteTrace 插入路由跟踪
func (k *KnowledgeRepositoryImpl) InsertKnowledgeRouteTrace(ctx context.Context, trace *entity.KnowledgeRouteTrace) error {
	return k.dbWithContext(ctx).Model(&model.KnowledgeRouteTrace{}).Create(convert.ToKnowledgeRouteTraceModel(trace)).Error
}

// SelectKnowledgeRouteTracePage 分页查询路由跟踪
func (k *KnowledgeRepositoryImpl) SelectKnowledgeRouteTracePage(ctx context.Context, conversationId, mode string, routeStatus, pageNo, pageSize int) ([]*entity.KnowledgeRouteTrace, int64, error) {
	builder := k.dbWithContext(ctx).Model(&model.KnowledgeRouteTrace{})
	if strutil.IsNotBlank(conversationId) {
		builder = builder.Where("conversation_id = ?", conversationId)
	}
	if strutil.IsNotBlank(mode) {
		builder = builder.Where("mode = ?", mode)
	}
	if routeStatus > 0 {
		builder = builder.Where("route_status = ?", routeStatus)
	}

	var total int64
	if err := builder.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*entity.KnowledgeRouteTrace
	if err := builder.Scopes(utils.Paginate(pageNo, pageSize)).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
