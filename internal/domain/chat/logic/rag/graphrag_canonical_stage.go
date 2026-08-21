package rag

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// GraphRAGCanonicalStage GraphRAG canonical 观测阶段，为最终文档中的 GraphRAG 文档生成观测摘要
type GraphRAGCanonicalStage struct{}

// NewGraphRAGCanonicalStage 创建 GraphRAG canonical 观测阶段
func NewGraphRAGCanonicalStage() *GraphRAGCanonicalStage {
	return &GraphRAGCanonicalStage{}
}

func (s *GraphRAGCanonicalStage) Name() string {
	return "GraphRAGCanonical"
}

// Execute 为最终文档中的 GraphRAG 文档生成 canonical 观测笔记，并添加检索完成日志
func (s *GraphRAGCanonicalStage) Execute(_ context.Context, state *RetrievalState) error {
	s.appendGraphRagCanonicalNotes(state.RetrievalResult, state.Input.SubQuestionIndex, state.Input.SubQuestion, state.FinalDocs)

	state.RetrievalResult.AddRetrievalNotef("子问题%d检索完成：%s，final=%d",
		state.Input.SubQuestionIndex, s.summarizeChannelResults(state.ChannelResults), len(state.FinalDocs))
	return nil
}

// graphRagCanonicalObservation GraphRAG canonical 观测记录
type graphRagCanonicalObservation struct {
	text          string
	priority      float64
	originalIndex int
}

// appendGraphRagCanonicalNotes 为最终文档中的 GraphRAG 文档生成 canonical 观测笔记
func (s *GraphRAGCanonicalStage) appendGraphRagCanonicalNotes(ragCtx *vo.RetrievalResult, subQuestionIndex int, subQuestion string, finalDocs []*vo.DocumentChunk) {
	if len(finalDocs) == 0 {
		return
	}

	var observations []graphRagCanonicalObservation
	for idx, doc := range finalDocs {
		if !s.isGraphRagDocument(doc) {
			continue
		}
		obs := s.buildGraphRagObservation(doc, idx)
		observations = append(observations, obs)
	}

	if len(observations) == 0 {
		return
	}

	// 按优先级降序、原始索引升序排序，取前 4 条
	sort.Slice(observations, func(i, j int) bool {
		if observations[i].priority != observations[j].priority {
			return observations[i].priority > observations[j].priority
		}
		return observations[i].originalIndex < observations[j].originalIndex
	})
	limit := min(4, len(observations))
	parts := make([]string, limit)
	for i := 0; i < limit; i++ {
		parts[i] = observations[i].text
	}
	summary := strings.Join(parts, "；")
	ragCtx.AddRetrievalNotef("子问题%d GraphRAG canonical 观测：%s", subQuestionIndex, summary)
	logx.Infof("GraphRAG canonical 观测: subQuestionIndex=%d, subQuestion='%s', observations=%s",
		subQuestionIndex, subQuestion, summary)
}

// isGraphRagDocument 判断文档是否为 GraphRAG 来源
func (s *GraphRAGCanonicalStage) isGraphRagDocument(doc *vo.DocumentChunk) bool {
	if doc == nil {
		return false
	}
	return doc.SourceType == "GRAPH_RAG" ||
		doc.Channel == "graph_rag" ||
		strings.Contains(doc.Channel, "graph_rag") ||
		strings.Contains(doc.SourceType, "GRAPH_RAG")
}

// buildGraphRagObservation 为单个 GraphRAG 文档构建观测记录
func (s *GraphRAGCanonicalStage) buildGraphRagObservation(doc *vo.DocumentChunk, originalIndex int) graphRagCanonicalObservation {
	var builder strings.Builder

	// 构建 canonical 标识
	canonicalName := utils.FirstNonBlank(doc.DocumentName, doc.Title, "unknown")
	builder.WriteString(canonicalName)

	// 补充文档统计信息
	score := doc.Score
	priority := math.Max(0, score)
	priority -= float64(originalIndex) * 0.0001

	return graphRagCanonicalObservation{
		text:          builder.String(),
		priority:      priority,
		originalIndex: originalIndex,
	}
}

// summarizeChannelResults 摘要每个检索渠道的文档数量
func (s *GraphRAGCanonicalStage) summarizeChannelResults(channelResults []*RetrievalChannelResult) string {
	if len(channelResults) == 0 {
		return "没有启用任何检索通道"
	}
	parts := make([]string, 0, len(channelResults))
	for _, result := range channelResults {
		parts = append(parts, fmt.Sprintf("%s=%d", result.Name, len(result.AcceptedDocuments)))
	}
	return strings.Join(parts, "，")
}
