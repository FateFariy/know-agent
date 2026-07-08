package logic

import (
	"context"
	"errors"
	"sort"
	"strconv"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	documentlogic "github.com/swiftbit/know-agent/internal/domain/document/logic"
	documentvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// TopicDocumentRelationVo 主题-文档关联的视图对象（含文档侧元数据）
type TopicDocumentRelationVo struct {
	TopicCode          string
	DocumentId         int64
	DocumentName       string
	KnowledgeScopeCode string
	KnowledgeScopeName string
	BusinessCategory   string
	DocumentTags       string
	RelationScore      float64
	RelationSource     string
	Reason             string
}

// KnowledgeLogicImpl 知识管理领域实现
type KnowledgeLogicImpl struct {
	repo          adapter.KnowledgeRepository
	documentLogic documentlogic.LifecycleLogic
}

// NewKnowledgeLogic 构造函数
func NewKnowledgeLogic(repo adapter.KnowledgeRepository, documentLogic documentlogic.LifecycleLogic) *KnowledgeLogicImpl {
	return &KnowledgeLogicImpl{repo: repo, documentLogic: documentLogic}
}

// ============ Scope ============

func (k *KnowledgeLogicImpl) SaveScope(ctx context.Context, scopeNode *entity.KnowledgeScopeNode) (*entity.KnowledgeScopeNode, error) {
	if err := k.repo.UpsertKnowledgeScopeNode(ctx, scopeNode); err != nil {
		return nil, err
	}
	return scopeNode, nil
}

func (k *KnowledgeLogicImpl) DeleteScope(ctx context.Context, scopeCode string) (bool, error) {
	if err := k.repo.DeleteKnowledgeScopeNode(ctx, strutil.Trim(scopeCode)); err != nil {
		return false, err
	}
	return true, nil
}

func (k *KnowledgeLogicImpl) ListScopes(ctx context.Context) ([]*entity.KnowledgeScopeNode, error) {
	return k.repo.SelectKnowledgeScopeNodes(ctx)
}

// ============ Topic ============

func (k *KnowledgeLogicImpl) SaveTopic(ctx context.Context, topicNode *entity.KnowledgeTopicNode) (*entity.KnowledgeTopicNode, error) {
	if err := k.repo.UpsertKnowledgeTopicNode(ctx, topicNode); err != nil {
		return nil, err
	}
	return topicNode, nil
}

func (k *KnowledgeLogicImpl) DeleteTopic(ctx context.Context, topicCode string) (bool, error) {
	if strutil.IsBlank(topicCode) {
		return false, nil
	}
	if err := k.repo.DeleteKnowledgeTopicNode(ctx, strutil.Trim(topicCode)); err != nil {
		return false, err
	}
	return true, nil
}

func (k *KnowledgeLogicImpl) ListTopics(ctx context.Context, scopeCode string) ([]*entity.KnowledgeTopicNode, error) {
	var (
		nodes []*entity.KnowledgeTopicNode
		err   error
	)
	if strutil.IsBlank(scopeCode) {
		nodes, err = k.repo.SelectKnowledgeTopicNodes(ctx)
	} else {
		nodes, err = k.repo.SelectKnowledgeTopicNodesByScopeCode(ctx, strutil.Trim(scopeCode))
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].SortOrder < nodes[j].SortOrder })
	return nodes, nil
}

// ============ Document Profile ============

func (k *KnowledgeLogicImpl) GetDocumentProfile(ctx context.Context, documentId int64) (*entity.KnowledgeDocumentProfile, error) {
	if documentId <= 0 {
		return nil, errors.New("documentId 非法")
	}
	return k.repo.SelectDocumentProfileByDocumentId(ctx, documentId)
}

