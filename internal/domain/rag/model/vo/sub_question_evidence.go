package vo

import "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"

// SubQuestionEvidence 子问题检索证据
type SubQuestionEvidence struct {
	SubQuestionIndex       int                        `json:"subQuestionIndex"`
	SubQuestion            string                     `json:"subQuestion"`
	References             []*vo.SearchReference      `json:"references"`
	Documents              []*DocumentCandidate       `json:"documents"`
	ChannelTraces          []*SubQuestionChannelTrace `json:"channelTraces"`
	FusedCandidateCount    *int                       `json:"fusedCandidateCount"`
	ParentCandidateCount   *int                       `json:"parentCandidateCount"`
	RerankedCandidateCount *int                       `json:"rerankedCandidateCount"`
}

// SubQuestionChannelTrace 子问题渠道执行追踪
type SubQuestionChannelTrace struct {
	ChannelName   string `json:"channelName"`
	RecalledCount int    `json:"recalledCount"`
	AcceptedCount int    `json:"acceptedCount"`
}

// DocumentCandidate 文档候选结果（对应 Java org.springframework.ai.document.Document）
type DocumentCandidate struct {
	ID      string                 `json:"id"`
	Content string                 `json:"content"`
	Meta    map[string]interface{} `json:"meta"`
	Score   float64                `json:"score"`
	Channel string                 `json:"channel"`
}
