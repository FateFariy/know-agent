package vo

import vo2 "github.com/swiftbit/know-agent/internal/domain/document/model/vo"

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

// DocumentChunk 文档块
type DocumentChunk struct {
	// ========== 向量检索直接得到 ==========
	ID                string  `json:"id"`                // 块ID
	Score             float64 `json:"score"`             // 相似度分数
	Content           string  `json:"content"`           // 文本内容
	SourceType        string  `json:"sourceType"`        // 文档来源类型
	Channel           string  `json:"channel"`           // 文档来源渠道
	TaskId            int64   `json:"taskId"`            // 任务ID
	ParentBlockId     int64   `json:"parentBlockId"`     // 父块ID
	DocumentId        int64   `json:"documentId"`        // 文档ID
	ChunkNo           int     `json:"chunkNo"`           // 块序号
	SectionPath       string  `json:"sectionPath"`       // 文档章节路径
	StructureNodeId   int64   `json:"structureNodeId"`   // 文档结构节点ID
	StructureNodeType int     `json:"structureNodeType"` // 文档结构节点类型
	CanonicalPath     string  `json:"canonicalPath"`     // 文档规范路径
	ItemIndex         int     `json:"itemIndex"`         // 文档项索引
	OriginalSnippet   string  `json:"originalSnippet"`   // 文档原始片段

	// ========== 从 KnowledgeDocument 补充 ==========
	DocumentName       string `json:"documentName"`       // 文档名称
	KnowledgeScopeCode string `json:"knowledgeScopeCode"` // 文档知识范围编码
	KnowledgeScopeName string `json:"knowledgeScopeName"` // 文档知识范围名称
	BusinessCategory   string `json:"businessCategory"`   // 文档业务分类
	DocumentTags       string `json:"documentTags"`       // 文档标签

	// ========== 其他来源（RRF/重排/外部工具等） ==========
	RRFScore         float64 `json:"rrfScore"`         // RRF分数
	RerankScore      float64 `json:"rerankScore"`      // 重排分数
	ParentBlockNo    int     `json:"parentBlockNo"`    // 父块序号
	Title            string  `json:"title"`            // 文档标题
	Url              string  `json:"url"`              // URL地址
	ToolName         string  `json:"toolName"`         // 文档工具名称
	RerankModel      string  `json:"rerankModel"`      // 重排模型
	RerankQuery      string  `json:"rerankQuery"`      // 重排查询
	RerankDurationMs string  `json:"rerankDurationMs"` // 重排耗时（毫秒）
}

func (d *DocumentChunk) FillKnowledge(knowledge *vo2.KnowledgeDocument) {
	if knowledge == nil {
		return
	}
	d.KnowledgeScopeCode = knowledge.KnowledgeScopeCode
	d.KnowledgeScopeName = knowledge.KnowledgeScopeName
	d.BusinessCategory = knowledge.BusinessCategory
	d.DocumentTags = knowledge.DocumentTags
	d.DocumentName = knowledge.DocumentName
}
