package enum

// QueryType 查询类型枚举
type QueryType = string

const (
	QueryTypeDocumentQA          QueryType = "DOCUMENT_QA"          // 文档问答: 基于知识库内容的问答
	QueryTypeFollowUp            QueryType = "FOLLOW_UP"            // 追问: 基于上下文的连续对话
	QueryTypeStructureNavigation QueryType = "STRUCTURE_NAVIGATION" // 结构导航: 文档结构/目录导航
	QueryTypeTableQuery          QueryType = "TABLE_QUERY"          // 表格查询: 结构化表格数据检索
	QueryTypeGraphRelation       QueryType = "GRAPH_RELATION"       // 图谱关系: 知识库图谱关系查询
	QueryTypeGlobalSummary       QueryType = "GLOBAL_SUMMARY"       // 全局摘要: 跨文档/全局内容总结
	QueryTypeOpenChat            QueryType = "OPEN_CHAT"            // 开放闲聊: 非知识库依赖的通用对话
	QueryTypeCapabilityQuery     QueryType = "CAPABILITY_QUERY"     // 能力查询: 系统/模型能力探测
)
