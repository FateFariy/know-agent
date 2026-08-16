package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// mockProvider 实现 GlobalRagRuntimeConfigProvider 接口，用于测试
type mockProvider struct {
	defaultOptions *vo.RagRuntimeOptions
}

func (m *mockProvider) CurrentOptions() *vo.RagRuntimeOptions {
	return m.defaultOptions
}

// newTestOptions 返回一个带有默认值的 RagRuntimeOptions，方便测试对比
func newTestOptions() *vo.RagRuntimeOptions {
	return &vo.RagRuntimeOptions{
		VectorTopK:                10,
		KeywordTopK:               5,
		GraphRagTopK:              3,
		GraphRagMaxHops:           2,
		RaptorTopK:                4,
		RaptorSourceChunkTopK:     6,
		CandidateTopK:             20,
		RerankCandidateTopK:       15,
		FinalTopK:                 5,
		RerankEnabled:             false,
		ChannelTimeout:            5 * time.Second,
		SubQuestionTimeout:        30 * time.Second,
		MinVectorSimilarity:       0.7,
		KeywordRelativeScoreFloor: 0.5,
		KeywordChannelEnabled:     true,
		TableChannelEnabled:       false,
		GraphRagChannelEnabled:    true,
		RaptorChannelEnabled:      false,
		Hybrid: &vo.HybridOptions{
			VectorWeight:        0.6,
			KeywordWeight:       0.2,
			TableWeight:         0.1,
			GraphRagWeight:      0.05,
			RaptorWeight:        0.05,
			RankWeight:          1.0,
			OriginalScoreWeight: 0.8,
			MetadataBoostWeight: 0.3,
			MaxMetadataBoost:    2.0,
		},
		KbConfigConflictFields: nil,
	}
}

// 辅助函数：生成知识库配置的 JSON 片段
func makeJSON(t *testing.T, v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	assert.NoError(t, err)
	return data
}

// TestResolver_Resolve_EmptyKnowledgeBases 测试空知识库列表
func TestResolver_Resolve_EmptyKnowledgeBases(t *testing.T) {
	defaultOpts := newTestOptions()
	provider := &mockProvider{defaultOptions: defaultOpts}
	resolver := NewResolver(provider)

	// 传入空切片或 nil
	result := resolver.Resolve([]*entity.KnowledgeBase{})
	assert.Equal(t, defaultOpts, result, "空列表应返回原 options")

	result = resolver.Resolve(nil)
	assert.Equal(t, defaultOpts, result, "nil 应返回原 options")
}

