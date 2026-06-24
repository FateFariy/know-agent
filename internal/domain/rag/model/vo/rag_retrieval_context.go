package vo

import (
	"fmt"

	list "github.com/duke-git/lancet/v2/datastructure/list"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// RagRetrievalContext RAG 检索上下文
type RagRetrievalContext struct {
	RetrievalQuestion       string                        `json:"retrievalQuestion"`
	SubQuestionEvidenceList []*SubQuestionEvidence        `json:"subQuestionEvidenceList"`
	RetrievalNotes          *list.CopyOnWriteList[string] `json:"retrievalNotes"`
	UsedChannels            *list.CopyOnWriteList[string] `json:"usedChannels"`
	FlattenedReferences     []*vo.SearchReference         `json:"flattenedReferences"`
}

func NewRagRetrievalContext(retrievalQuestion string) *RagRetrievalContext {
	return &RagRetrievalContext{
		RetrievalQuestion: retrievalQuestion,
		RetrievalNotes:    list.NewCopyOnWriteList([]string{}),
		UsedChannels:      list.NewCopyOnWriteList([]string{}),
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
func (c *RagRetrievalContext) FlattenReferences() []*vo.SearchReference {
	if c.FlattenedReferences != nil {
		return c.FlattenedReferences
	}
	var refs []*vo.SearchReference
	for _, sq := range c.SubQuestionEvidenceList {
		refs = append(refs, sq.References...)
	}
	c.FlattenedReferences = refs
	return refs
}

// AddRetrievalNotef 添加检索笔记
func (c *RagRetrievalContext) AddRetrievalNotef(format string, args ...any) {
	note := fmt.Sprintf(format, args...)
	c.RetrievalNotes.Add(note)
}

// AddUsedChannel 添加已使用的渠道
func (c *RagRetrievalContext) AddUsedChannel(channel string) {
	if c.UsedChannels.IndexOf(channel) == -1 {
		c.UsedChannels.Add(channel)
	}
}
