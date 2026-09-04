package model

import (
	"github.com/swiftbit/know-agent/common"
)

// KnowledgeRouteTrace 知识路由追踪实体
type KnowledgeRouteTrace struct {
	common.Model
	ConversationId                 string  `gorm:"column:conversation_id;type:varchar(255)"`              // 会话ID
	ExchangeId                     int64   `gorm:"column:exchange_id;type:bigint"`                        // 交换ID
	Question                       string  `gorm:"column:question;type:text"`                             // 原始问题
	RewriteQuestion                string  `gorm:"column:rewrite_question;type:text"`                     // 改写后问题
	Mode                           string  `gorm:"column:mode;type:varchar(50)"`                          // 路由模式
	KnowledgeBaseSelectionMode     string  `gorm:"column:knowledge_base_selection_mode;type:varchar(50)"` // 知识库选择模式
	SelectedKnowledgeBaseIdsJson   string  `gorm:"column:selected_knowledge_base_ids_json;type:text"`     // 选中知识库ID列表JSON
	SelectedKnowledgeBaseNamesJson string  `gorm:"column:selected_knowledge_base_names_json;type:text"`   // 选中知识库名称列表JSON
	AllowedDocumentIdsJson         string  `gorm:"column:allowed_document_ids_json;type:text"`            // 允许文档ID列表JSON
	TopScopesJson                  string  `gorm:"column:top_scopes_json;type:text"`                      // 顶级Scope列表JSON
	TopTopicsJson                  string  `gorm:"column:top_topics_json;type:text"`                      // 顶级Topic列表JSON
	TopDocumentsJson               string  `gorm:"column:top_documents_json;type:text"`                   // 顶级文档列表JSON
	SelectedDocumentId             int64   `gorm:"column:selected_document_id;type:bigint"`               // 选中文档ID
	HitSelectedDocument            *int    `gorm:"column:hit_selected_document;type:tinyint"`             // 是否命中选中文档（1是，0否）
	Confidence                     float64 `gorm:"column:confidence;type:decimal(10,4)"`                  // 置信度
	RouteStatus                    int     `gorm:"column:route_status;type:tinyint"`                      // 路由状态
	Reason                         string  `gorm:"column:reason;type:text"`                               // 错误信息
}
