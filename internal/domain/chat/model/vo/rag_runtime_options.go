package vo

import (
	"time"

	"github.com/swiftbit/know-agent/internal/config"
)

// HybridOptions 混合检索权重配置
type HybridOptions struct {
	VectorWeight        float64
	KeywordWeight       float64
	TableWeight         float64
	GraphRagWeight      float64
	RaptorWeight        float64
	RankWeight          float64
	OriginalScoreWeight float64
	MetadataBoostWeight float64
	MaxMetadataBoost    float64
}

func NewDefaultHybridOptions() *HybridOptions {
	return &HybridOptions{
		VectorWeight:        1.0,
		KeywordWeight:       1.0,
		TableWeight:         1.2,
		GraphRagWeight:      1.1,
		RaptorWeight:        1.05,
		RankWeight:          1.0,
		OriginalScoreWeight: 0.08,
		MetadataBoostWeight: 0.04,
		MaxMetadataBoost:    1.0,
	}
}

// RagRuntimeOptions RAG 运行时配置
type RagRuntimeOptions struct {
	VectorTopK                int
	KeywordTopK               int
	GraphRagTopK              int
	GraphRagMaxHops           int
	RaptorTopK                int
	RaptorSourceChunkTopK     int
	CandidateTopK             int
	RerankCandidateTopK       int
	FinalTopK                 int
	RerankEnabled             bool
	ChannelTimeout            time.Duration
	SubQuestionTimeout        time.Duration
	MinVectorSimilarity       float64
	KeywordRelativeScoreFloor float64
	KeywordChannelEnabled     bool
	TableChannelEnabled       bool
	GraphRagChannelEnabled    bool
	RaptorChannelEnabled      bool
	Hybrid                    *HybridOptions
	KbConfigConflictFields    []string
}

// NewDefaultRagRuntimeOptions 返回 RagRuntimeOptions 的默认值
func NewDefaultRagRuntimeOptions() *RagRuntimeOptions {
	return &RagRuntimeOptions{
		VectorTopK:                10,
		KeywordTopK:               10,
		GraphRagTopK:              5,
		GraphRagMaxHops:           2,
		RaptorTopK:                5,
		RaptorSourceChunkTopK:     3,
		CandidateTopK:             40,
		RerankCandidateTopK:       24,
		FinalTopK:                 6,
		RerankEnabled:             true,
		ChannelTimeout:            12 * time.Second,
		SubQuestionTimeout:        40 * time.Second,
		MinVectorSimilarity:       0.45,
		KeywordRelativeScoreFloor: 0.35,
		KeywordChannelEnabled:     true,
		TableChannelEnabled:       true,
		GraphRagChannelEnabled:    true,
		RaptorChannelEnabled:      true,
		Hybrid:                    NewDefaultHybridOptions(),
		KbConfigConflictFields:    []string{},
	}
}

// FromChatRagConfig 从 ChatRagConfig 转换为 RagRuntimeOptions
func FromChatRagConfig(config *config.Config) *RagRuntimeOptions {
	if config == nil {
		return NewDefaultRagRuntimeOptions()
	}

	opts := NewDefaultRagRuntimeOptions() // 先获取默认值，再覆盖

	// 覆盖所有字段
	rag := config.Chat.Rag
	opts.VectorTopK = rag.Vector.TopK
	opts.KeywordTopK = rag.Keyword.TopK
	opts.GraphRagTopK = rag.GraphRag.TopK
	opts.GraphRagMaxHops = rag.GraphRag.MaxHops
	opts.RaptorTopK = rag.Raptor.TopK
	opts.RaptorSourceChunkTopK = rag.Raptor.SourceChunkTopK
	opts.CandidateTopK = rag.FinalTopK
	opts.RerankCandidateTopK = rag.Rerank.TopN
	opts.FinalTopK = rag.FinalTopK
	opts.RerankEnabled = rag.Rerank.Enabled
	opts.ChannelTimeout = rag.ChannelTimeout
	opts.SubQuestionTimeout = rag.SubQuestionTimeout
	opts.MinVectorSimilarity = rag.Vector.MinSimilarity
	opts.KeywordRelativeScoreFloor = rag.Keyword.RelativeScoreFloor
	opts.KeywordChannelEnabled = rag.Keyword.Enabled
	opts.TableChannelEnabled = rag.Table.Enabled
	opts.GraphRagChannelEnabled = rag.GraphRag.Enabled
	opts.RaptorChannelEnabled = rag.Raptor.Enabled
	hybrid := rag.Hybrid
	opts.Hybrid = &HybridOptions{
		VectorWeight:        hybrid.VectorWeight,
		KeywordWeight:       hybrid.KeywordWeight,
		TableWeight:         hybrid.TableWeight,
		GraphRagWeight:      hybrid.GraphRagWeight,
		RaptorWeight:        hybrid.RaptorWeight,
		RankWeight:          hybrid.RankWeight,
		OriginalScoreWeight: hybrid.OriginalScoreWeight,
		MetadataBoostWeight: hybrid.MetadataBoostWeight,
		MaxMetadataBoost:    hybrid.MaxMetadataBoost,
	}

	return opts
}

// DeepCopy 深拷贝 RagRuntimeOptions
func (o *RagRuntimeOptions) DeepCopy() *RagRuntimeOptions {
	if o == nil {
		return nil
	}

	var hybrid HybridOptions
	var rag RagRuntimeOptions
	if o.Hybrid != nil {
		hybrid = *o.Hybrid
	}
	rag = *o

	rag.Hybrid = &hybrid
	rag.KbConfigConflictFields = append([]string{}, o.KbConfigConflictFields...)

	return &rag
}
