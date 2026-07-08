package entity

// KnowledgeRouteTrace 知识路由跟踪记录
type KnowledgeRouteTrace struct {
	ID                  int64   `gorm:"column:id"`                    // 主键
	ConversationId      string  `gorm:"column:conversation_id"`       // 会话ID
	ExchangeId          int64   `gorm:"column:exchange_id"`           // 交互ID
	Question            string  `gorm:"column:question"`              // 问题
	RewriteQuestion     string  `gorm:"column:rewrite_question"`      // 重写问题
	Mode                string  `gorm:"column:mode"`                  // 模式
	TopScopesJson       string  `gorm:"column:top_scopes_json"`       // 顶级范围JSON
	TopTopicsJson       string  `gorm:"column:top_topics_json"`       // 顶级话题JSON
	TopDocumentsJson    string  `gorm:"column:top_documents_json"`    // 顶级文档JSON
	SelectedDocumentId  int64   `gorm:"column:selected_document_id"`  // 选中的文档ID
	HitSelectedDocument int     `gorm:"column:hit_selected_document"` // 命中选中的文档
	Confidence          float64 `gorm:"column:confidence"`            // 置信度
	RouteStatus         int     `gorm:"column:route_status"`          // 路由状态
	ErrorMsg            string  `gorm:"column:error_msg"`             // 错误信息
}
