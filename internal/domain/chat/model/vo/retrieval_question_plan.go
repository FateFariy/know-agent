package vo

import (
	"fmt"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

// RetrievalQuestionPlan 检索问题计划
type RetrievalQuestionPlan struct {
	CurrentQuestion          string                     // 当前问题
	RewrittenQuestion        string                     // 改写后问题
	NormalizedQuery          string                     // 归一化查询
	ExecutionQueries         []*RetrievalExecutionQuery // 执行查询列表
	FollowUp                 bool                       // 是否追问
	HistoryInherited         bool                       // 是否继承历史
	HistoryInheritanceSource string                     // 历史继承来源
	InheritedContextAnchors  []*RetrievalContextAnchor  // 继承的上下文锚点列表
	MainQuestion             string                     // 主问题(已弃用)
	RetrievalQuestion        string                     // 检索问题(已弃用)
	SubQuestions             []string                   // 子问题列表(已弃用)
	RetrievalMode            string                     // 检索模式(已弃用)
	MaxResults               int                        // 最大结果数(已弃用)
	ScoreThreshold           float64                    // 分数阈值(已弃用)
	ExpandToParent           bool                       // 是否扩展到父级(已弃用)
	ExpandToChildren         bool                       // 是否扩展到子级(已弃用)
}

type RetrievalExecutionQuery struct {
	Index           int      // 查询索引
	SourceQuestion  string   // 原始问题
	NormalizedQuery string   // 归一化查询
	ExecutionQuery  string   // 执行查询语句
	ContextHints    []string // 上下文提示列表
}

type RetrievalContextAnchor struct {
	DocumentId      int64  // 文档ID
	SectionPath     string // 段落路径
	StructureNodeId int64  // 结构节点ID
	ParentChunkId   int64  // 父块ID
	ChunkId         int64  // 分块ID
	Source          string // 来源标识
}

func (a *RetrievalContextAnchor) UniqueKey() string {
	if a == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d:%d:%d", a.DocumentId, a.StructureNodeId, a.ParentChunkId, a.ChunkId)
}

// AnchorHint 生成单个锚点的提示字符串
func (a *RetrievalContextAnchor) AnchorHint() string {
	if a == nil {
		return ""
	}
	var builder strings.Builder
	appendHintPart := func(name string, value any) {
		trimmed := utils.Trim(utils.ToString(value))
		if trimmed == "" {
			return
		}
		if builder.Len() > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(name)
		builder.WriteString("=")
		builder.WriteString(trimmed)
	}

	appendHintPart("documentId", a.DocumentId)
	appendHintPart("sectionPath", a.SectionPath)
	appendHintPart("structureNodeId", a.StructureNodeId)
	appendHintPart("parentBlockId", a.ParentChunkId)
	appendHintPart("chunkId", a.ChunkId)
	return utils.Trim(builder.String())
}
