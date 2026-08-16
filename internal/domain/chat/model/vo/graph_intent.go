package vo

// GraphIntent 图谱检索意图
type GraphIntent struct {
	Requested      bool     `json:"requested"`      // 是否请求
	Entities       []string `json:"entities"`       // 实体列表
	TargetEntities []string `json:"targetEntities"` // 目标实体列表
	MaxHops        int      `json:"maxHops"`        // 最大跳数
	Source         string   `json:"source"`         // 来源
}

// BuildGraphIntent 构建图谱检索意图
func BuildGraphIntent(intentResult *IntentRecognitionResult, runtime *RagRuntimeOptions) *GraphIntent {
	maxHops := 2
	if runtime != nil {
		maxHops = runtime.GraphRagMaxHops
	}
	return &GraphIntent{
		Requested:      channelRequested(intentResult, "GRAPH_RELATION", "GRAPH_RAG"),
		Entities:       normalizeStringsLimit(intentResult.Entities, 8),
		TargetEntities: normalizeStringsLimit(intentResult.TargetEntities, 8),
		MaxHops:        maxHops,
		Source:         intentSource(intentResult),
	}
}
