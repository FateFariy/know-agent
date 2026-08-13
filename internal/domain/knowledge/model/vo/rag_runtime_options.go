package vo

import (
	"time"
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