// TestResolver_Resolve_SingleKB_FullOverride 测试单个知识库完全覆盖
func TestResolver_Resolve_SingleKB_FullOverride(t *testing.T) {
	defaultOpts := newTestOptions()
	provider := &mockProvider{defaultOptions: defaultOpts}
	resolver := NewResolver(provider)

	// 构造一个覆盖所有字段的配置
	cfg := RuntimeConfig{
		VectorTopK:                intPtr(100),
		KeywordTopK:               intPtr(50),
		GraphRagTopK:              intPtr(30),
		GraphRagMaxHops:           intPtr(5),
		RaptorTopK:                intPtr(40),
		RaptorSourceChunkTopK:     intPtr(60),
		CandidateTopK:             intPtr(200),
		RerankCandidateTopK:       intPtr(150),
		FinalTopK:                 intPtr(50),
		RerankEnabled:             boolPtr(true),
		ChannelTimeout:            durationPtr(10 * time.Second),
		SubQuestionTimeout:        durationPtr(60 * time.Second),
		MinVectorSimilarity:       float64Ptr(0.9),
		KeywordRelativeScoreFloor: float64Ptr(0.8),
		KeywordChannelEnabled:     boolPtr(false),
		TableChannelEnabled:       boolPtr(true),
		GraphRagChannelEnabled:    boolPtr(false),
		RaptorChannelEnabled:      boolPtr(true),
		Hybrid: &HybridConfig{
			VectorWeight:        float64Ptr(0.5),
			KeywordWeight:       float64Ptr(0.3),
			TableWeight:         float64Ptr(0.1),
			GraphRagWeight:      float64Ptr(0.05),
			RaptorWeight:        float64Ptr(0.05),
			RankWeight:          float64Ptr(0.9),
			OriginalScoreWeight: float64Ptr(0.7),
			MetadataBoostWeight: float64Ptr(0.2),
			MaxMetadataBoost:    float64Ptr(1.5),
		},
	}
	jsonData := makeJSON(t, cfg)
	kb := &entity.KnowledgeBase{
		ID:                  1,
		BaseName:            "test-kb",
		RetrievalConfigJson: jsonData, // 所有字段放在这一个片段中，其他片段为空
		GraphRagConfigJson:  nil,
		RaptorConfigJson:    nil,
	}

	result := resolver.Resolve([]*entity.KnowledgeBase{kb})

	// 验证所有字段都被覆盖
	expected := *defaultOpts
	expected.VectorTopK = *cfg.VectorTopK
	expected.KeywordTopK = *cfg.KeywordTopK
	expected.GraphRagTopK = *cfg.GraphRagTopK
	expected.GraphRagMaxHops = *cfg.GraphRagMaxHops
	expected.RaptorTopK = *cfg.RaptorTopK
	expected.RaptorSourceChunkTopK = *cfg.RaptorSourceChunkTopK
	expected.CandidateTopK = *cfg.CandidateTopK
	expected.RerankCandidateTopK = *cfg.RerankCandidateTopK
	expected.FinalTopK = *cfg.FinalTopK
	expected.RerankEnabled = *cfg.RerankEnabled
	expected.ChannelTimeout = *cfg.ChannelTimeout
	expected.SubQuestionTimeout = *cfg.SubQuestionTimeout
	expected.MinVectorSimilarity = *cfg.MinVectorSimilarity
	expected.KeywordRelativeScoreFloor = *cfg.KeywordRelativeScoreFloor
	expected.KeywordChannelEnabled = *cfg.KeywordChannelEnabled
	expected.TableChannelEnabled = *cfg.TableChannelEnabled
	expected.GraphRagChannelEnabled = *cfg.GraphRagChannelEnabled
	expected.RaptorChannelEnabled = *cfg.RaptorChannelEnabled
	expected.Hybrid.VectorWeight = *cfg.Hybrid.VectorWeight
	expected.Hybrid.KeywordWeight = *cfg.Hybrid.KeywordWeight
	expected.Hybrid.TableWeight = *cfg.Hybrid.TableWeight
	expected.Hybrid.GraphRagWeight = *cfg.Hybrid.GraphRagWeight
	expected.Hybrid.RaptorWeight = *cfg.Hybrid.RaptorWeight
	expected.Hybrid.RankWeight = *cfg.Hybrid.RankWeight
	expected.Hybrid.OriginalScoreWeight = *cfg.Hybrid.OriginalScoreWeight
	expected.Hybrid.MetadataBoostWeight = *cfg.Hybrid.MetadataBoostWeight
	expected.Hybrid.MaxMetadataBoost = *cfg.Hybrid.MaxMetadataBoost
	// 冲突字段应为空
	expected.KbConfigConflictFields = nil

	assert.Equal(t, &expected, result)
}

// TestResolver_Resolve_SingleKB_PartialOverride 测试单个知识库部分覆盖
func TestResolver_Resolve_SingleKB_PartialOverride(t *testing.T) {
	defaultOpts := newTestOptions()
	provider := &mockProvider{defaultOptions: defaultOpts}
	resolver := NewResolver(provider)

	// 只设置部分字段
	cfg := RuntimeConfig{
		VectorTopK:    intPtr(999),
		RerankEnabled: boolPtr(true),
		Hybrid: &HybridConfig{
			VectorWeight: float64Ptr(0.88),
		},
	}
	jsonData := makeJSON(t, cfg)
	kb := &entity.KnowledgeBase{
		ID:                  2,
		BaseName:            "partial",
		RetrievalConfigJson: jsonData,
		GraphRagConfigJson:  nil,
		RaptorConfigJson:    nil,
	}

	result := resolver.Resolve([]*entity.KnowledgeBase{kb})

	// 只有设置了的字段改变，其他保留默认
	expected := *defaultOpts
	expected.VectorTopK = *cfg.VectorTopK
	expected.RerankEnabled = *cfg.RerankEnabled
	expected.Hybrid.VectorWeight = *cfg.Hybrid.VectorWeight
	expected.KbConfigConflictFields = nil

	assert.Equal(t, &expected, result)
}

// TestResolver_Resolve_SingleKB_InvalidJSON 测试 JSON 解析失败的情况
func TestResolver_Resolve_SingleKB_InvalidJSON(t *testing.T) {
	defaultOpts := newTestOptions()
	provider := &mockProvider{defaultOptions: defaultOpts}
	resolver := NewResolver(provider)

	// 非法 JSON
	invalidJSON := json.RawMessage([]byte(`{invalid}`))
	kb := &entity.KnowledgeBase{
		ID:                  3,
		BaseName:            "invalid",
		RetrievalConfigJson: invalidJSON,
		GraphRagConfigJson:  nil,
		RaptorConfigJson:    nil,
	}

	result := resolver.Resolve([]*entity.KnowledgeBase{kb})

	// 解析失败，配置被忽略，返回原 options
	assert.Equal(t, defaultOpts, result)
}

