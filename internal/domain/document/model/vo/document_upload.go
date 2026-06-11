package vo

// DocumentUpload 文档上传
type DocumentUpload struct {
	DocumentId     int64  // 文档ID
	TaskId         int64  // 任务ID
	DocumentName   string // 文档名称
	ParseStatus    int    // 解析状态
	StrategyStatus int    // 策略状态
	IndexStatus    int    // 索引状态
}
