package config

import (
	"encoding/json"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// Resolver 实现配置解析与合并
type Resolver struct {
	provider GlobalConfigProvider
}

// NewResolver 创建 Resolver 实例
func NewResolver(provider GlobalConfigProvider) *Resolver {
	return &Resolver{provider: provider}
}

// ResolveRagRuntimeOptions 根据知识库列表解析出最终的 RagRuntimeOptions
func (r *Resolver) ResolveRagRuntimeOptions(knowledgeBases []*entity.KnowledgeBase) *vo.RagRuntimeOptions {
	options := r.provider.CurrentOptions()
	if len(knowledgeBases) == 0 {
		return options
	}

	// 解析所有知识库配置
	configs := make([]*RuntimeConfig, 0, len(knowledgeBases))
	for _, kb := range knowledgeBases {
		if kb == nil {
			continue
		}
		cfg := r.parseConfig(kb)
		configs = append(configs, cfg)
	}

	if len(configs) == 0 {
		return options
	}

	if len(configs) == 1 {
		r.applySingle(options, configs[0])
		return options
	}

	r.applyMerged(options, configs)
	return options
}

// ResolveIndexingOptions 根据知识库列表解析出索引配置 todo 暂时返回全局默认
func (r *Resolver) ResolveIndexingOptions(knowledgeBases []*entity.KnowledgeBase) *vo.IndexingOptions {
	options := r.provider.CurrentIndexingOptions()
	if len(knowledgeBases) == 0 {
		return options
	}
	return options
}

// parseConfig 解析单个知识库的多个 JSON 片段并合并
func (r *Resolver) parseConfig(kb *entity.KnowledgeBase) *RuntimeConfig {
	merged := &RuntimeConfig{
		VectorTopK:                new(int),
		KeywordTopK:               new(int),
		GraphRagTopK:              new(int),
		GraphRagMaxHops:           new(int),
		RaptorTopK:                new(int),
		RaptorSourceChunkTopK:     new(int),
		CandidateTopK:             new(int),
		RerankCandidateTopK:       new(int),
		FinalTopK:                 new(int),
		RerankEnabled:             new(bool),
		ChannelTimeout:            new(time.Duration),
		SubQuestionTimeout:        new(time.Duration),
		MinVectorSimilarity:       new(float64),
		KeywordRelativeScoreFloor: new(float64),
		KeywordChannelEnabled:     new(bool),
		TableChannelEnabled:       new(bool),
		GraphRagChannelEnabled:    new(bool),
		RaptorChannelEnabled:      new(bool),
		Hybrid: &HybridConfig{
			VectorWeight:        new(float64),
			KeywordWeight:       new(float64),
			TableWeight:         new(float64),
			GraphRagWeight:      new(float64),
			RaptorWeight:        new(float64),
			RankWeight:          new(float64),
			OriginalScoreWeight: new(float64),
			MetadataBoostWeight: new(float64),
			MaxMetadataBoost:    new(float64),
		},
	}
	// 顺序合并，后解析的覆盖先解析的（如果有同名字段）
	r.mergeInto(merged, r.parseJSON(kb.RetrievalConfigJson, kb))
	r.mergeInto(merged, r.parseJSON(kb.GraphRagConfigJson, kb))
	r.mergeInto(merged, r.parseJSON(kb.RaptorConfigJson, kb))
	return merged
}

// parseJSON 将 JSON 字符串解析为 RuntimeConfig，失败返回空配置
func (r *Resolver) parseJSON(raw json.RawMessage, kb *entity.KnowledgeBase) *RuntimeConfig {
	if len(raw) == 0 {
		return &RuntimeConfig{}
	}
	var cfg RuntimeConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		id := int64(0)
		name := ""
		if kb != nil {
			id = kb.ID
			name = kb.BaseName
		}
		logx.Warnf("知识库 RAG 配置 JSON 解析失败，将忽略该段配置: knowledgeBaseId=%d, knowledgeBaseName=%s, err=%v", id, name, err)
		return &RuntimeConfig{}
	}
	return &cfg
}

