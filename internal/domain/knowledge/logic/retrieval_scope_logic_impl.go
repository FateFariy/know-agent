package logic

import (
	"context"
	"strconv"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/config"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

// KnowledgeBaseRetrievalScopeLogicImpl 知识库检索范围
type KnowledgeBaseRetrievalScopeLogicImpl struct {
	repo       adapter.KnowledgeRepository
	docGateway adapter.DocumentGateway
	resolver   *config.Resolver
}

func NewKnowledgeBaseRetrievalScopeLogicImpl(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway, global config.GlobalConfigProvider) *KnowledgeBaseRetrievalScopeLogicImpl {
	return &KnowledgeBaseRetrievalScopeLogicImpl{
		repo:       repo,
		docGateway: docGateway,
		resolver:   config.NewResolver(global),
	}
}

// DetermineKnowledgeScope 根据聊天模式和知识库选择模式解析检索范围
func (s *KnowledgeBaseRetrievalScopeLogicImpl) DetermineKnowledgeScope(ctx context.Context, chatMode, selectMode string, kbIds []string) (*aggregate.KnowledgeBaseSelectionSnapshot, error) {
	// 解析选择模式
	resolvedMode := utils.BlankToDefault(selectMode, enum.KbSelectionModeNone)

	snapshot := &aggregate.KnowledgeBaseSelectionSnapshot{
		SelectionMode:     enum.KbSelectionModeNone,
		RagRuntimeOptions: s.resolver.ResolveRagRuntimeOptions(nil),
	}
	// 开放式聊天或无选择模式时返回空快照
	if chatMode == "open_chat" || resolvedMode == enum.KbSelectionModeNone {
		return snapshot, nil
	}

	// 根据模式选择知识库
	var selectedBases []*entity.KnowledgeBase
	var err error
	switch resolvedMode {
	case enum.KbSelectionModeAll:
		selectedBases, err = s.selectAllWithRetrievableDocuments(ctx)
	case enum.KbSelectionModeSelected:
		selectedBases, err = s.selectExplicit(ctx, kbIds)
	default:
		selectedBases = nil
	}
	if err != nil {
		return nil, err
	}

	if len(selectedBases) == 0 {
		return snapshot, nil
	}

	baseIds := utils.Map(selectedBases, func(base *entity.KnowledgeBase) int64 {
		return base.ID
	})

	// 查询允许的文档
	allowedDocuments, err := s.docGateway.FindRetrievableByKbIds(ctx, baseIds)
	if err != nil {
		return nil, err
	}

	// 提取有可检索文档的知识库
	mapBy := utils.MapBy(allowedDocuments, func(doc *vo.DocumentMetadata) (int64, struct{}) {
		return doc.KnowledgeBaseId, struct{}{}
	})

	// 收集选中的知识库
	selectedBaseIds := make([]int64, 0, len(selectedBases))
	selectedBaseNames := make([]string, 0, len(selectedBases))
	i := 0
	for _, base := range selectedBases {
		if _, exists := mapBy[base.ID]; exists {
			selectedBaseIds = append(selectedBaseIds, base.ID)
			selectedBaseNames = append(selectedBaseNames, base.BaseName)
			selectedBases[i] = base
			i++
		}
	}
	selectedBases = selectedBases[:i]
	if len(selectedBases) == 0 {
		return snapshot, nil
	}

	// 构建快照
	snapshot.SelectionMode = resolvedMode
	snapshot.SelectedKnowledgeBaseIds = selectedBaseIds
	snapshot.SelectedKnowledgeBaseNames = selectedBaseNames
	snapshot.SelectedKnowledgeBases = selectedBases
	snapshot.AllowedDocuments = allowedDocuments
	// todo 暂时注释掉，后续根据需要开启
	//snapshot.RagRuntimeOptions = s.resolver.ResolveRagRuntimeOptions(selectedBases)

	return snapshot, nil
}

// selectAllWithRetrievableDocuments 选择所有启用且有可检索文档的知识库
func (s *KnowledgeBaseRetrievalScopeLogicImpl) selectAllWithRetrievableDocuments(ctx context.Context) ([]*entity.KnowledgeBase, error) {
	enabledBases, err := s.repo.SelectKnowledgeBases(ctx)
	if err != nil {
		return nil, err
	}
	return enabledBases, nil
}

// selectExplicit 根据用户选择的ID列表选择知识库
func (s *KnowledgeBaseRetrievalScopeLogicImpl) selectExplicit(ctx context.Context, selectedKnowledgeBaseIds []string) ([]*entity.KnowledgeBase, error) {
	// 解析并去重知识库ID
	ids := utils.FilterMapUniqueLimit(selectedKnowledgeBaseIds, -1, func(rawId string) (int64, int64, bool) {
		id, _ := strconv.ParseInt(rawId, 10, 64)
		return id, id, id > 0
	})

	if len(ids) == 0 {
		return nil, errorx.ErrKnowledgeBaseMissing
	}

	// 查询知识库列表
	bases, err := s.repo.SelectKnowledgeBaseByIds(ctx, ids)
	if err != nil {
		return nil, err
	}

	// 构建ID索引
	byId := utils.MapBy(bases, func(base *entity.KnowledgeBase) (int64, *entity.KnowledgeBase) {
		return base.ID, base
	})

	// 检查所有请求的ID是否都存在
	for _, id := range ids {
		if _, exists := byId[id]; !exists {
			return nil, errorx.ErrKnowledgeBaseDisabled.Format(id)
		}
	}

	// 按原始ID顺序返回
	result := make([]*entity.KnowledgeBase, 0, len(ids))
	for _, id := range ids {
		result = append(result, byId[id])
	}

	return result, nil
}
