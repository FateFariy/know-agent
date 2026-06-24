package model

import "github.com/swiftbit/know-agent/common"

type DocumentProfile struct {
	common.Model
	documentId           int64  `gorm:"column:document_id"`            // 文档ID
	profileVersion       int    `gorm:"column:profile_version"`        // 文档配置版本
	documentSummary      string `gorm:"column:document_summary"`       // 文档摘要
	documentType         string `gorm:"column:document_type"`          // 文档类型
	coreTopics           string `gorm:"column:core_topics"`            // 核心主题
	exampleQuestions     string `gorm:"column:example_questions"`      // 示例问题
	graphFriendly        int    `gorm:"column:graph_friendly"`         // 是否支持图友好
	supportsGraphOutline int    `gorm:"column:supports_graph_outline"` // 是否支持图大纲
	supportsItemLookup   int    `gorm:"column:supports_item_lookup"`   // 是否支持项目查找
	supportsGraphAssist  int    `gorm:"column:supports_graph_assist"`  // 是否支持图辅助
	profileSource        string `gorm:"column:profile_source"`         // 文档配置来源
	profileStatus        int    `gorm:"column:profile_status"`         // 文档配置状态
	errorMsg             string `gorm:"column:error_msg"`              // 错误信息
}