// mergeInto 将 source 中非 nil 字段复制到 target（target 原有值保留，除非 source 有值）
func (r *Resolver) mergeInto(target, source *RuntimeConfig) {
	if source == nil {
		return
	}
	setIfNotNil(source.VectorTopK, target.VectorTopK)
	setIfNotNil(source.KeywordTopK, target.KeywordTopK)
	setIfNotNil(source.GraphRagTopK, target.GraphRagTopK)
	setIfNotNil(source.GraphRagMaxHops, target.GraphRagMaxHops)
	setIfNotNil(source.RaptorTopK, target.RaptorTopK)
	setIfNotNil(source.RaptorSourceChunkTopK, target.RaptorSourceChunkTopK)
	setIfNotNil(source.CandidateTopK, target.CandidateTopK)
	setIfNotNil(source.RerankCandidateTopK, target.RerankCandidateTopK)
	setIfNotNil(source.FinalTopK, target.FinalTopK)
	setIfNotNil(source.RerankEnabled, target.RerankEnabled)
	setIfNotNil(source.ChannelTimeout, target.ChannelTimeout)
	setIfNotNil(source.SubQuestionTimeout, target.SubQuestionTimeout)
	setIfNotNil(source.MinVectorSimilarity, target.MinVectorSimilarity)
	setIfNotNil(source.KeywordRelativeScoreFloor, target.KeywordRelativeScoreFloor)
	setIfNotNil(source.KeywordChannelEnabled, target.KeywordChannelEnabled)
	setIfNotNil(source.TableChannelEnabled, target.TableChannelEnabled)
	setIfNotNil(source.GraphRagChannelEnabled, target.GraphRagChannelEnabled)
	setIfNotNil(source.RaptorChannelEnabled, target.RaptorChannelEnabled)

	if source.Hybrid != nil {
		if target.Hybrid == nil {
			target.Hybrid = &HybridConfig{}
		}
		r.mergeHybridInto(target.Hybrid, source.Hybrid)
	}
}

// mergeHybridInto 同 mergeInto 但处理 HybridConfig
func (r *Resolver) mergeHybridInto(target, source *HybridConfig) {
	setIfNotNil(source.VectorWeight, target.VectorWeight)
	setIfNotNil(source.KeywordWeight, target.KeywordWeight)
	setIfNotNil(source.TableWeight, target.TableWeight)
	setIfNotNil(source.GraphRagWeight, target.GraphRagWeight)
	setIfNotNil(source.RaptorWeight, target.RaptorWeight)
	setIfNotNil(source.RankWeight, target.RankWeight)
	setIfNotNil(source.OriginalScoreWeight, target.OriginalScoreWeight)
	setIfNotNil(source.MetadataBoostWeight, target.MetadataBoostWeight)
	setIfNotNil(source.MaxMetadataBoost, target.MaxMetadataBoost)
}

// applySingle 单知识库配置覆盖全局选项
func (r *Resolver) applySingle(options *vo.RagRuntimeOptions, cfg *RuntimeConfig) {
	setIfNotNil(cfg.VectorTopK, &options.VectorTopK)
	setIfNotNil(cfg.KeywordTopK, &options.KeywordTopK)
	setIfNotNil(cfg.GraphRagTopK, &options.GraphRagTopK)
	setIfNotNil(cfg.GraphRagMaxHops, &options.GraphRagMaxHops)
	setIfNotNil(cfg.RaptorTopK, &options.RaptorTopK)
	setIfNotNil(cfg.RaptorSourceChunkTopK, &options.RaptorSourceChunkTopK)
	setIfNotNil(cfg.CandidateTopK, &options.CandidateTopK)
	setIfNotNil(cfg.RerankCandidateTopK, &options.RerankCandidateTopK)
	setIfNotNil(cfg.FinalTopK, &options.FinalTopK)
	setIfNotNil(cfg.RerankEnabled, &options.RerankEnabled)
	setIfNotNil(cfg.ChannelTimeout, &options.ChannelTimeout)
	setIfNotNil(cfg.SubQuestionTimeout, &options.SubQuestionTimeout)
	setIfNotNil(cfg.MinVectorSimilarity, &options.MinVectorSimilarity)
	setIfNotNil(cfg.KeywordRelativeScoreFloor, &options.KeywordRelativeScoreFloor)
	setIfNotNil(cfg.KeywordChannelEnabled, &options.KeywordChannelEnabled)
	setIfNotNil(cfg.TableChannelEnabled, &options.TableChannelEnabled)
	setIfNotNil(cfg.GraphRagChannelEnabled, &options.GraphRagChannelEnabled)
	setIfNotNil(cfg.RaptorChannelEnabled, &options.RaptorChannelEnabled)

	if cfg.Hybrid != nil {
		if options.Hybrid == nil {
			options.Hybrid = vo.NewDefaultHybridOptions()
		}
		r.applySingleHybrid(options.Hybrid, cfg.Hybrid)
	}
}

