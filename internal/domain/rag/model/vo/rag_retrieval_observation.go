package vo

import (
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/convertor"

	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// ChannelExecutionView 渠道执行观测视图
// 记录每个检索通道的执行统计信息，包括召回数量、接受数量、分数等
type ChannelExecutionView struct {
	ExchangeId         int64   `json:"exchangeId"`         // 对话交换ID
	TraceId            string  `json:"traceId"`            // 追踪ID
	SubQuestionIndex   int     `json:"subQuestionIndex"`   // 子问题索引
	SubQuestion        string  `json:"subQuestion"`        // 子问题文本
	ChannelType        string  `json:"channelType"`        // 渠道类型
	ExecutionState     int     `json:"executionState"`     // 执行状态：1=成功，0=失败
	RecalledCount      int     `json:"recalledCount"`      // 召回数量（原始结果数量）
	AcceptedCount      int     `json:"acceptedCount"`      // 接受数量（过滤后数量）
	FinalSelectedCount int     `json:"finalSelectedCount"` // 最终选中数量
	AvgScore           float64 `json:"avgScore"`           // 平均分数
	MaxScore           float64 `json:"maxScore"`           // 最大分数
	MinScore           float64 `json:"minScore"`           // 最小分数
}

// SetScores 设置分数统计
func (v *ChannelExecutionView) SetScores(docs []*klvo.Document) {
	if len(docs) == 0 {
		return
	}

	var total float64
	v.MinScore = docs[0].Score
	v.MaxScore = docs[0].Score
	for _, doc := range docs {
		total += doc.Score
		v.MinScore = min(v.MinScore, doc.Score)
		v.MaxScore = max(v.MaxScore, doc.Score)
	}
	v.AvgScore = total / float64(len(docs))
}

// RetrievalResultView 检索结果观测视图
// 记录每个检索文档的详细信息，包括各阶段分数、是否通过闸门、是否被选中等
type RetrievalResultView struct {
	ExchangeId       int64   `json:"exchangeId"`       // 对话交换ID
	TraceId          string  `json:"traceId"`          // 追踪ID
	SubQuestionIndex int     `json:"subQuestionIndex"` // 子问题索引
	SubQuestion      string  `json:"subQuestion"`      // 子问题文本
	ChannelType      string  `json:"channelType"`      // 渠道类型
	ChannelRank      int     `json:"channelRank"`      // 该文档在该通道中的排名（从1开始）
	OriginalScore    float64 `json:"originalScore"`    // 原始分数（向量相似度或关键词分数）
	RrfScore         float64 `json:"rrfScore"`         // RRF融合分数
	RerankScore      float64 `json:"rerankScore"`      // 重排序分数
	DocumentId       int64   `json:"documentId"`       // 文档ID
	DocumentName     string  `json:"documentName"`     // 文档名称
	ChunkId          int64   `json:"chunkId"`          // 块ID
	ChunkNo          int     `json:"chunkNo"`          // 块序号
	SectionPath      string  `json:"sectionPath"`      // 章节路径
	ChunkTextPreview string  `json:"chunkTextPreview"` // 块文本预览（最多500字符）
	ChunkCharCount   int     `json:"chunkCharCount"`   // 块字符数
	GatePassed       bool    `json:"gatePassed"`       // 是否通过闸门过滤
	Selected         bool    `json:"selected"`         // 是否被选入最终结果
	FinalRank        int     `json:"finalRank"`        // 最终排名（未被选中时为0）
	SelectionReason  string  `json:"selectionReason"`  // 选择原因
}

// SetDocumentInfo 设置文档基本信息
func (v *RetrievalResultView) SetDocumentInfo(doc *klvo.Document) {
	if doc == nil {
		return
	}

	// 从Meta中提取信息
	if doc.Meta != nil {
		v.RrfScore, _ = convertor.ToFloat(doc.Meta[klvo.MetaRRFScore])
		v.RerankScore, _ = convertor.ToFloat(doc.Meta[klvo.MetaRerankScore])
		v.DocumentId, _ = convertor.ToInt(doc.Meta[klvo.MetaDocumentID])
		v.DocumentName = convertor.ToString(doc.Meta[klvo.MetaDocumentName])
		v.ChunkId, _ = convertor.ToInt(doc.Meta[klvo.MetaChunkID])
		toInt, _ := convertor.ToInt(doc.Meta[klvo.MetaChunkNo])
		v.ChunkNo = int(toInt)
		v.SectionPath = convertor.ToString(doc.Meta[klvo.MetaSectionPath])
	}

	// 设置原始分数
	v.OriginalScore = doc.Score

	// 设置文本预览
	v.ChunkCharCount = utf8.RuneCountInString(doc.Content)
	if utf8.RuneCountInString(doc.Content) > 500 {
		v.ChunkTextPreview = string([]rune(doc.Content)[:500]) + "..."
	} else {
		v.ChunkTextPreview = doc.Content
	}
}

// RagRetrievalObservation RAG检索观测数据
// 包含渠道执行和检索结果的完整观测信息
type RagRetrievalObservation struct {
	ChannelExecutions []*ChannelExecutionView `json:"channelExecutions"` // 渠道执行列表
	RetrievalResults  []*RetrievalResultView  `json:"retrievalResults"`  // 检索结果列表
}
