package rag

import (
	"context"
	"math"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// FusionStage 多通道融合阶段
type FusionStage struct {
	fusion Fusion
}

func NewFusionStage(fusion Fusion) *FusionStage {
	return &FusionStage{
		fusion: fusion,
	}
}

func (s *FusionStage) Name() string {
	return "Fusion"
}

// Execute 对过滤后的通道结果执行加权混合融合，结果写入 state.FusedDocs
func (s *FusionStage) Execute(ctx context.Context, state *RetrievalState) error {
	if len(state.ChannelResults) == 0 {
		return nil
	}
	s.preprocess(state)
	state.FusedDocs = s.fusion.Fuse(ctx, state.ChannelResults, state.Plan)
	return nil
}

func (s *FusionStage) preprocess(state *RetrievalState) {
	if len(state.ChannelResults) == 0 {
		return
	}
	// 计算各通道最大分数
	maxScoreMap := resolveChannelMaxScoreMap(state.ChannelResults)
	for _, chResult := range state.ChannelResults {
		if chResult == nil {
			continue
		}
		for _, doc := range chResult.AcceptedDocuments {
			if doc == nil {
				continue
			}
			// 归一化原始分数
			doc.NormalizeScore(maxScoreMap[chResult.Name])
			// 元数据提升
			doc.MetadataBoost = DefaultMetadataBoost(doc, state.Plan)
		}
	}
	return
}

// resolveChannelMaxScoreMap 计算每个通道内文档的最大分数，用于后续归一化
func resolveChannelMaxScoreMap(results []*RetrievalChannelResult) map[string]float64 {
	maxScoreMap := make(map[string]float64, len(results))
	for _, result := range results {
		if result == nil || len(result.AcceptedDocuments) == 0 {
			continue
		}
		maxScore := 0.0
		for _, doc := range result.AcceptedDocuments {
			if doc != nil && doc.Score > maxScore {
				maxScore = doc.Score
			}
		}
		maxScoreMap[result.Name] = math.Max(maxScore, 0)
	}
	return maxScoreMap
}

// DefaultMetadataBoost 默认的元数据提升计算函数
//
// 基于文档来源类型（chunk type）计算提升值：
//   - table 类型：+0.03
//   - image/figure 类型：+0.02
func DefaultMetadataBoost(doc *vo.DocumentChunk, _ *vo.RetrievalPlan) float64 {
	if doc == nil {
		return 0
	}
	boost := chunkTypeBoost(doc.SourceType)
	return math.Min(boost, 1)
}

// chunkTypeBoost 根据文档来源类型计算基础提升值。
func chunkTypeBoost(sourceType string) float64 {
	switch sourceType {
	case "table":
		return 0.03
	case "image", "figure":
		return 0.02
	default:
		return 0
	}
}