// applySingleHybrid 单配置覆盖 HybridOptions
func (r *Resolver) applySingleHybrid(target *vo.HybridOptions, source *HybridConfig) {
	setIfNotNil(source.VectorWeight, &target.VectorWeight)
	setIfNotNil(source.KeywordWeight, &target.KeywordWeight)
	setIfNotNil(source.TableWeight, &target.TableWeight)
	setIfNotNil(source.GraphRagWeight, &target.GraphRagWeight)
	setIfNotNil(source.RaptorWeight, &target.RaptorWeight)
	setIfNotNil(source.RankWeight, &target.RankWeight)
	setIfNotNil(source.OriginalScoreWeight, &target.OriginalScoreWeight)
	setIfNotNil(source.MetadataBoostWeight, &target.MetadataBoostWeight)
	setIfNotNil(source.MaxMetadataBoost, &target.MaxMetadataBoost)
}

// applyMerged 多个配置合并，检测冲突
func (r *Resolver) applyMerged(options *vo.RagRuntimeOptions, configs []*RuntimeConfig) {
	conflicts := make([]string, 0)

	// 普通字段冲突检测
	mergeField(configs, func(c *RuntimeConfig) *int { return c.VectorTopK },
		func(v int) { options.VectorTopK = v }, "vectorTopK", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *int { return c.KeywordTopK }, func(v int) { options.KeywordTopK = v }, "keywordTopK", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *int { return c.GraphRagTopK }, func(v int) { options.GraphRagTopK = v }, "graphRagTopK", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *int { return c.GraphRagMaxHops }, func(v int) { options.GraphRagMaxHops = v }, "graphRagMaxHops", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *int { return c.RaptorTopK }, func(v int) { options.RaptorTopK = v }, "raptorTopK", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *int { return c.RaptorSourceChunkTopK }, func(v int) { options.RaptorSourceChunkTopK = v }, "raptorSourceChunkTopK", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *int { return c.CandidateTopK }, func(v int) { options.CandidateTopK = v }, "candidateTopK", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *int { return c.RerankCandidateTopK }, func(v int) { options.RerankCandidateTopK = v }, "rerankCandidateTopK", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *int { return c.FinalTopK }, func(v int) { options.FinalTopK = v }, "finalTopK", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *bool { return c.RerankEnabled }, func(v bool) { options.RerankEnabled = v }, "rerankEnabled", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *time.Duration { return c.ChannelTimeout }, func(v time.Duration) { options.ChannelTimeout = v }, "channelTimeoutMs", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *time.Duration { return c.SubQuestionTimeout }, func(v time.Duration) { options.SubQuestionTimeout = v }, "subQuestionTimeoutMs", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *float64 { return c.MinVectorSimilarity }, func(v float64) { options.MinVectorSimilarity = v }, "minVectorSimilarity", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *float64 { return c.KeywordRelativeScoreFloor }, func(v float64) { options.KeywordRelativeScoreFloor = v }, "keywordRelativeScoreFloor", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *bool { return c.KeywordChannelEnabled }, func(v bool) { options.KeywordChannelEnabled = v }, "keywordChannelEnabled", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *bool { return c.TableChannelEnabled }, func(v bool) { options.TableChannelEnabled = v }, "tableChannelEnabled", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *bool { return c.GraphRagChannelEnabled }, func(v bool) { options.GraphRagChannelEnabled = v }, "graphRagChannelEnabled", &conflicts)
	mergeField(configs, func(c *RuntimeConfig) *bool { return c.RaptorChannelEnabled }, func(v bool) { options.RaptorChannelEnabled = v }, "raptorChannelEnabled", &conflicts)

	// Hybrid 字段冲突检测
	if options.Hybrid == nil {
		options.Hybrid = vo.NewDefaultHybridOptions()
	}
	mergeHybridField(configs, func(h *HybridConfig) *float64 { return h.VectorWeight },
		func(v float64) { options.Hybrid.VectorWeight = v }, "hybrid.vectorWeight", &conflicts)
	mergeHybridField(configs, func(h *HybridConfig) *float64 { return h.KeywordWeight },
		func(v float64) { options.Hybrid.KeywordWeight = v }, "hybrid.keywordWeight", &conflicts)
	mergeHybridField(configs, func(h *HybridConfig) *float64 { return h.TableWeight },
		func(v float64) { options.Hybrid.TableWeight = v }, "hybrid.tableWeight", &conflicts)
	mergeHybridField(configs, func(h *HybridConfig) *float64 { return h.GraphRagWeight },
		func(v float64) { options.Hybrid.GraphRagWeight = v }, "hybrid.graphRagWeight", &conflicts)
	mergeHybridField(configs, func(h *HybridConfig) *float64 { return h.RaptorWeight },
		func(v float64) { options.Hybrid.RaptorWeight = v }, "hybrid.raptorWeight", &conflicts)
	mergeHybridField(configs, func(h *HybridConfig) *float64 { return h.RankWeight },
		func(v float64) { options.Hybrid.RankWeight = v }, "hybrid.rankWeight", &conflicts)
	mergeHybridField(configs, func(h *HybridConfig) *float64 { return h.OriginalScoreWeight },
		func(v float64) { options.Hybrid.OriginalScoreWeight = v }, "hybrid.originalScoreWeight", &conflicts)
	mergeHybridField(configs, func(h *HybridConfig) *float64 { return h.MetadataBoostWeight },
		func(v float64) { options.Hybrid.MetadataBoostWeight = v }, "hybrid.metadataBoostWeight", &conflicts)
	mergeHybridField(configs, func(h *HybridConfig) *float64 { return h.MaxMetadataBoost },
		func(v float64) { options.Hybrid.MaxMetadataBoost = v }, "hybrid.maxMetadataBoost", &conflicts)

	// 去重冲突字段
	uniq := map[string]struct{}{}
	for _, f := range conflicts {
		uniq[f] = struct{}{}
	}
	if len(uniq) == 0 {
		return
	}
	options.KbConfigConflictFields = make([]string, 0, len(uniq))
	for f := range uniq {
		options.KbConfigConflictFields = append(options.KbConfigConflictFields, f)
	}
}

