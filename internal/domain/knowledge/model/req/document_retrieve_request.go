package req

// DocumentRetrieveRequest 文档检索请求
type DocumentRetrieveRequest struct {
	Query             string   // 查询语句
	DocumentIdList    []int64  // 文档ID列表（可选）
	KnowledgeScopeId  string   // 知识域ID（可选）
	TopK              int      // 返回数量
	ScoreThreshold    float64  // 分数阈值
	BusinessCategory  string   // 业务类别（可选）
	DocumentTags      []string // 文档标签（可选）
}