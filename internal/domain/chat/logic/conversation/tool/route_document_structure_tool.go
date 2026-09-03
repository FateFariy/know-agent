package tool

import "github.com/swiftbit/know-agent/internal/domain/chat/model/enum"

type RouteStructureInput struct {
	Question        string                              `json:"query" jsonschema_description:"改写问题"`                // 改写问题
	QueryType       string                              `json:"queryType" jsonschema_description:"查询类型"`            // 查询类型
	Channels        []enum.RetrievalChannel             `json:"channels" jsonschema_description:"检索通道"`             // 检索通道
	Operations      []enum.StructureNavigationOperation `json:"operations"`                                         // 结构导航操作
	SectionAnchors  []string                            `json:"sectionAnchors" jsonschema_description:"显式章节锚点"`     // 显式章节锚点
	HasStructureNav bool                                `json:"hasStructureNav" jsonschema_description:"是否高置信结构导航"` // 是否高置信结构导航
}

type RouteStructureOutput struct {
	Action        string `json:"action"`                  // fresh_topic / item_reference / structure_navigation
	RetrievalMode string `json:"retrievalMode"`           // 恒为 RETRIEVAL
	TargetSection string `json:"targetSection,omitempty"` // 结构锚点提示
	ItemIndex     *int   `json:"itemIndex,omitempty"`
}
