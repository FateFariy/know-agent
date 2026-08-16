package vo

// StructureNavigationResult 结构导航结果
type StructureNavigationResult struct {
	DocumentId      int64    `json:"documentId"`      // 文档ID
	ParseTaskId     int64    `json:"parseTaskId"`     // 解析任务ID
	AnchorNodeId    int64    `json:"anchorNodeId"`    // 锚点节点ID
	Current         string   `json:"current"`         // 当前节点
	Parent          string   `json:"parent"`          // 父节点
	PreviousSibling string   `json:"previousSibling"` // 前一个兄弟节点
	NextSibling     string   `json:"nextSibling"`     // 后一个兄弟节点
	DirectChildren  []string `json:"directChildren"`  // 直接子节点列表
	Deterministic   bool     `json:"deterministic"`   // 是否确定性
	MissReason      string   `json:"missReason"`      // 未命中原因
}
