package enum

import (
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

// QueryType 查询类型枚举
type QueryType = string

const (
	QueryTypeDocumentQA          QueryType = "document_qa"          // 文档问答: 基于知识库内容的问答
	QueryTypeFollowUp            QueryType = "follow_up"            // 追问: 基于上下文的连续对话
	QueryTypeStructureNavigation QueryType = "structure_navigation" // 结构导航: 文档结构/目录导航
	QueryTypeTableQuery          QueryType = "table_query"          // 表格查询: 结构化表格数据检索
	QueryTypeGraphRelation       QueryType = "graph_relation"       // 图谱关系: 知识库图谱关系查询
	QueryTypeGlobalSummary       QueryType = "global_summary"       // 全局摘要: 跨文档/全局内容总结
	QueryTypeCapabilityQuery     QueryType = "capability_query"     // 能力查询: 系统/模型能力探测
)

func ParseQueryType(raw string) QueryType {
	normalized := strings.ToUpper(utils.Trim(raw))
	if normalized == "" {
		return QueryTypeDocumentQA
	}
	// 检查是否属于已知类型
	switch normalized {
	case QueryTypeDocumentQA, QueryTypeStructureNavigation,
		QueryTypeTableQuery, QueryTypeGraphRelation,
		QueryTypeGlobalSummary,
		QueryTypeFollowUp, QueryTypeCapabilityQuery:
		return normalized
	default:
		return QueryTypeDocumentQA
	}
}
