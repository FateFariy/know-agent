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
	FlattenedReferences     []*SearchReference
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
	if len(c.FlattenedReferences) > 0 {
		return c.FlattenedReferences
	}
	var refs []*SearchReference
	for _, sq := range c.SubQuestionEvidenceList {
		refs = append(refs, sq.References...)
	}
	c.FlattenedReferences = refs
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
