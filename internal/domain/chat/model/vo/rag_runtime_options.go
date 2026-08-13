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
