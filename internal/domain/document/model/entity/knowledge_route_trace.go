package entity

import (
	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/common/utils"
)

// KnowledgeRouteTrace 知识路由追踪实体
type KnowledgeRouteTrace struct {
	ID                  int64   `gorm:"column:id;primaryKey"`         // 主键ID
	CreateTime          int64   `gorm:"column:create_time"`           // 创建时间
	UpdateTime          int64   `gorm:"column:edit_time"`             // 更新时间
	Deleted             int     `gorm:"column:deleted"`               // 逻辑删除标记
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

func (m *KnowledgeRouteTrace) BeforeCreate(tx *gorm.DB) error {
	if m.ID == 0 {
		m.ID = utils.GetSnowflakeNextID()
	}
	m.Deleted = 1
	return nil
}
