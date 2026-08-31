package entity

// KnowledgeRouteTrace 知识路由跟踪记录
type KnowledgeRouteTrace struct {
	ID                             int64   `gorm:"column:id"`                                 // 主键
	ConversationId                 string  `gorm:"column:conversation_id"`                    // 会话ID
	ExchangeId                     int64   `gorm:"column:exchange_id"`                        // 交换ID
	Question                       string  `gorm:"column:question"`                           // 原始问题
	RewriteQuestion                string  `gorm:"column:rewrite_question"`                   // 改写后问题
	Mode                           string  `gorm:"column:mode"`                               // 路由模式
	KnowledgeBaseSelectionMode     string  `gorm:"column:knowledge_base_selection_mode"`      // 知识库选择模式
	SelectedKnowledgeBaseIdsJson   string  `gorm:"column:selected_knowledge_base_ids_json"`   // 选中知识库ID列表JSON
	SelectedKnowledgeBaseNamesJson string  `gorm:"column:selected_knowledge_base_names_json"` // 选中知识库名称列表JSON
	AllowedDocumentIdsJson         string  `gorm:"column:allowed_document_ids_json"`          // 允许文档ID列表JSON
	TopScopesJson                  string  `gorm:"column:top_scopes_json"`                    // 顶级Scope列表JSON
	TopTopicsJson                  string  `gorm:"column:top_topics_json"`                    // 顶级Topic列表JSON
	TopDocumentsJson               string  `gorm:"column:top_documents_json"`                 // 顶级文档列表JSON
	SelectedDocumentId             int64   `gorm:"column:selected_document_id"`               // 选中文档ID
	HitSelectedDocument            *int    `gorm:"column:hit_selected_document"`              // 是否命中选中文档（1是，0否）
	Confidence                     float64 `gorm:"column:confidence"`                         // 置信度
	RouteStatus                    int     `gorm:"column:route_status"`                       // 路由状态
	Reason                         string  `gorm:"column:error_msg"`                          // 选择理由
}
