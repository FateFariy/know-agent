package vo

import "github.com/swiftbit/know-agent/common/utils"

// SubQuestionEvidence 子问题检索证据
type SubQuestionEvidence struct {
	SubQuestionIndex       int                        // 子问题索引
	SubQuestion            string                     // 子问题内容
	SourceDocuments        []*DocumentChunk           // 源文档列表
	ContextDocuments       []*DocumentChunk           // 上下文文档列表
	References             []*SearchReference         // 搜索引用列表
	ChannelTraces          []*SubQuestionChannelTrace // 渠道追踪记录
	FusedCandidateCount    int                        // 融合候选数量
	ParentCandidateCount   int                        // 父级候选数量
	RerankedCandidateCount int                        // 重排序候选数量
	//EvidenceSelectionLedger EvidenceSelectionLedger   // 证据选择记录（观测）
	ObservationPersistence *ObservationPersistence // 观察持久化信息（观测）
}

// SubQuestionChannelTrace 子问题渠道执行追踪
type SubQuestionChannelTrace struct {
	ChannelName     string  `json:"channelName"`     // 渠道名称
	RecalledCount   int     `json:"recalledCount"`   // 召回数量
	AcceptedCount   int     `json:"acceptedCount"`   // 接受数量
	RetrievalIntent string  `json:"retrievalIntent"` // 检索意图
	ChannelWeight   float64 `json:"channelWeight"`   // 通道权重
}

// ObservationPersistence 观察持久化信息
type ObservationPersistence struct {
	SchemaVersion string `json:"schemaVersion"` // 模式版本
	//Status                  Status   // 处理状态
	ExpectedCandidateCount  int `json:"expectedCandidateCount"`  // 预期候选数量
	PersistedCandidateCount int `json:"persistedCandidateCount"` // 已持久化候选数量
	//ErrorType               ErrorType // 错误类型
}

func (s *SubQuestionEvidence) GetReferenceMaps() []map[string]any {
	sqRefs := make([]map[string]any, len(s.References))
	for i, ref := range s.References {
		sqRefs[i] = map[string]any{
			"referenceId":  ref.ReferenceId,
			"documentName": utils.BlankToDefault(ref.DocumentName, ref.Title),
			"sectionPath":  ref.SectionPath,
			"channel":      ref.Channel,
		}
	}
	return sqRefs
}

// BuildSubQuestionSnapshot 构建子问题追踪快照
func (s *SubQuestionEvidence) BuildSubQuestionSnapshot() map[string]any {
	return map[string]any{
		"index":                  s.SubQuestionIndex,
		"question":               s.SubQuestion,
		"referenceCount":         len(s.References),
		"sourceDocumentCount":    len(s.SourceDocuments),
		"fusedCandidateCount":    s.FusedCandidateCount,
		"parentCandidateCount":   s.ParentCandidateCount,
		"rerankedCandidateCount": s.RerankedCandidateCount,
		"observationPersistence": s.ObservationPersistence,
		"channelTraces":          s.ChannelTraces,
		"references":             s.GetReferenceMaps(),
	}
}
