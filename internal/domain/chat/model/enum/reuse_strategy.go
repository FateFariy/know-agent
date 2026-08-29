package enum

// ReuseStrategy 语义缓存命中后的复用策略（仅控制 Generate 分支）
type ReuseStrategy = int

const (
	ReuseRetrievalOnly      ReuseStrategy = iota // 复用检索结果，重跑 Generate
	ReuseAnswerAndRetrieval                      // 答案+检索结果一起复用，跳过 Generate
)
