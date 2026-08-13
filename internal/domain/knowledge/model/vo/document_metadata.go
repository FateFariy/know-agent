package vo

import "github.com/swiftbit/know-agent/common/utils"

type DocumentMetadata struct {
	DocumentId        int64  `gorm:"column:id"`                  // 文档ID
	DocumentName      string `gorm:"column:document_name"`       // 文档名称
	KnowledgeBaseId   int64  `gorm:"column:knowledge_base_id"`   // 知识库ID
	KnowledgeBaseName string `gorm:"column:knowledge_base_name"` // 知识库名称
	LastIndexTaskId   int64  `gorm:"column:last_index_task_id"`  // 上一次索引任务ID
}

// BuildRouteText 拼接文档元数据 + 画像作为路由文本
func (doc *DocumentMetadata) BuildRouteText(profile *DocumentProfile) string {
	if doc == nil {
		return ""
	}
	if profile == nil {
		return utils.JoinNonBlank(" ", doc.DocumentName)
	}
	return utils.JoinNonBlank(" ", doc.DocumentName, profile.DocumentSummary, profile.CoreTopics, profile.ExampleQuestions, profile.DocumentType)
}
