package vo

// SearchDocument 检索文档
type SearchDocument struct {
	ID      string                 `json:"id"`      // 文档ID
	Content string                 `json:"content"` // 文档内容
	Meta    map[string]interface{} `json:"meta"`    // 元数据
	Score   float64                `json:"score"`   // 相似度分数
}