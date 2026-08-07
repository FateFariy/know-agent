package model

import "github.com/swiftbit/know-agent/common"

type SuperAgentDocumentProfile struct {
	common.Model
	DocumentId           int64   `gorm:"column:document_id;type:bigint"`          // 文档ID
	ProfileVersion       int     `gorm:"column:profile_version;type:int"`         // 配置版本
	DocumentSummary      string  `gorm:"column:document_summary;type:text"`       // 文档摘要
	DocumentType         string  `gorm:"column:document_type;type:varchar(64)"`   // 文档类型
	CoreTopics           string  `gorm:"column:core_topics;type:text"`            // 核心主题
	ExampleQuestions     string  `gorm:"column:example_questions;type:text"`      // 示例问题
	GraphFriendly        int     `gorm:"column:graph_friendly;type:int"`          // 图谱友好标记
	SupportsGraphOutline int     `gorm:"column:supports_graph_outline;type:int"`  // 支持图谱大纲
	SupportsItemLookup   int     `gorm:"column:supports_item_lookup;type:int"`    // 支持条目查询
	SupportsGraphAssist  int     `gorm:"column:supports_graph_assist;type:int"`   // 支持图谱辅助
	ProfileSource        string  `gorm:"column:profile_source;type:varchar(255)"` // 配置来源
	ProfileStatus        int     `gorm:"column:profile_status;type:int"`          // 配置状态
	ErrorMsg             *string `gorm:"column:error_msg;type:text"`              // 错误信息
}
