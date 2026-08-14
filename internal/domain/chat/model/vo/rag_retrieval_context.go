package vo

import (
	"fmt"

	list "github.com/duke-git/lancet/v2/datastructure/list"
)

// RagRetrievalContext RAG 检索上下文
type RagRetrievalContext struct {
	RetrievalQuestion       string
	SubQuestionEvidenceList []*SubQuestionEvidence
	retrievalNotes          *list.CopyOnWriteList[string]
	usedChannels            *list.CopyOnWriteList[string]
}

func NewRagRetrievalContext(retrievalQuestion string) *RagRetrievalContext {
	return &RagRetrievalContext{
		RetrievalQuestion: retrievalQuestion,
		retrievalNotes:    list.NewCopyOnWriteList([]string{}),
		usedChannels:      list.NewCopyOnWriteList([]string{}),
	}
}

// IsEmpty 判断检索上下文是否为空（所有子问题均无证据）
func (c *RagRetrievalContext) IsEmpty() bool {
	if len(c.SubQuestionEvidenceList) == 0 {
		return true
	}
	for _, sq := range c.SubQuestionEvidenceList {
		if len(sq.References) > 0 {
			return false
		}
	}
	return true
}

// FlattenReferences 合并所有子问题的引用
func (c *RagRetrievalContext) FlattenReferences() []*SearchReference {
	if c == nil || len(c.SubQuestionEvidenceList) == 0 {
		return nil
	}
	var refs []*SearchReference
	for _, sq := range c.SubQuestionEvidenceList {
		refs = append(refs, sq.References...)
	}
	return refs
}

// AddRetrievalNotef 添加检索笔记
func (c *RagRetrievalContext) AddRetrievalNotef(format string, args ...any) {
	note := fmt.Sprintf(format, args...)
	c.retrievalNotes.Add(note)
}

// AddUsedChannel 添加已使用的渠道
func (c *RagRetrievalContext) AddUsedChannel(channel string) {
	if !c.usedChannels.Contain(channel) {
		c.usedChannels.Add(channel)
	}
}

// UsedChannels 获取已使用的渠道
func (c *RagRetrievalContext) UsedChannels() []string {
	size := c.usedChannels.Size()
	if size == 0 {
		return nil
	}
	return c.usedChannels.SubList(0, size)
}

// RetrievalNotes 获取检索笔记
func (c *RagRetrievalContext) RetrievalNotes() []string {
	size := c.retrievalNotes.Size()
	if size == 0 {
		return nil
	}
	return c.retrievalNotes.SubList(0, size)
}

// ToSnapshot 构建检索阶段快照
func (c *RagRetrievalContext) ToSnapshot(plan *ConversationExecutionPlan) map[string]any {
	references := c.FlattenReferences()
	return map[string]any{
		"retrievalQuestion": c.RetrievalQuestion,
		"retrievalPlan":     plan.RetrievalPlan,
		"usedChannels":      c.UsedChannels(),
		"retrievalNotes":    c.RetrievalNotes(),
		"referenceCount":    len(references),
		"subQuestionCount":  len(c.SubQuestionEvidenceList),
		"subQuestions":      c.BuildSubQuestionSnapshots(),
		"references":        ToRefSnapshotList(references),
	}
}

// ValidateEvidenceBudgetScope 验证证据预算范围：子问题索引必须为正数且唯一
func (c *RagRetrievalContext) ValidateEvidenceBudgetScope() error {
	indexSet := make(map[int]struct{}, len(c.SubQuestionEvidenceList))
	for _, sq := range c.SubQuestionEvidenceList {
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
func (c *RagRetrievalContext) BuildSubQuestionBudgetDetails(promptResult *RagPromptAssemblyResult) []map[string]any {
	details := make([]map[string]any, len(c.SubQuestionEvidenceList))
	for i, sq := range c.SubQuestionEvidenceList {
		details[i] = map[string]any{
			"subQuestionIndex": sq.SubQuestionIndex,
			"question":         sq.SubQuestion,
			"referenceCount":   len(sq.References),
			//"documentCount":    len(sq.Documents),
		}
	}
	_ = promptResult // 保留扩展位置
	return details
}

// BuildSubQuestionSnapshots 构建子问题追踪快照列表
func (c *RagRetrievalContext) BuildSubQuestionSnapshots() []map[string]any {
	subQuestions := make([]map[string]any, len(c.SubQuestionEvidenceList))
	for i, sq := range c.SubQuestionEvidenceList {
		subQuestions[i] = sq.BuildSubQuestionSnapshot()
	}
	return subQuestions
}
