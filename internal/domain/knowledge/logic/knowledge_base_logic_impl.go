package logic

import (
	"context"
	"errors"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

// KnowledgeConfigLogicImpl 知识库配置管理
type KnowledgeConfigLogicImpl struct {
	repo       adapter.KnowledgeRepository
	docGateway adapter.DocumentGateway
}

func NewKnowledgeConfigLogicImpl(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway) *KnowledgeConfigLogicImpl {
	return &KnowledgeConfigLogicImpl{
		repo:       repo,
		docGateway: docGateway,
	}
}

// SaveKnowledgeConfig 保存/更新知识库配置
func (k *KnowledgeConfigLogicImpl) SaveKnowledgeConfig(ctx context.Context, config *entity.KnowledgeConfig) (*entity.KnowledgeConfig, error) {
	if strutil.IsBlank(config.BaseName) {
		return nil, errors.New("baseName 不能为空")
	}

	// 名称唯一性校验
	existing, err := k.repo.SelectKnowledgeConfigByBaseName(ctx, strutil.Trim(config.BaseName))
	if err != nil && !errors.Is(err, errorx.ErrKnowledgeBaseNotFound) {
		return nil, err
	}
	if existing.ID != config.ID {
		return nil, errors.New("知识库名称已存在")
	}

	if config.ID > 0 {
		// 更新：查询现有记录确认存在
		_, err = k.repo.SelectKnowledgeConfigById(ctx, config.ID)
		if err != nil {
			return nil, err
		}
		if err = k.repo.UpdateKnowledgeConfigById(ctx, config); err != nil {
			return nil, err
		}
	} else {
		// 新增：分配雪花ID
		config.ID = utils.GetSnowflakeNextID()
		if err = k.repo.InsertKnowledgeConfig(ctx, config); err != nil {
			return nil, err
		}
	}

	// 如果标记为默认，清除其他默认标记
	if config.IsDefault == 1 {
		if err = k.repo.ClearOtherDefaults(ctx, config.ID); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// DeleteKnowledgeConfig 删除知识库配置
func (k *KnowledgeConfigLogicImpl) DeleteKnowledgeConfig(ctx context.Context, id int64) (bool, error) {
	if id <= 0 {
		return false, errors.New("id 不能为空")
	}
	if err := k.repo.DeleteKnowledgeConfigById(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

// ListKnowledgeConfigs 查询所有知识库配置列表
func (k *KnowledgeConfigLogicImpl) ListKnowledgeConfigs(ctx context.Context) ([]*entity.KnowledgeConfig, error) {
	return k.repo.SelectKnowledgeConfigs(ctx)
}

// GetKnowledgeConfig 根据ID查询知识库配置详情
func (k *KnowledgeConfigLogicImpl) GetKnowledgeConfig(ctx context.Context, id int64) (*entity.KnowledgeConfig, error) {
	if id <= 0 {
		return nil, errors.New("id 不能为空")
	}
	config, err := k.repo.SelectKnowledgeConfigById(ctx, id)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// UpdateKnowledgeConfigSetting 更新知识库配置
func (k *KnowledgeConfigLogicImpl) UpdateKnowledgeConfigSetting(ctx context.Context, config *entity.KnowledgeConfig) (*entity.KnowledgeConfig, error) {
	if config.ID <= 0 {
		return nil, errors.New("id 不能为空")
	}

	// 查询现有记录确认存在
	existing, err := k.repo.SelectKnowledgeConfigById(ctx, config.ID)
	if err != nil {
		return nil, err
	}

	// 仅更新配置JSON字段
	existing.RetrievalConfigJson = config.RetrievalConfigJson
	existing.GraphRagConfigJson = config.GraphRagConfigJson
	existing.RaptorConfigJson = config.RaptorConfigJson
	existing.MetadataFilterJson = config.MetadataFilterJson

	if err = k.repo.UpdateKnowledgeConfigById(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// ListEnabledKnowledgeConfigs 查询所有启用的知识库配置
func (k *KnowledgeConfigLogicImpl) ListEnabledKnowledgeConfigs(ctx context.Context) ([]*entity.KnowledgeConfig, error) {
	return k.repo.SelectKnowledgeConfigs(ctx)
}

// ListKnowledgeConfigsByIds 根据ID列表查询知识库配置
func (k *KnowledgeConfigLogicImpl) ListKnowledgeConfigsByIds(ctx context.Context, ids []int64) ([]*entity.KnowledgeConfig, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return k.repo.SelectKnowledgeConfigByIds(ctx, ids)
}

// GetEnabledKnowledgeConfig 根据ID获取启用的知识库配置
func (k *KnowledgeConfigLogicImpl) GetEnabledKnowledgeConfig(ctx context.Context, id int64) (*entity.KnowledgeConfig, error) {
	if id <= 0 {
		return nil, errors.New("id 不能为空")
	}
	config, err := k.repo.SelectKnowledgeConfigById(ctx, id)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// ListKnowledgeConfigOptions 查询知识库选项列表
func (k *KnowledgeConfigLogicImpl) ListKnowledgeConfigOptions(ctx context.Context) ([]*KnowledgeConfigOption, error) {
	configs, err := k.repo.SelectKnowledgeConfigs(ctx)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return []*KnowledgeConfigOption{}, nil
	}

	// 收集所有知识库ID
	kbIds := make([]int64, 0, len(configs))
	for _, c := range configs {
		kbIds = append(kbIds, c.ID)
	}

	// 批量统计可检索文档数量
	retrievableCounts, err := k.docGateway.CountRetrievableDocumentsByKnowledgeBaseIds(ctx, kbIds)
	if err != nil {
		return nil, err
	}

	// 构建选项列表
	options := make([]*KnowledgeConfigOption, 0, len(configs))
	for _, c := range configs {
		option := &KnowledgeConfigOption{
			ID:               c.ID,
			BaseName:         c.BaseName,
			Description:      c.Description,
			IsDefault:        c.IsDefault,
			RetrievableCount: retrievableCounts[c.ID],
		}
		options = append(options, option)
	}
	return options, nil
}