// TestResolver_Resolve_MultipleKB_AllSame 测试多个知识库配置一致
func TestResolver_Resolve_MultipleKB_AllSame(t *testing.T) {
	defaultOpts := newTestOptions()
	provider := &mockProvider{defaultOptions: defaultOpts}
	resolver := NewResolver(provider)

	// 定义两个配置，字段相同
	commonCfg := RuntimeConfig{
		VectorTopK:    intPtr(42),
		KeywordTopK:   intPtr(24),
		RerankEnabled: boolPtr(true),
		Hybrid: &HybridConfig{
			KeywordWeight: float64Ptr(0.75),
		},
	}
	jsonData1 := makeJSON(t, commonCfg)
	jsonData2 := makeJSON(t, commonCfg)

	kb1 := &entity.KnowledgeBase{ID: 4, BaseName: "kb1", RetrievalConfigJson: jsonData1}
	kb2 := &entity.KnowledgeBase{ID: 5, BaseName: "kb2", RetrievalConfigJson: jsonData2}

	result := resolver.Resolve([]*entity.KnowledgeBase{kb1, kb2})

	expected := *defaultOpts
	expected.VectorTopK = *commonCfg.VectorTopK
	expected.KeywordTopK = *commonCfg.KeywordTopK
	expected.RerankEnabled = *commonCfg.RerankEnabled
	expected.Hybrid.KeywordWeight = *commonCfg.Hybrid.KeywordWeight
	expected.KbConfigConflictFields = nil

	assert.Equal(t, &expected, result)
}

// TestResolver_Resolve_MultipleKB_Conflicts 测试多个知识库配置冲突
func TestResolver_Resolve_MultipleKB_Conflicts(t *testing.T) {
	defaultOpts := newTestOptions()
	provider := &mockProvider{defaultOptions: defaultOpts}
	resolver := NewResolver(provider)

	// 配置1
	cfg1 := RuntimeConfig{
		VectorTopK:    intPtr(10),
		KeywordTopK:   intPtr(20),
		RerankEnabled: boolPtr(true),
		Hybrid: &HybridConfig{
			VectorWeight: float64Ptr(0.5),
		},
	}
	// 配置2：VectorTopK 不同，KeywordTopK 相同，RerankEnabled 不同，Hybrid.VectorWeight 不同
	cfg2 := RuntimeConfig{
		VectorTopK:    intPtr(100),    // 冲突
		KeywordTopK:   intPtr(20),     // 一致
		RerankEnabled: boolPtr(false), // 冲突
		Hybrid: &HybridConfig{
			VectorWeight: float64Ptr(0.9), // 冲突
		},
	}
	json1 := makeJSON(t, cfg1)
	json2 := makeJSON(t, cfg2)

	kb1 := &entity.KnowledgeBase{ID: 6, BaseName: "kb1", RetrievalConfigJson: json1}
	kb2 := &entity.KnowledgeBase{ID: 7, BaseName: "kb2", RetrievalConfigJson: json2}

	result := resolver.Resolve([]*entity.KnowledgeBase{kb1, kb2})

	// 冲突字段保持默认值（未设置），冲突列表记录这些字段
	expected := *defaultOpts
	// 这些字段应保持默认值，不被覆盖
	// 但 KeywordTopK 一致，应设置为 20
	expected.KeywordTopK = 20
	// 其他冲突字段保持默认
	expected.KbConfigConflictFields = []string{"vectorTopK", "rerankEnabled", "hybrid.vectorWeight"}
	// 由于冲突，VectorTopK 仍为默认 10? 默认是10，但注意：defaultOpts.VectorTopK是10，而冲突后没有被设置，所以保持10。
	// 同样 RerankEnabled 默认 false，保持 false。
	// Hybrid.VectorWeight 默认 0.6，保持 0.6。
	assert.ElementsMatch(t, expected.KbConfigConflictFields, result.KbConfigConflictFields)
	// 比较其他字段
	expected.Hybrid = defaultOpts.Hybrid // 全部保持默认
	expected.VectorTopK = defaultOpts.VectorTopK
	expected.RerankEnabled = defaultOpts.RerankEnabled
	assert.Equal(t, &expected, result)
}

