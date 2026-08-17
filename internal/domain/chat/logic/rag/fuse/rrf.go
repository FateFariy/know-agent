package fuse

import (
	"context"
	"math"
	"slices"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// RRF 融合常量
const rrfK = 60

// MetadataBoostCalculator 计算文档元数据提升值的函数类型
// 接收文档块和检索计划，返回非负的元数据提升值
type MetadataBoostCalculator func(doc *vo.DocumentChunk, plan *vo.RetrievalPlan) float64

// RRFFusion 基于加权 RRF 的多通道混合融合器
//
// 对每个通道结果依次计算：
//   - RRF 排名分数（基于 rank 位置）
//   - 归一化原始分数（基于通道内最大分数）
//   - 元数据提升值（来自 build-time 索引/结构化特征）
//
// 按通道权重加权求和后，得到最终融合分数，并选取候选窗口内的文档。
type RRFFusion struct {
	// calcMetadataBoost 外部注入的元数据提升计算函数，允许自定义提升策略。
	// 默认使用 DefaultMetadataBoost。
	calcMetadataBoost MetadataBoostCalculator
}

// NewRRFFusion 创建 RRF 融合器。
// calc 为可选的元数据提升计算函数，传 nil 则使用 DefaultMetadataBoost。
func NewRRFFusion(calc MetadataBoostCalculator) *RRFFusion {
	if calc == nil {
		calc = DefaultMetadataBoost
	}
	return &RRFFusion{calcMetadataBoost: calc}
}

// candidateHolder 融合过程中单个候选文档的累积状态
type candidateHolder struct {
	document      *vo.DocumentChunk
	channels      []string
	rrfScore      float64
	rankScore     float64
	originalScore float64
	metadataBoost float64
	score         float64
	vectorScore   *float64
	keywordScore  *float64
}

// newCandidateHolder 从文档创建候选持有者，保留文档引用并清空融合相关临时字段
func newCandidateHolder(doc *vo.DocumentChunk) *candidateHolder {
	return &candidateHolder{
		document: doc,
		channels: make([]string, 0, 2),
	}
}

// Fuse 对多通道检索结果执行加权混合融合
//
// 处理流程：
//  1. 解析各通道的最大分数，用于后续归一化
//  2. 遍历各通道结果，对每个文档累积加权 RRF 分数、归一化原始分数和元数据提升
//  3. 按通道权重、排名权重和原始分数权重计算最终融合分数
//  4. 按融合分数降序排序，截取候选窗口（plan.CandidateWindow）内的文档
//  5. 回写融合分数到文档的 RRFScore 字段
func (f *RRFFusion) Fuse(_ context.Context, results []*rag.RetrievalChannelResult, plan *vo.RetrievalPlan) vo.DocumentChunks {
	if len(results) == 0 {
		return nil
	}

	holders := make(map[string]*candidateHolder)
	channelMaxScoreMap := resolveChannelMaxScoreMap(results)

	for _, channelResult := range results {
		f.accumulateWeightedHybrid(channelResult, holders, channelMaxScoreMap, plan)
	}

	sortedHolders := f.sortAndFinalize(holders, plan)
	selected := f.selectHybridCandidates(sortedHolders, plan)

	// 回写融合分数到文档
	for _, holder := range selected {
		holder.document.RRFScore = holder.score
		holder.document.Channel = resolveChannelLabel(holder.channels)
	}

	result := make(vo.DocumentChunks, 0, len(selected))
	for _, holder := range selected {
		result = append(result, holder.document)
	}
	return result
}

// resolveChannelMaxScoreMap 计算每个通道内文档的最大分数，用于后续归一化
func resolveChannelMaxScoreMap(results []*rag.RetrievalChannelResult) map[string]float64 {
	maxScoreMap := make(map[string]float64, len(results))
	for _, result := range results {
		if result == nil || len(result.Documents) == 0 {
			continue
		}
		maxScore := 0.0
		for _, doc := range result.Documents {
			if doc != nil && doc.Score > maxScore {
				maxScore = doc.Score
			}
		}
		maxScoreMap[result.ChannelName] = math.Max(maxScore, 0)
	}
	return maxScoreMap
}

// accumulateWeightedHybrid 遍历单个通道结果，对每个文档累积加权融合分数
func (f *RRFFusion) accumulateWeightedHybrid(
	channelResult *rag.RetrievalChannelResult,
	holders map[string]*candidateHolder,
	channelMaxScoreMap map[string]float64,
	plan *vo.RetrievalPlan,
) {
	if channelResult == nil || len(channelResult.Documents) == 0 {
		return
	}

	channelWeight := resolveChannelWeight(channelResult.ChannelName, plan)
	channelMaxScore := channelMaxScoreMap[channelResult.ChannelName]

	for rank, doc := range channelResult.Documents {
		if doc == nil {
			continue
		}

		// RRF 排名分数
		rrfScore := 1.0 / float64(rrfK+rank+1)
		normalizedRankScore := float64(rrfK+1) * rrfScore

		// 归一化原始分数
		normalizedOriginalScore := normalizeOriginalScore(doc.Score, channelMaxScore)

		// 获取或创建候选持有者，以文档 ID 为融合标识
		holder, exists := holders[doc.ID]
		if !exists {
			holder = newCandidateHolder(doc)
			holders[doc.ID] = holder
		} else {
			// 后续通道的同文档最多保留 GraphRAG 元数据
			mergeGraphRagMetadata(holder, doc)
		}

		// 累积加权分数
		rankWeight := hybridRankWeight(plan)
		originalScoreWeight := hybridOriginalScoreWeight(plan)

		holder.rrfScore += rrfScore
		holder.rankScore += channelWeight * rankWeight * normalizedRankScore
		holder.originalScore += channelWeight * originalScoreWeight * normalizedOriginalScore
		holder.metadataBoost = math.Max(holder.metadataBoost, f.calcMetadataBoost(doc, plan))

		// 记录通道来源
		if !slices.Contains(holder.channels, channelResult.ChannelName) {
			holder.channels = append(holder.channels, channelResult.ChannelName)
		}

		// 保留各通道原始分数
		if channelResult.ChannelName == enum.RetrievalChannelVector {
			v := doc.Score
			holder.vectorScore = &v
		}
		if channelResult.ChannelName == enum.RetrievalChannelKeyword {
			v := doc.Score
			holder.keywordScore = &v
		}
	}
}

// sortAndFinalize 计算最终融合分数并按分数降序排序
func (f *RRFFusion) sortAndFinalize(holders map[string]*candidateHolder, plan *vo.RetrievalPlan) []*candidateHolder {
	result := make([]*candidateHolder, 0, len(holders))
	for _, holder := range holders {
		f.finishHybridScore(holder, plan)
		result = append(result, holder)
	}
	slices.SortFunc(result, func(a, b *candidateHolder) int {
		if a.score > b.score {
			return -1
		}
		if a.score < b.score {
			return 1
		}
		return 0
	})
	return result
}

// finishHybridScore 计算候选文档的最终融合分数。
//
// 最终分数 = 加权排名分数 + 加权原始分数 + 元数据提升权重 * min(元数据提升, 最大元数据提升)
func (f *RRFFusion) finishHybridScore(holder *candidateHolder, plan *vo.RetrievalPlan) {
	metadataBoostWeight := hybridMetadataBoostWeight(plan)
	maxMetadataBoost := hybridMaxMetadataBoost(plan)
	holder.score = holder.rankScore + holder.originalScore + metadataBoostWeight*math.Min(holder.metadataBoost, maxMetadataBoost)
}

// selectHybridCandidates 从排序后的候选列表中选取候选窗口内的文档。
func (f *RRFFusion) selectHybridCandidates(sortedHolders []*candidateHolder, plan *vo.RetrievalPlan) []*candidateHolder {
	candidateTopK := resolveCandidateWindow(plan)
	if candidateTopK <= 0 || len(sortedHolders) == 0 {
		return nil
	}
	if candidateTopK >= len(sortedHolders) {
		return sortedHolders
	}
	return sortedHolders[:candidateTopK]
}

// normalizeOriginalScore 对原始分数进行通道内归一化，返回 [0, 1] 范围的值。
func normalizeOriginalScore(originalScore, channelMaxScore float64) float64 {
	if originalScore <= 0 || channelMaxScore <= 0 {
		return 0
	}
	return math.Min(1, originalScore/channelMaxScore)
}

// resolveChannelWeight 解析通道权重，返回非负值
func resolveChannelWeight(channelName string, plan *vo.RetrievalPlan) float64 {
	if plan == nil || channelName == "" {
		return 1
	}
	for _, ch := range plan.Channels {
		if ch != nil && ch.Channel == channelName {
			return math.Max(0, ch.Weight)
		}
	}
	return 1
}

// resolveChannelLabel 根据通道数量决定标签：多通道返回 "hybrid"，否则返回首个通道名。
func resolveChannelLabel(channels []string) string {
	if len(channels) > 1 {
		return "hybrid"
	}
	if len(channels) == 1 {
		return channels[0]
	}
	return ""
}

// resolveCandidateWindow 解析候选窗口大小，返回非负值。
func resolveCandidateWindow(plan *vo.RetrievalPlan) int {
	if plan == nil {
		return 0
	}
	if plan.CandidateWindow > 0 {
		return plan.CandidateWindow
	}
	return 0
}

// hybridRankWeight 获取排名权重。
func hybridRankWeight(plan *vo.RetrievalPlan) float64 {
	if plan == nil || plan.RankFeatures == nil {
		return 1
	}
	return math.Max(0, plan.RankFeatures.RankWeight)
}

// hybridOriginalScoreWeight 获取原始分数权重。
func hybridOriginalScoreWeight(plan *vo.RetrievalPlan) float64 {
	if plan == nil || plan.RankFeatures == nil {
		return 0.08
	}
	return math.Max(0, plan.RankFeatures.OriginalScoreWeight)
}

// hybridMetadataBoostWeight 获取元数据提升权重。
func hybridMetadataBoostWeight(plan *vo.RetrievalPlan) float64 {
	if plan == nil || plan.RankFeatures == nil {
		return 0.04
	}
	return math.Max(0, plan.RankFeatures.MetadataBoostWeight)
}

// hybridMaxMetadataBoost 获取最大元数据提升值。
func hybridMaxMetadataBoost(plan *vo.RetrievalPlan) float64 {
	if plan == nil || plan.RankFeatures == nil {
		return 1
	}
	return math.Max(0, plan.RankFeatures.MaxMetadataBoost)
}

// DefaultMetadataBoost 默认的元数据提升计算函数。
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

// mergeGraphRagMetadata 合并 GraphRAG 元数据：当新文档的优先级更高时，替换持有者的元数据。
// 使用 metadataPriority 决定哪个文档的元数据更优，并逐字段补充缺失值。
func mergeGraphRagMetadata(holder *candidateHolder, candidate *vo.DocumentChunk) {
	if holder == nil || candidate == nil {
		return
	}
	if !isGraphRagSource(candidate) {
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

// isGraphRagSource 判断文档是否来自 GraphRAG 通道。
func isGraphRagSource(doc *vo.DocumentChunk) bool {
	if doc == nil {
		return false
	}
	return doc.Channel == enum.RetrievalChannelGraphRAG ||
		doc.SourceType == "GRAPH_RAG"
}

// graphRagPriority 计算文档的 GraphRAG 元数据优先级。
// 基于来源类型、文档名称等综合判断，仅用于 GraphRAG 元数据合并决策。
func graphRagPriority(doc *vo.DocumentChunk) float64 {
	if doc == nil || !isGraphRagSource(doc) {
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
