package vo

import (
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

// RagRetrievalObservation RAG检索观测数据
// 包含渠道执行和检索结果的完整观测信息
type RagRetrievalObservation struct {
	ChannelExecutions []*ChannelExecutionView `json:"channelExecutions"` // 渠道执行列表
	RetrievalResults  []*RetrievalResultView  `json:"retrievalResults"`  // 检索结果列表
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

// SetDocumentInfo 设置文档基本信息
func (v *RetrievalResultView) SetDocumentInfo(doc *klvo.Document) {
	if doc == nil {
		return
	}

	// 从Meta中提取信息
	if doc.Meta != nil {
		if val, ok := doc.Meta["rerankScore"]; ok {
			if score, ok := val.(float64); ok {
				v.RerankScore = score
			}
		}
		if val, ok := doc.Meta[klvo.MetaDocumentID]; ok {
			if id, ok := val.(int64); ok {
				v.DocumentId = id
			}
		}
		if val, ok := doc.Meta[klvo.MetaDocumentName]; ok {
			if name, ok := val.(string); ok {
				v.DocumentName = name
			}
		}
		if val, ok := doc.Meta[klvo.MetaChunkID]; ok {
			if id, ok := val.(int64); ok {
				v.ChunkId = id
			}
		}
		if val, ok := doc.Meta[klvo.MetaChunkNo]; ok {
			if no, ok := val.(int); ok {
				v.ChunkNo = no
			}
		}
		if val, ok := doc.Meta[klvo.MetaSectionPath]; ok {
			if path, ok := val.(string); ok {
				v.SectionPath = path
			}
		}
	}

	// 设置原始分数
	v.OriginalScore = doc.Score

	// 设置文本预览
	if doc.Content != "" {
		v.ChunkCharCount = len(doc.Content)
		if len(doc.Content) > 500 {
			v.ChunkTextPreview = doc.Content[:500]
		} else {
			v.ChunkTextPreview = doc.Content
		}
	}
}
