package vo

// RaptorIntent RAPTOR检索意图
type RaptorIntent struct {
	Requested        bool   `json:"requested"`        // 是否请求
	SummaryRequested bool   `json:"summaryRequested"` // 是否请求摘要
	SourceChunkTopK  int    `json:"sourceChunkTopK"`  // 源分块TopK
	Source           string `json:"source"`           // 来源
}

// BuildRaptorIntent 构建RAPTOR检索意图
func BuildRaptorIntent(intentResult *IntentRecognitionResult, runtime *RagRuntimeOptions) *RaptorIntent {
	requested := channelRequested(intentResult, "GLOBAL_SUMMARY", "RAPTOR")
	sourceChunkTopK := 10
	if runtime != nil {
		sourceChunkTopK = runtime.RaptorSourceChunkTopK
	}
	return &RaptorIntent{
		Requested:        requested,
		SummaryRequested: requested,
		SourceChunkTopK:  sourceChunkTopK,
		Source:           intentSource(intentResult),
	}
}