// mergeField 泛型辅助函数，检测多个配置中某字段是否一致，若一致则设置，否则记录冲突
func mergeField[T comparable](configs []*RuntimeConfig,
	getter func(*RuntimeConfig) *T,
	setter func(T),
	fieldName string,
	conflicts *[]string) {

	var values []*T
	for _, cfg := range configs {
		values = append(values, getter(cfg))
	}
	// 检查是否全部为 nil
	allNil := true
	for _, v := range values {
		if v != nil {
			allNil = false
			break
		}
	}
	if allNil {
		return
	}
	// 检查是否部分 nil
	hasNil := false
	for _, v := range values {
		if v == nil {
			hasNil = true
			break
		}
	}
	if hasNil {
		*conflicts = append(*conflicts, fieldName)
		return
	}
	// 全部非 nil，检查是否相等
	first := *values[0]
	allSame := true
	for _, v := range values {
		if *v != first {
			allSame = false
			break
		}
	}
	if allSame {
		setter(first)
	} else {
		*conflicts = append(*conflicts, fieldName)
	}
}

func mergeHybridField[T comparable](configs []*RuntimeConfig,
	getter func(*HybridConfig) *T,
	setter func(T),
	fieldName string,
	conflicts *[]string) {

	values := make([]*T, 0, len(configs))
	for _, cfg := range configs {
		if cfg.Hybrid != nil {
			values = append(values, getter(cfg.Hybrid))
		} else {
			values = append(values, nil)
		}
	}
	// 与 mergeField 相同逻辑
	allNil := true
	for _, v := range values {
		if v != nil {
			allNil = false
			break
		}
	}
	if allNil {
		return
	}
	hasNil := false
	for _, v := range values {
		if v == nil {
			hasNil = true
			break
		}
	}
	if hasNil {
		*conflicts = append(*conflicts, fieldName)
		return
	}
	first := *values[0]
	allSame := true
	for _, v := range values {
		if *v != first {
			allSame = false
			break
		}
	}
	if allSame {
		setter(first)
	} else {
		*conflicts = append(*conflicts, fieldName)
	}
}

