package vo

import (
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// SubQuestionEvidence 子问题检索证据
type SubQuestionEvidence struct {
	SubQuestionIndex       int                        `json:"subQuestionIndex"`       // 子问题索引
	SubQuestion            string                     `json:"subQuestion"`            // 子问题
	References             []*SearchReference         `json:"references"`             // 检索参考
	Documents              []*klvo.Document           `json:"documents"`              // 文档
	ChannelTraces          []*SubQuestionChannelTrace `json:"channelTraces"`          // 渠道追踪
	FusedCandidateCount    int                        `json:"fusedCandidateCount"`    // 混合候选数量
	ParentCandidateCount   int                        `json:"parentCandidateCount"`   // 父候选数量
	RerankedCandidateCount int                        `json:"rerankedCandidateCount"` // 重排序候选数量
}

// SubQuestionChannelTrace 子问题渠道执行追踪
type SubQuestionChannelTrace struct {
	ChannelName   string `json:"channelName"`   // 渠道名称
	RecalledCount int    `json:"recalledCount"` // 召回数量
	AcceptedCount int    `json:"acceptedCount"` // 接受数量
}
