package model

import (
	"github.com/swiftbit/know-agent/common"
)

// KnowledgeRouteTrace 知识路由追踪实体
type KnowledgeRouteTrace struct {
	common.Model
	ConversationId                 string  `gorm:"column:conversation_id;type:varchar(64)"`               // 会话ID
	ExchangeId                     int64   `gorm:"column:exchange_id;type:bigint"`                        // 交互ID
	Question                       string  `gorm:"column:question;type:text"`                             // 问题
	RewriteQuestion                string  `gorm:"column:rewrite_question;type:text"`                     // 改写后的问题
	Mode                           string  `gorm:"column:mode;type:varchar(50)"`                          // 模式
	KnowledgeBaseSelectionMode     string  `gorm:"column:knowledge_base_selection_mode;type:varchar(50)"` // 知识库选择模式
	SelectedKnowledgeBaseIdsJson   string  `gorm:"column:selected_knowledge_base_ids_json;type:text"`     // 已选知识库ID列表JSON
	SelectedKnowledgeBaseNamesJson string  `gorm:"column:selected_knowledge_base_names_json;type:text"`   // 已选知识库名称列表JSON
	AllowedDocumentIdsJson         string  `gorm:"column:allowed_document_ids_json;type:text"`            // 允许的文档ID列表JSON
	TopScopesJson                  string  `gorm:"column:top_scopes_json;type:text"`                      // Top范围JSON
	TopTopicsJson                  string  `gorm:"column:top_topics_json;type:text"`                      // Top主题JSON
	TopDocumentsJson               string  `gorm:"column:top_documents_json;type:text"`                   // Top文档JSON
	SelectedDocumentId             int64   `gorm:"column:selected_document_id;type:bigint"`               // 已选文档ID
	HitSelectedDocument            int     `gorm:"column:hit_selected_document;type:int"`                 // 命中已选文档标记
	Confidence                     float64 `gorm:"column:confidence;type:decimal(10,5)"`                  // 置信度
	RouteStatus                    int     `gorm:"column:route_status;type:int"`                          // 路由状态
	ErrorMsg                       string  `gorm:"column:error_msg;type:text"`                            // 错误信息
}
