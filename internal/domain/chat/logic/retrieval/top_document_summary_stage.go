package retrieval

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

// TopDocumentSummaryStage 检索结果摘要阶段，为最终文档生成 Top 文档摘要笔记
type TopDocumentSummaryStage struct {
	topN int
}

// NewTopDocumentSummaryStage 创建检索结果摘要阶段，默认取前 4 条
func NewTopDocumentSummaryStage() *TopDocumentSummaryStage {
	return &TopDocumentSummaryStage{
		topN: 4,
	}
}

func (s *TopDocumentSummaryStage) Name() string {
	return "TopDocumentSummary"
}

// Execute 为最终文档生成摘要笔记，并添加检索完成日志
func (s *TopDocumentSummaryStage) Execute(_ context.Context, state *RetrievalState) error {
	s.appendTopDocumentSummaries(state.RetrievalResult, state.Input.SubQuestionIndex, state.Input.SubQuestion, state.FinalDocs)

	state.RetrievalResult.AddRetrievalNotef("子问题%d检索完成：%s，final=%d",
		state.Input.SubQuestionIndex, s.summarizeChannelResults(state.ChannelResults), len(state.FinalDocs))
	return nil
}

// documentSummaryNote 文档摘要笔记（内部临时结构）
type documentSummaryNote struct {
	text          string
	priority      float64
	originalIndex int
}

// appendTopDocumentSummaries 为最终文档生成摘要笔记
func (s *TopDocumentSummaryStage) appendTopDocumentSummaries(ragCtx *vo.RetrievalResult, subQuestionIndex int, subQuestion string, finalDocs []*vo.DocumentChunk) {
	if len(finalDocs) == 0 {
		return
	}

	// 为所有文档构建笔记
	notes := make([]*documentSummaryNote, 0, len(finalDocs))
	for idx, doc := range finalDocs {
		notes = append(notes, s.buildDocumentSummaryNote(doc, idx))
	}

	// 按优先级（分数）降序、原始索引升序排序
	sort.Slice(notes, func(i, j int) bool {
		if notes[i].priority != notes[j].priority {
			return notes[i].priority > notes[j].priority
		}
		return notes[i].originalIndex < notes[j].originalIndex
	})

	// 取前 N 条
	limit := min(s.topN, len(notes))
	parts := make([]string, limit)
	for i := 0; i < limit; i++ {
		parts[i] = notes[i].text
	}
	summary := strings.Join(parts, "；")

	// 记录到检索结果和日志
	ragCtx.AddRetrievalNotef("子问题%d 检索结果摘要：%s", subQuestionIndex, summary)
	logx.Infof("检索结果摘要: subQuestionIndex=%d, subQuestion='%s', summaries=%s", subQuestionIndex, subQuestion, summary)
}

// buildDocumentSummaryNote 为单个文档构建摘要笔记
func (s *TopDocumentSummaryStage) buildDocumentSummaryNote(doc *vo.DocumentChunk, originalIndex int) *documentSummaryNote {
	var builder strings.Builder

	// 提取文档标识
	docName := utils.FirstNonBlank(doc.DocumentName, doc.Title, "unknown")
	builder.WriteString(docName)

	// 使用文档分数作为优先级，加微小偏移量保持原始顺序
	priority := math.Max(0, doc.Score)
	priority -= float64(originalIndex) * 0.0001

	return &documentSummaryNote{
		text:          builder.String(),
		priority:      priority,
		originalIndex: originalIndex,
	}
}

// summarizeChannelResults 摘要每个检索渠道的文档数量
func (s *TopDocumentSummaryStage) summarizeChannelResults(channelResults []*RetrievalChannelResult) string {
	if len(channelResults) == 0 {
		return "没有启用任何检索通道"
	}
	parts := make([]string, 0, len(channelResults))
	for _, result := range channelResults {
		parts = append(parts, fmt.Sprintf("%s=%d", result.Name, len(result.AcceptedDocuments)))
	}
	return strings.Join(parts, "，")
}
