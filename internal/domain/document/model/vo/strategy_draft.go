package vo

// DocumentStrategyPlanDraft 策略方案草稿
type DocumentStrategyPlanDraft struct {
	ParentSteps      []*DocumentStrategyStepDraft // 父块步骤列表
	ChildSteps       []*DocumentStrategyStepDraft // 子块步骤列表
	StrategySnapshot string                       // 策略快照
	RecommendReason  string                       // 推荐理由
}

// DocumentStrategyStepDraft 策略步骤草稿
type DocumentStrategyStepDraft struct {
	PipelineType    string // 流水线类型
	StrategyType    int    // 策略类型
	StrategyRole    int    // 策略角色
	SourceType      int    // 来源类型
	RecommendReason string // 推荐理由
}

// ChunkCandidate 块候选
type ChunkCandidate struct {
	SectionPath       string // 章节路径
	StructureNodeId   int64  // 结构体节点ID
	StructureNodeType int    // 结构体节点类型
	CanonicalPath     string // 标准路径
	ItemIndex         int    // 项目索引
	Text              string // 文本内容
	SourceType        int    // 来源类型
}

// ParentBlockCandidate 父块候选
type ParentBlockCandidate struct {
	SectionPath       string            // 章节路径
	StructureNodeId   int64             // 结构体节点ID
	StructureNodeType int               // 结构体节点类型
	CanonicalPath     string            // 标准路径
	ItemIndex         int               // 项目索引
	Text              string            // 文本内容
	SourceType        int               // 来源类型
	ChildChunks       []*ChunkCandidate // 子块列表
}
