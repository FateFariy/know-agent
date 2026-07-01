package vo

// DocumentKnowledgeMetadataKeys 文档知识元数据键常量
const (
	MetaSourceType          = "sourceType"
	MetaChannel             = "channel"
	MetaScore               = "score"
	MetaRRFScore            = "rrfScore"
	MetaRerankScore         = "rerankScore"
	MetaDocumentID          = "documentId"
	MetaDocumentName        = "documentName"
	MetaTaskID              = "taskId"
	MetaParentBlockID       = "parentBlockId"
	MetaParentBlockNo       = "parentBlockNo"
	MetaChunkID             = "chunkId"
	MetaChunkNo             = "chunkNo"
	MetaSectionPath         = "sectionPath"
	MetaStructureNodeID     = "structureNodeId"
	MetaStructureNodeType   = "structureNodeType"
	MetaCanonicalPath       = "canonicalPath"
	MetaItemIndex           = "itemIndex"
	MetaKnowledgeScopeCode  = "knowledgeScopeCode"
	MetaKnowledgeScopeName  = "knowledgeScopeName"
	MetaBusinessCategory    = "businessCategory"
	MetaDocumentTags        = "documentTags"
	MetaTitle               = "title"
	MetaURL                 = "url"
	MetaToolName            = "toolName"
	MetaOriginalSnippet     = "originalSnippet"
	MetaRerankModel         = "rerankModel"
	MetaRerankQuery         = "rerankQuery"
	MetaRerankDurationMs    = "rerankDurationMs"
	MetaRerankOriginalIndex = "rerankOriginalIndex"
)

// Document 文档
type Document struct {
	ID      string         `json:"id"`      // 文档ID
	Content string         `json:"content"` // 文档内容
	Score   float64        `json:"score"`   // 相似度分数
	Meta    map[string]any `json:"meta"`    // 元数据
}
