package enum

type KnowledgeBaseSelectionMode string

const (
	KbSelectionModeNone     KnowledgeBaseSelectionMode = "none"     // 不使用知识库检索
	KbSelectionModeAll      KnowledgeBaseSelectionMode = "all"      // 使用全部启用知识库
	KbSelectionModeSelected KnowledgeBaseSelectionMode = "selected" // 使用显式选择知识库
)
