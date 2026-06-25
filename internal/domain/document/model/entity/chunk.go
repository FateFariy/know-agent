package entity

// DocumentProfile 文档属性实体
type DocumentProfile struct {
	ID                   int64  `gorm:"column:id"`                     // 主键ID
	DocumentId           int64  `gorm:"column:document_id"`            // 文档ID
	ProfileVersion       int    `gorm:"column:profile_version"`        // 属性版本
	DocumentSummary      string `gorm:"column:document_summary"`       // 文档摘要
	DocumentType         string `gorm:"column:document_type"`          // 文档类型
	CoreTopics           string `gorm:"column:core_topics"`            // 核心话题
	ExampleQuestions     string `gorm:"column:example_questions"`      // 示例问题
	GraphFriendly        int    `gorm:"column:graph_friendly"`         // 图谱友好度
	SupportsGraphOutline int    `gorm:"column:supports_graph_outline"` // 支持图谱大纲
	SupportsItemLookup   int    `gorm:"column:supports_item_lookup"`   // 支持条目检索
	SupportsGraphAssist  int    `gorm:"column:supports_graph_assist"`  // 支持图谱辅助
	ProfileSource        string `gorm:"column:profile_source"`         // 属性来源
	ProfileStatus        int    `gorm:"column:profile_status"`         // 属性状态
	ErrorMsg             string `gorm:"column:error_msg"`              // 错误信息
}

// TopicDocumentRelation 话题文档关联实体
type TopicDocumentRelation struct {
	ID             int64  `gorm:"column:id"`              // 主键ID
	TopicCode      string `gorm:"column:topic_code"`      // 话题编码
	DocumentId     int64  `gorm:"column:document_id"`     // 文档ID
	RelationScore  string `gorm:"column:relation_score"`  // 关联分数
	RelationSource string `gorm:"column:relation_source"` // 关联来源
	Reason         string `gorm:"column:reason"`          // 关联原因
}
