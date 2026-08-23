package fuse

import (
	"cmp"
	"context"
	"math"
	"slices"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// RRF 融合常量
const rrfK = 60

// RRFFusion 基于加权 RRF 的多通道混合融合器
//
// 对每个通道结果依次计算：
//   - RRF 排名分数（基于 rank 位置）
//   - 归一化原始分数（基于通道内最大分数）
//   - 元数据提升值（来自 build-time 索引/结构化特征）
//
// 按通道权重加权求和后，得到最终融合分数，并选取候选窗口内的文档
type RRFFusion struct {
}

// NewRRFFusion 创建 RRF 融合器
func NewRRFFusion() *RRFFusion {
	return &RRFFusion{}
}

// Fuse 对多通道检索结果执行加权混合融合
//
// 处理流程：
//  1. 解析各通道的最大分数，用于后续归一化
//  2. 遍历各通道结果，对每个文档累积加权 RRF 分数、归一化原始分数和元数据提升
//  3. 按通道权重、排名权重和原始分数权重计算最终融合分数
//  4. 按融合分数降序排序，截取候选窗口（plan.CandidateTopK）内的文档
//  5. 回写融合分数到文档的 RRFScore 字段
func (f *RRFFusion) Fuse(_ context.Context, results []*retrieval.RetrievalChannelResult, plan *vo.RetrievalPlan) []*vo.DocumentChunk {
	if len(results) == 0 || plan == nil {
		return nil
	}

	holders := f.accumulateWeightedHybrid(results, plan)
	sortedHolders := f.sortAndFinalize(holders, plan)
	selected := f.selectHybridCandidates(sortedHolders, plan)

	// 回写融合分数到文档
	result := make([]*vo.DocumentChunk, 0, len(selected))
	for _, holder := range selected {
		document := holder.document
		document.Score = holder.score
		document.RRFScore = holder.rrfScore
		if len(holder.channels) > 0 {
			document.Channel = utils.Ternary(len(holder.channels) > 1, "hybrid", holder.channels[0])
		}
		result = append(result, document)
	}
	return result
}

// accumulateWeightedHybrid 遍历单个通道结果，对每个文档累积加权融合分数
func (f *RRFFusion) accumulateWeightedHybrid(channelResults []*retrieval.RetrievalChannelResult, plan *vo.RetrievalPlan) map[string]*candidateHolder {
	holders := make(map[string]*candidateHolder)
	channelMap := utils.MapBy(plan.Channels, func(channel *vo.RetrievalChannelPlan) (string, *vo.RetrievalChannelPlan) {
		return channel.Name, channel
	})
	for _, channelResult := range channelResults {
		if channelResult == nil || len(channelResult.AcceptedDocuments) == 0 {
			continue
		}
		channel := channelMap[channelResult.Name]
		for rank, doc := range channelResult.AcceptedDocuments {
			if doc == nil {
				continue
			}

			// RRF 排名分数
			rrfScore := 1.0 / float64(rrfK+rank+1)
			normalizedRankScore := float64(rrfK+1) * rrfScore

			// 获取或创建候选持有者，以文档 ID 为融合标识 todo 标识待优化，暂用文档 ID
			holder, exists := holders[doc.ID]
			if !exists {
				holder = newCandidateHolder(doc)
				holders[doc.ID] = holder
			} else {
				// 后续通道的同文档最多保留 GraphRAG 元数据
				mergeGraphRagMetadata(holder, doc)
			}

			// 累积加权分数
			rankWeight := 1.0
			originalScoreWeight := 0.08
			if plan.RankFeatures != nil {
				rankWeight = math.Max(plan.RankFeatures.RankWeight, 0)
				originalScoreWeight = math.Max(plan.RankFeatures.OriginalScoreWeight, 0)
			}

			// 通道权重
			channelWeight := channel.Weight

			holder.rrfScore += rrfScore
			holder.rankScore += channelWeight * rankWeight * normalizedRankScore
			holder.originalScore += channelWeight * originalScoreWeight * doc.NormalizedScore
			holder.metadataBoost = math.Max(holder.metadataBoost, doc.MetadataBoost)

			// 记录通道来源
			if !slices.Contains(holder.channels, channelResult.Name) {
				holder.channels = append(holder.channels, channelResult.Name)
			}

			// 保留各通道原始分数
			if channelResult.Name == enum.RetrievalChannelVector {
				holder.vectorScore = doc.Score
			}
			if channelResult.Name == enum.RetrievalChannelKeyword {
				holder.keywordScore = doc.Score
			}
		}
	}

	return holders
}

// sortAndFinalize 计算最终融合分数并按分数降序排序
func (f *RRFFusion) sortAndFinalize(holders map[string]*candidateHolder, plan *vo.RetrievalPlan) []*candidateHolder {
	result := make([]*candidateHolder, 0, len(holders))
	for _, holder := range holders {
		holder.calculateHybridScore(plan)
		result = append(result, holder)
	}
	slices.SortFunc(result, func(a, b *candidateHolder) int { return cmp.Compare(a.score, b.score) })
	return result
}

// selectHybridCandidates 从排序后的候选列表中选取候选窗口内的文档
func (f *RRFFusion) selectHybridCandidates(sortedHolders []*candidateHolder, plan *vo.RetrievalPlan) []*candidateHolder {
	candidateTopK := max(plan.CandidateTopK, 0)
	if candidateTopK <= 0 || len(sortedHolders) == 0 {
		return nil
	}
	if candidateTopK >= len(sortedHolders) {
		return sortedHolders
	}
	return sortedHolders[:candidateTopK]
}

// candidateHolder 融合过程中单个候选文档的累积状态
type candidateHolder struct {
	document      *vo.DocumentChunk // 原始文档块
	channels      []string          // 参与融合的通道标签
	rrfScore      float64           // 累计 RRF 排名分数
	rankScore     float64           // 累计加权排名分数
	originalScore float64           // 累计加权原始分数
	metadataBoost float64           // 元数据提升值
	score         float64           // 最终融合分数
	vectorScore   float64           // 向量分数，可选
	keywordScore  float64           // 关键词分数，可选
}

// newCandidateHolder 从文档创建候选持有者，保留文档引用并清空融合相关临时字段
func newCandidateHolder(doc *vo.DocumentChunk) *candidateHolder {
	clone := *doc
	return &candidateHolder{
		document: &clone,
		channels: make([]string, 0, 2),
	}
}

// calculateHybridScore 计算候选文档的最终融合分数【加权排名分数 + 加权原始分数 + 元数据提升权重 * min(元数据提升, 最大元数据提升)】
func (h *candidateHolder) calculateHybridScore(plan *vo.RetrievalPlan) {
	metadataBoostWeight := 0.04
	maxMetadataBoost := 1.0
	if plan != nil && plan.RankFeatures != nil {
		metadataBoostWeight = math.Max(0, plan.RankFeatures.MetadataBoostWeight)
		maxMetadataBoost = math.Max(0, plan.RankFeatures.MaxMetadataBoost)
	}
	h.score = h.rankScore + h.originalScore + metadataBoostWeight*math.Min(h.metadataBoost, maxMetadataBoost)
}

// mergeGraphRagMetadata 合并 GraphRAG 元数据：当新文档的优先级更高时，替换持有者的元数据。
// 使用 metadataPriority 决定哪个文档的元数据更优，并逐字段补充缺失值。
func mergeGraphRagMetadata(holder *candidateHolder, candidate *vo.DocumentChunk) {
	if holder == nil || candidate == nil {
		return
	}
	if !candidate.IsGraphRagSource() {
		return
	}

	// 按优先级选择元数据生产者
	if graphRagPriority(candidate) > graphRagPriority(holder.document) {
		holder.document.SourceType = candidate.SourceType
		holder.document.DocumentName = candidate.DocumentName
		holder.document.ToolName = candidate.ToolName
		holder.document.Url = candidate.Url
		holder.document.Title = candidate.Title
	}
}

// graphRagPriority 计算文档的 GraphRAG 元数据优先级。
// 基于来源类型、文档名称等综合判断，仅用于 GraphRAG 元数据合并决策。
func graphRagPriority(doc *vo.DocumentChunk) float64 {
	if doc == nil || !doc.IsGraphRagSource() {
		return 0
	}
	priority := 0.0
	if doc.SourceType != "" {
		priority += 1
	}
	if doc.DocumentName != "" {
		priority += 1
	}
	if doc.ToolName != "" {
		priority += 0.5
	}
	return priority
}