// TestResolver_Resolve_MultipleKB_SomeNil 测试部分配置为 nil 导致冲突
func TestResolver_Resolve_MultipleKB_SomeNil(t *testing.T) {
	defaultOpts := newTestOptions()
	provider := &mockProvider{defaultOptions: defaultOpts}
	resolver := NewResolver(provider)

	cfg1 := RuntimeConfig{
		VectorTopK: intPtr(5),
	}
	// cfg2 不设置 VectorTopK，即为 nil
	cfg2 := RuntimeConfig{
		KeywordTopK: intPtr(3),
	}
	json1 := makeJSON(t, cfg1)
	json2 := makeJSON(t, cfg2)

	kb1 := &entity.KnowledgeBase{ID: 8, BaseName: "kb1", RetrievalConfigJson: json1}
	kb2 := &entity.KnowledgeBase{ID: 9, BaseName: "kb2", RetrievalConfigJson: json2}

	result := resolver.Resolve([]*entity.KnowledgeBase{kb1, kb2})

	// VectorTopK 有值 (5) 和 nil 冲突，记录冲突，不设置
	// KeywordTopK 有值 (3) 和 nil? 实际上 cfg2 有 KeywordTopK，cfg1 没有，所以也是冲突
	expected := *defaultOpts
	expected.KbConfigConflictFields = []string{"vectorTopK", "keywordTopK"}
	// 其他保持默认
	assert.ElementsMatch(t, expected.KbConfigConflictFields, result.KbConfigConflictFields)
	// 确保 VectorTopK 和 KeywordTopK 保持默认
	assert.Equal(t, defaultOpts.VectorTopK, result.VectorTopK)
	assert.Equal(t, defaultOpts.KeywordTopK, result.KeywordTopK)
}

// TestResolver_Resolve_MultipleKB_HybridConflict 测试 Hybrid 内字段冲突
func TestResolver_Resolve_MultipleKB_HybridConflict(t *testing.T) {
	defaultOpts := newTestOptions()
	provider := &mockProvider{defaultOptions: defaultOpts}
	resolver := NewResolver(provider)

	cfg1 := RuntimeConfig{
		Hybrid: &HybridConfig{
			VectorWeight: float64Ptr(0.5),
		},
	}
	cfg2 := RuntimeConfig{
		Hybrid: &HybridConfig{
			VectorWeight: float64Ptr(0.7), // 冲突
		},
	}
	json1 := makeJSON(t, cfg1)
	json2 := makeJSON(t, cfg2)

	kb1 := &entity.KnowledgeBase{ID: 10, BaseName: "kb1", RetrievalConfigJson: json1}
	kb2 := &entity.KnowledgeBase{ID: 11, BaseName: "kb2", RetrievalConfigJson: json2}

	result := resolver.Resolve([]*entity.KnowledgeBase{kb1, kb2})

	expected := *defaultOpts
	expected.KbConfigConflictFields = []string{"hybrid.vectorWeight"}
	// Hybrid 其他字段保持默认
	assert.ElementsMatch(t, expected.KbConfigConflictFields, result.KbConfigConflictFields)
	assert.Equal(t, defaultOpts.Hybrid.VectorWeight, result.Hybrid.VectorWeight)
}

// TestResolver_Resolve_MultipleKB_AllSameHybrid 测试 Hybrid 一致
func TestResolver_Resolve_MultipleKB_AllSameHybrid(t *testing.T) {
	defaultOpts := newTestOptions()
	provider := &mockProvider{defaultOptions: defaultOpts}
	resolver := NewResolver(provider)

	commonHybrid := HybridConfig{
		VectorWeight:  float64Ptr(0.33),
		KeywordWeight: float64Ptr(0.33),
		TableWeight:   float64Ptr(0.34),
	}
	cfg1 := RuntimeConfig{Hybrid: &commonHybrid}
	cfg2 := RuntimeConfig{Hybrid: &commonHybrid}
	json1 := makeJSON(t, cfg1)
	json2 := makeJSON(t, cfg2)

	kb1 := &entity.KnowledgeBase{ID: 12, BaseName: "kb1", RetrievalConfigJson: json1}
	kb2 := &entity.KnowledgeBase{ID: 13, BaseName: "kb2", RetrievalConfigJson: json2}

	result := resolver.Resolve([]*entity.KnowledgeBase{kb1, kb2})

	expected := *defaultOpts
	expected.Hybrid.VectorWeight = *commonHybrid.VectorWeight
	expected.Hybrid.KeywordWeight = *commonHybrid.KeywordWeight
	expected.Hybrid.TableWeight = *commonHybrid.TableWeight
	expected.KbConfigConflictFields = nil
	assert.Equal(t, &expected, result)
}

// 辅助指针函数
func intPtr(i int) *int                          { return &i }
func boolPtr(b bool) *bool                       { return &b }
func durationPtr(d time.Duration) *time.Duration { return &d }
func float64Ptr(f float64) *float64              { return &f }
