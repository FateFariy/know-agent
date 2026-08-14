package vo

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
	ParentBlockId   int64  // 父块ID
	ChunkId         int64  // 分块ID
	Source          string // 来源标识
}
