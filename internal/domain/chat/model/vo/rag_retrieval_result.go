package vo

import (
	"fmt"

	list "github.com/duke-git/lancet/v2/datastructure/list"
)

// RetrievalResult RAG 检索结果
type RetrievalResult struct {
	RetrievalQuestion       string
	SubQuestionEvidenceList []*SubQuestionEvidence
	EvidenceText            string
	retrievalNotes          *list.CopyOnWriteList[string]
	usedChannels            *list.CopyOnWriteList[string]
}

func NewRagRetrievalResult(retrievalQuestion string) *RetrievalResult {
	return &RetrievalResult{
		RetrievalQuestion: retrievalQuestion,
		retrievalNotes:    list.NewCopyOnWriteList([]string{}),
		usedChannels:      list.NewCopyOnWriteList([]string{}),
	}
}

// IsEmpty 判断检索上下文是否为空（所有子问题均无证据）
func (r *RetrievalResult) IsEmpty() bool {
	if r == nil || len(r.SubQuestionEvidenceList) == 0 {
		return true
	}
	for _, sq := range r.SubQuestionEvidenceList {
		if len(sq.References) > 0 {
			return false
		}
	}
	return true
}

// FlattenReferences 合并所有子问题的引用
func (r *RetrievalResult) FlattenReferences() []*SearchReference {
	if r == nil || len(r.SubQuestionEvidenceList) == 0 {
		return nil
	}
	var refs []*SearchReference
	for _, sq := range r.SubQuestionEvidenceList {
		refs = append(refs, sq.References...)
	}
	return refs
}

// AddRetrievalNotef 添加检索笔记
func (r *RetrievalResult) AddRetrievalNotef(format string, args ...any) {
	note := fmt.Sprintf(format, args...)
	r.retrievalNotes.Add(note)
}

// AddUsedChannel 添加已使用的渠道
func (r *RetrievalResult) AddUsedChannel(channel string) {
	if !r.usedChannels.Contain(channel) {
		r.usedChannels.Add(channel)
	}
}

// UsedChannels 获取已使用的渠道
func (r *RetrievalResult) UsedChannels() []string {
	size := r.usedChannels.Size()
	if size == 0 {
		return nil
	}
	return r.usedChannels.SubList(0, size)
}

// RetrievalNotes 获取检索笔记
func (r *RetrievalResult) RetrievalNotes() []string {
	size := r.retrievalNotes.Size()
	if size == 0 {
		return nil
	}
	return r.retrievalNotes.SubList(0, size)
}

// ToSnapshot 构建检索阶段快照
func (r *RetrievalResult) ToSnapshot(plan *RetrievalPlan) map[string]any {
	references := r.FlattenReferences()
	return map[string]any{
		"retrievalQuestion": r.RetrievalQuestion,
		"retrievalPlan":     plan,
		"usedChannels":      r.UsedChannels(),
		"retrievalNotes":    r.RetrievalNotes(),
		"referenceCount":    len(references),
		"subQuestionCount":  len(r.SubQuestionEvidenceList),
		"subQuestions":      r.BuildSubQuestionSnapshots(),
		"references":        ToRefSnapshotList(references),
	}
}

// ValidateEvidenceBudgetScope 验证证据预算范围：子问题索引必须为正数且唯一
func (r *RetrievalResult) ValidateEvidenceBudgetScope() error {
	indexSet := make(map[int]struct{}, len(r.SubQuestionEvidenceList))
	for _, sq := range r.SubQuestionEvidenceList {
		if sq.SubQuestionIndex <= 0 {
			return fmt.Errorf("证据预算子问题索引必须为正数: index=%d", sq.SubQuestionIndex)
		}
		if _, exists := indexSet[sq.SubQuestionIndex]; exists {
			return fmt.Errorf("证据预算子问题索引必须唯一: 重复 index=%d", sq.SubQuestionIndex)
		}
		indexSet[sq.SubQuestionIndex] = struct{}{}
	}
	return nil
}

// BuildSubQuestionBudgetDetails 构建子问题预算详情列表
func (r *RetrievalResult) BuildSubQuestionBudgetDetails(promptResult *RagPromptAssemblyResult) []map[string]any {
	details := make([]map[string]any, len(r.SubQuestionEvidenceList))
	for i, sq := range r.SubQuestionEvidenceList {
		details[i] = map[string]any{
			"subQuestionIndex": sq.SubQuestionIndex,
			"question":         sq.SubQuestion,
			"referenceCount":   len(sq.References),
		}
	}
	_ = promptResult // 保留扩展位置
	return details
}

// BuildSubQuestionSnapshots 构建子问题追踪快照列表
func (r *RetrievalResult) BuildSubQuestionSnapshots() []map[string]any {
	subQuestions := make([]map[string]any, len(r.SubQuestionEvidenceList))
	for i, sq := range r.SubQuestionEvidenceList {
		subQuestions[i] = sq.BuildSubQuestionSnapshot()
	}
	return subQuestions
}

// RetrievalContexts 获取检索上下文
func (r *RetrievalResult) RetrievalContexts() []string {
	if r == nil {
		return nil
	}
	var contexts []string
	for _, evidence := range r.SubQuestionEvidenceList {
		for _, ref := range evidence.References {
			contexts = append(contexts, ref.Snippet)
		}
	}
	return contexts
}
