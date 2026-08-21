package config

import (
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

type LocalConfig struct {
	conf *config.Config
}

func NewLocalConfig(svcCtx *svc.ServiceContext) *LocalConfig {
	return &LocalConfig{
		conf: svcCtx.Config,
	}
}

func (c *LocalConfig) CurrentOptions() *vo.RagRuntimeOptions {
	opts := vo.NewDefaultRagRuntimeOptions()
	if c == nil {
		return opts
	}

	// 覆盖所有字段
	rag := c.conf.Chat.Rag
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
	opts.Hybrid = &vo.HybridOptions{
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

func (c *LocalConfig) CurrentIndexingOptions() *vo.IndexingOptions {
	if c == nil || c.conf == nil {
		return &vo.IndexingOptions{}
	}

	chunk := c.conf.Chunk
	rag := c.conf.Chat.Rag

	return &vo.IndexingOptions{
		Chunk: vo.ChunkOptions{
			ChildRecursiveMaxChars:           chunk.RecursiveMaxChars,
			ChildRecursiveOverlapChars:       chunk.RecursiveOverlapChars,
			ChildSemanticMaxChars:            chunk.SemanticMaxChars,
			ChildSemanticMinChars:            chunk.SemanticMinChars,
			ChildSemanticSimilarityThreshold: chunk.SemanticSimilarityThreshold,
			ParentBlockMaxChars:              chunk.ParentChunkMaxChars,
			ParentBlockOverlapChars:          chunk.ParentChunkOverlapChars,
			ParentSemanticMaxChars:           chunk.ParentSemanticMaxChars,
			ParentSemanticMinChars:           chunk.ParentSemanticMinChars,
		},
		GraphRag: vo.GraphRagBuildOptions{
			GraphRagBuildEnabled: rag.GraphRag.Enabled,
		},
		Raptor: vo.RaptorBuildOptions{
			RaptorBuildEnabled:        rag.Raptor.Enabled,
			RaptorMaxClusterSize:      rag.Raptor.MaxClusterSize,
			RaptorMaxLevels:           rag.Raptor.MaxLevels,
			RaptorLlmSummaryEnabled:   rag.Raptor.LlmSummaryEnabled,
			RaptorSummaryQualityFloor: rag.Raptor.SummaryQualityFloor,
		},
	}
}
