package model

import "github.com/swiftbit/know-agent/common"

// KnowledgeDocumentProfile 文档画像
type KnowledgeDocumentProfile struct {
	common.Model
	DocumentId           int64   `gorm:"column:document_id"`            // 文档ID
	ProfileVersion       int     `gorm:"column:profile_version"`        // 画像版本
	DocumentSummary      string  `gorm:"column:document_summary"`       // 文档摘要
	DocumentType         string  `gorm:"column:document_type"`          // 文档类型
	CoreTopics           string  `gorm:"column:core_topics"`            // 核心主题
	ExampleQuestions     string  `gorm:"column:example_questions"`      // 示例问题
	GraphFriendly        int     `gorm:"column:graph_friendly"`         // 是否支持图表示
	SupportsGraphOutline int     `gorm:"column:supports_graph_outline"` // 是否支持图大纲
	SupportsItemLookup   int     `gorm:"column:supports_item_lookup"`   // 是否支持项目查找
	SupportsGraphAssist  int     `gorm:"column:supports_graph_assist"`  // 是否支持图辅助
	ProfileSource        string  `gorm:"column:profile_source"`         // 画像来源
	ProfileStatus        int     `gorm:"column:profile_status"`         // 画像状态
	ErrorMsg             *string `gorm:"column:error_msg"`              // 错误信息
}
