package vo

import (
	"fmt"

	"github.com/swiftbit/know-agent/common/utils"
)

// DocumentKnowledgeMetadataKeys 文档知识元数据键常量
const (
	MetaSourceType          = "source_type"
	MetaChannel             = "channel"
	MetaScore               = "score"
	MetaRRFScore            = "rrfScore"
	MetaRerankScore         = "rerankScore"
	MetaDocumentId          = "document_id"
	MetaDocumentName        = "documentName"
	MetaTaskId              = "task_id"
	MetaPlanId              = "plan_id"
	MetaParentBlockId       = "parent_block_id"
	MetaParentBlockNo       = "parent_block_no"
	MetaChunkId             = "chunk_id"
	MetaChunkNo             = "chunk_no"
	MetaSectionPath         = "section_path"
	MetaStructureNodeId     = "structure_node_id"
	MetaStructureNodeType   = "structure_node_type"
	MetaCanonicalPath       = "canonical_path"
	MetaItemIndex           = "item_index"
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

	// ========== 从 DocumentMetadata 补充 ==========
	DocumentName      string `json:"documentName"`      // 文档名称
	KnowledgeBaseId   string `json:"knowledgeBaseId"`   // 知识库ID
	KnowledgeBaseName string `json:"knowledgeBaseName"` // 知识库名称

	// ========== 其他来源（RRF/重排/外部工具等） ==========
	IsElevated          int     `json:"isElevated"`          // 是否提升
	RRFScore            float64 `json:"rrfScore"`            // RRF分数
	RerankScore         float64 `json:"rerankScore"`         // 重排分数
	ParentBlockNo       int     `json:"parentBlockNo"`       // 父块序号
	Title               string  `json:"title"`               // 文档标题
	Url                 string  `json:"url"`                 // URL地址
	ToolName            string  `json:"toolName"`            // 文档工具名称
	RerankModel         string  `json:"rerankModel"`         // 重排模型
	RerankQuery         string  `json:"rerankQuery"`         // 重排查询
	RerankDurationMs    int64   `json:"rerankDurationMs"`    // 重排耗时（毫秒）
	RerankOriginalIndex int     `json:"rerankOriginalIndex"` // 重排原始索引
}

func (d *DocumentChunk) FillKnowledge(knowledge *DocumentMetadata) {
	if knowledge == nil {
		return
	}
	d.DocumentName = knowledge.DocumentName
}

// NeedsMetadataFallback 判断文档是否需要从知识库回填元数据
// 当文档名称、知识范围编码或知识范围名称为空时，认为需要回填
func (d *DocumentChunk) NeedsMetadataFallback() bool {
	if d == nil {
		return false
	}
	return utils.IsBlank(d.DocumentName) ||
		utils.IsBlank(d.KnowledgeBaseId) ||
		utils.IsBlank(d.KnowledgeBaseName)
}

// EnrichFromMetadata 使用知识库回填文档的缺失元数据字段
func (d *DocumentChunk) EnrichFromMetadata(metadata *DocumentMetadata) {
	if d == nil || metadata == nil {
		return
	}
	if utils.IsBlank(d.DocumentName) && utils.IsNotBlank(metadata.DocumentName) {
		d.DocumentName = metadata.DocumentName
	}
	if utils.IsBlank(d.KnowledgeBaseId) && metadata.KnowledgeBaseId != 0 {
		d.KnowledgeBaseId = fmt.Sprintf("%d", metadata.KnowledgeBaseId)
	}
	if utils.IsBlank(d.KnowledgeBaseName) && utils.IsNotBlank(metadata.KnowledgeBaseName) {
		d.KnowledgeBaseName = metadata.KnowledgeBaseName
	}
}

type DocumentChunks []*DocumentChunk

func (d DocumentChunks) TopScore() float64 {
	if len(d) == 0 {
		return 0
	}
	topScore := 0.0
	for _, chunk := range d {
		if chunk != nil && chunk.Score > topScore {
			topScore = chunk.Score
		}
	}
	return topScore
}