// 辅助设置函数：如果 src 非 nil，将值赋给 dst
func setIfNotNil[T any](src *T, dst *T) {
	if src != nil {
		*dst = *src
	}
}

type HybridConfig struct {
	VectorWeight        *float64 `json:"vectorWeight,omitempty"`
	KeywordWeight       *float64 `json:"keywordWeight,omitempty"`
	TableWeight         *float64 `json:"tableWeight,omitempty"`
	GraphRagWeight      *float64 `json:"graphRagWeight,omitempty"`
	RaptorWeight        *float64 `json:"raptorWeight,omitempty"`
	RankWeight          *float64 `json:"rankWeight,omitempty"`
	OriginalScoreWeight *float64 `json:"originalScoreWeight,omitempty"`
	MetadataBoostWeight *float64 `json:"metadataBoostWeight,omitempty"`
	MaxMetadataBoost    *float64 `json:"maxMetadataBoost,omitempty"`
}

type RuntimeConfig struct {
	VectorTopK                *int           `json:"vectorTopK,omitempty"`
	KeywordTopK               *int           `json:"keywordTopK,omitempty"`
	GraphRagTopK              *int           `json:"graphRagTopK,omitempty"`
	GraphRagMaxHops           *int           `json:"graphRagMaxHops,omitempty"`
	RaptorTopK                *int           `json:"raptorTopK,omitempty"`
	RaptorSourceChunkTopK     *int           `json:"raptorSourceChunkTopK,omitempty"`
	CandidateTopK             *int           `json:"candidateTopK,omitempty"`
	RerankCandidateTopK       *int           `json:"rerankCandidateTopK,omitempty"`
	FinalTopK                 *int           `json:"finalTopK,omitempty"`
	RerankEnabled             *bool          `json:"rerankEnabled,omitempty"`
	ChannelTimeout            *time.Duration `json:"channelTimeout,omitempty"`
	SubQuestionTimeout        *time.Duration `json:"subQuestionTimeout,omitempty"`
	MinVectorSimilarity       *float64       `json:"minVectorSimilarity,omitempty"`
	KeywordRelativeScoreFloor *float64       `json:"keywordRelativeScoreFloor,omitempty"`
	KeywordChannelEnabled     *bool          `json:"keywordChannelEnabled,omitempty"`
	TableChannelEnabled       *bool          `json:"tableChannelEnabled,omitempty"`
	GraphRagChannelEnabled    *bool          `json:"graphRagChannelEnabled,omitempty"`
	RaptorChannelEnabled      *bool          `json:"raptorChannelEnabled,omitempty"`
	Hybrid                    *HybridConfig  `json:"hybrid,omitempty"`
}
