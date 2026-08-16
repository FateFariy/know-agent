package vo

import "github.com/swiftbit/know-agent/internal/domain/chat/model/enum"

// AssemblyInput 检索计划组装输入
// 单一入口对象，将所有构建 RetrievalPlan 所需的输入集中封装
type AssemblyInput struct {
	OriginalQuestion      string                      // 原始用户问题
	RewrittenQuestion     string                      // 重写后的问题
	RewriteSubQuestions   []string                    // 重写生成的子问题列表
	KnowledgeBaseMode     string                      // 知识库查询模式
	KnowledgeBaseIds      []int64                     // 知识库ID列表
	AllowedDocumentIds    []int64                     // 允许的文档ID列表
	DocumentScope         []int64                     // 文档范围ID列表
	TaskScope             []int64                     // 任务范围ID列表
	IntentResult          *IntentRecognitionResult    // 意图识别结果
	NavigationDecision    *DocumentNavigationDecision // 文档导航决策
	StructureNavigation   *StructureNavigationIntent  // 结构导航意图
	ChatMode              enum.ChatQueryMode          // 对话查询模式
	RuntimeOptions        *RagRuntimeOptions          // RAG运行时选项
	ScopedEvidenceAnchors []*EvidenceAnchor           // 作用域内的证据锚点
}

// KnowledgeRoutePlan 知识路由计划
type KnowledgeRoutePlan struct {
	Enabled          bool                  // 是否启用路由
	ChannelName      string                // 路由通道名称
	Confidence       float64               // 路由置信度
	Source           string                // 路由来源
	ThresholdApplied float64               // 应用的阈值
	AboveThreshold   bool                  // 是否超过阈值
	Candidates       []*RetrievalCandidate // 路由候选列表
}
