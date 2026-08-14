package vo

import "github.com/swiftbit/know-agent/internal/domain/chat/model/enum"

// DocumentNavigationDecision 文档问答路由结果
type DocumentNavigationDecision struct {
	NavigationAction        enum.DocumentNavigationAction `json:"navigationAction"` // 导航动作
	ExecutionMode           enum.ExecutionMode            // 执行模式
	StructureAnchor         *ConversationStructureAnchor  `json:"structureAnchor"`         // 结构锚点
	ItemAnchor              *ConversationItemAnchor       `json:"itemAnchor"`              // 项目锚点
	RetrievalPlan           *RetrievalQuestionPlan        `json:"retrievalPlan"`           // 检索问题计划
	IntentRecognitionResult *IntentRecognitionResult      `json:"intentRecognitionResult"` // 意图识别结果
	SummaryText             string                        `json:"summaryText"`             // 摘要文本
	QueryContextHints       []string                      `json:"queryContextHints"`       // 查询上下文提示
	SoftSectionHints        []string                      `json:"softSectionHints"`        // 软章节提示
	ExecutionModeName       string                        `json:"executionMode"`           // 执行模式
}

// ConversationStructureAnchor 会话结构锚点
type ConversationStructureAnchor struct {
	RootSectionCode   string `json:"rootSectionCode"`   // 根章节代码
	RootSectionTitle  string `json:"rootSectionTitle"`  // 根章节标题
	TargetSectionHint string `json:"targetSectionHint"` // 目标章节提示
	StructureNodeId   int64  `json:"structureNodeId"`   // 结构节点ID
	CanonicalPath     string `json:"canonicalPath"`     // 正规路径
	ScopeMode         string `json:"scopeMode"`         // 作用域模式
}

// ConversationItemAnchor 会话项目锚点
type ConversationItemAnchor struct {
	ItemIndex       int    `json:"itemIndex"`       // 项目索引
	ItemText        string `json:"itemText"`        // 项目文本
	StructureNodeId int64  `json:"structureNodeId"` // 结构节点ID
	CanonicalPath   string `json:"canonicalPath"`   // 正规路径
}