func (k *KnowledgeLogicImpl) RegenerateDocumentProfile(ctx context.Context, documentId int64) (*entity.KnowledgeDocumentProfile, error) {
	if documentId <= 0 {
		return nil, errors.New("documentId 非法")
	}
	// 此处仅执行画像的写入/更新操作；真正的“重新生成”由上游的异步 pipeline 完成
	profile := &entity.KnowledgeDocumentProfile{
		DocumentId:    documentId,
		ProfileStatus: 1, // 标记为生成中
	}
	if err := k.repo.UpsertDocumentProfile(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (k *KnowledgeLogicImpl) BatchRegenerateDocumentProfiles(ctx context.Context, documentIds []int64) ([]*entity.KnowledgeDocumentProfile, error) {
	if len(documentIds) == 0 {
		return nil, errors.New("documentIds 不能为空")
	}
	profiles := make([]*entity.KnowledgeDocumentProfile, 0, len(documentIds))
	for _, id := range documentIds {
		if id <= 0 {
			continue
		}
		profiles = append(profiles, &entity.KnowledgeDocumentProfile{DocumentId: id, ProfileStatus: 1})
	}
	if err := k.repo.BatchUpsertDocumentProfiles(ctx, profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

// ============ Topic-Document Relation ============

func (k *KnowledgeLogicImpl) ListTopicDocumentRelations(ctx context.Context, topicCode string) ([]TopicDocumentRelationVo, error) {
	if strutil.IsBlank(topicCode) {
		return nil, nil
	}
	relations, err := k.repo.SelectTopicDocumentRelationsByTopicCode(ctx, strutil.Trim(topicCode))
	if err != nil {
		return nil, err
	}
	if len(relations) == 0 {
		return nil, nil
	}
	documents, err := k.documentLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		return nil, err
	}
	docMap := make(map[int64]*documentvo.KnowledgeDocument, len(documents))
	for _, d := range documents {
		docMap[d.DocumentId] = d
	}
	result := make([]TopicDocumentRelationVo, 0, len(relations))
	for _, rel := range relations {
		vo := TopicDocumentRelationVo{
			TopicCode:      rel.TopicCode,
			DocumentId:     rel.DocumentId,
			RelationScore:  rel.RelationScore,
			RelationSource: rel.RelationSource,
			Reason:         rel.Reason,
		}
		if doc := docMap[rel.DocumentId]; doc != nil {
			vo.DocumentName = doc.DocumentName
			vo.KnowledgeScopeCode = doc.KnowledgeScopeCode
			vo.KnowledgeScopeName = doc.KnowledgeScopeName
			vo.BusinessCategory = doc.BusinessCategory
			vo.DocumentTags = doc.DocumentTags
		}
		result = append(result, vo)
	}
	return result, nil
}

func (k *KnowledgeLogicImpl) SaveTopicDocumentRelation(ctx context.Context, topicCode string, documentId int64, relationScore float64, relationSource, reason string) (*entity.KnowledgeTopicDocumentRelation, error) {
	if strutil.IsBlank(topicCode) || documentId <= 0 {
		return nil, errors.New("topicCode 与 documentId 不能为空")
	}
	rel := &entity.KnowledgeTopicDocumentRelation{
		TopicCode:      strutil.Trim(topicCode),
		DocumentId:     documentId,
		RelationScore:  relationScore,
		RelationSource: strutil.Trim(relationSource),
		Reason:         strutil.Trim(reason),
	}
	if err := k.repo.UpsertTopicDocumentRelation(ctx, rel); err != nil {
		return nil, err
	}
	return rel, nil
}

func (k *KnowledgeLogicImpl) RemoveTopicDocumentRelation(ctx context.Context, topicCode string, documentId int64) (bool, error) {
	if strutil.IsBlank(topicCode) || documentId <= 0 {
		return false, nil
	}
	if err := k.repo.RemoveTopicDocumentRelation(ctx, strutil.Trim(topicCode), documentId); err != nil {
		return false, err
	}
	return true, nil
}

// ============ Route Trace ============

func (k *KnowledgeLogicImpl) QueryRouteTracePage(ctx context.Context, conversationId, mode, routeStatus string, pageNo, pageSize int32) ([]*entity.KnowledgeRouteTrace, int64, error) {
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	return k.repo.SelectKnowledgeRouteTracePage(ctx, strutil.Trim(conversationId), strutil.Trim(mode), strutil.Trim(routeStatus), pageNo, pageSize)
}

// ============ helpers for converting route status code <-> string ============

// RouteStatusName 按整数返回对应的字符串描述（用于对外展示）
func RouteStatusName(status int) string {
	switch vo.RouteStatus(status) {
	case vo.RouteStatusSuccess:
		return "SUCCESS"
	case vo.RouteStatusLowConfidence:
		return "LOW_CONFIDENCE"
	case vo.RouteStatusFailed:
		return "FAILED"
	}
	return strconv.Itoa(status)
}

// RouteStatusHitFlag 将布尔/int 值转为字符串展示字段
func RouteStatusHitFlag(hit int) string {
	if hit > 0 {
		return "Y"
	}
	return "N"
}

// ensure utils.LimitSlice 在本包可引用
var _ = utils.LimitSlice[int]
