package vo

// RaptorIntent RAPTOR检索意图
type RaptorIntent struct {
	Requested        bool `json:"requested"`        // 是否请求
	SummaryRequested bool `json:"summaryRequested"` // 是否请求摘要
	SourceChunkTopK  int  `json:"sourceChunkTopK"`  // 源分块TopK
}

// Clone 深拷贝RAPTOR意图
func (r *RaptorIntent) Clone() *RaptorIntent {
	if r == nil {
		return nil
	}
	s := *r
	return &s
}
