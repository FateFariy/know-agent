package vo

import (
	"fmt"
	"strconv"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

type SearchReference struct {
	ReferenceId       string  `json:"referenceId"`       // 参考ID
	SourceType        string  `json:"sourceType"`        // 来源类型
	Title             string  `json:"title"`             // 标题
	Url               string  `json:"url"`               // URL地址
	Snippet           string  `json:"snippet"`           // 摘要
	DocumentId        int64   `json:"documentId"`        // 文档ID
	DocumentName      string  `json:"documentName"`      // 文档名称
	ChunkId           int64   `json:"chunkId"`           // 块ID
	ParentBlockId     int64   `json:"parentBlockId"`     // 父块ID
	ParentBlockNo     int     `json:"parentBlockNo"`     // 父块序号
	ChunkNo           int     `json:"chunkNo"`           // 块序号
	SectionPath       string  `json:"sectionPath"`       // 节点路径
	StructureNodeId   int64   `json:"structureNodeId"`   // 结构节点ID
	StructureNodeType int     `json:"structureNodeType"` // 结构节点类型
	CanonicalPath     string  `json:"canonicalPath"`     // 规范路径
	ItemIndex         int     `json:"itemIndex"`         // 项索引
	Score             float64 `json:"score"`             // 分数
	SubQuestionIndex  int     `json:"subQuestionIndex"`  // 子问题索引
	SubQuestion       string  `json:"subQuestion"`       // 子问题内容
	Channel           string  `json:"channel"`           // 渠道名称
	ToolName          string  `json:"toolName"`          // 工具名称
	KnowledgeBaseId   string  `json:"knowledgeBaseId"`   // 知识库ID
	KnowledgeBaseName string  `json:"knowledgeBaseName"` // 知识库名称
}

func NewSearchReference(chunk *DocumentChunk, subQuestionIndex, referenceNumber int, subQuestion string) *SearchReference {
	if chunk == nil {
		return &SearchReference{}
	}

	sourceType := utils.BlankToDefault(chunk.SourceType, "DOCUMENT")
	ref := &SearchReference{
		ReferenceId:      strconv.Itoa(referenceNumber),
		SourceType:       sourceType,
		Snippet:          chunk.OriginalSnippet,
		SubQuestionIndex: subQuestionIndex,
		SubQuestion:      subQuestion,
		Score:            chunk.Score,
		Channel:          chunk.Channel,
	}
	if sourceType == "WEB" {
		ref.Title = utils.BlankToDefault(chunk.Title, "网页来源")
		ref.Url = chunk.Url
		ref.ToolName = utils.BlankToDefault(chunk.ToolName, "tavily_search")
		return ref
	}
	ref.Title = utils.BlankToDefault(chunk.Title, "文档片段")
	ref.DocumentId = chunk.DocumentId
	ref.DocumentName = chunk.DocumentName
	ref.ParentBlockId = chunk.ParentBlockId
	ref.ParentBlockNo = chunk.ParentBlockNo
	ref.ChunkId, _ = convertor.ToInt(chunk.ID)
	ref.ChunkNo = chunk.ChunkNo
	ref.SectionPath = chunk.SectionPath
	ref.StructureNodeId = chunk.StructureNodeId
	ref.StructureNodeType = chunk.StructureNodeType
	ref.CanonicalPath = chunk.CanonicalPath
	ref.ItemIndex = chunk.ItemIndex
	ref.KnowledgeBaseId = chunk.KnowledgeBaseId
	ref.KnowledgeBaseName = chunk.KnowledgeBaseName
	return ref
}

// UniqueKey 生成唯一键
func (r *SearchReference) UniqueKey() string {
	if r.ParentBlockId > 0 {
		return fmt.Sprintf("PARENT:%d", r.ParentBlockId)
	}
	if r.ChunkId > 0 {
		return fmt.Sprintf("DOCUMENT:%d", r.ChunkId)
	}
	if r.Url != "" {
		return fmt.Sprintf("WEB:%s", r.Url)
	}
	return fmt.Sprintf("%s:%s:%s", utils.BlankToDefault(r.SourceType, "UNKNOWN"), r.Title, r.Snippet)
}

// ReferenceSummary 生成引用摘要
func (r *SearchReference) ReferenceSummary(suffix string) string {
	title := utils.BlankToDefault(r.DocumentName, r.Title)
	path := utils.BlankToDefault(r.SectionPath, r.Url)
	refId := utils.BlankToDefault(r.ReferenceId, "-")
	if strutil.IsBlank(path) {
		return "[" + refId + "] " + title + " | " + suffix
	}
	return "[" + refId + "] " + title + " | " + path + " | " + suffix
}

// HasUsableAnchor 检查引用是否有可用的锚点信息
func (r *SearchReference) HasUsableAnchor() bool {
	if r == nil {
		return false
	}
	return r.DocumentId != 0 ||
		r.StructureNodeId != 0 ||
		r.ParentBlockId != 0 ||
		r.ChunkId != 0 ||
		r.SectionPath != ""
}

// ToSnapshot 将单个搜索引用转换为快照
func (r *SearchReference) ToSnapshot() map[string]any {
	return map[string]any{
		"referenceId":        r.ReferenceId,
		"sourceType":         r.SourceType,
		"documentId":         r.DocumentId,
		"documentName":       utils.BlankToDefault(r.DocumentName, r.Title),
		"chunkId":            r.ChunkId,
		"chunkNo":            r.ChunkNo,
		"parentBlockId":      r.ParentBlockId,
		"parentBlockNo":      r.ParentBlockNo,
		"sectionPath":        r.SectionPath,
		"channel":            r.Channel,
		"score":              r.Score,
		"subQuestionIndex":   r.SubQuestionIndex,
		"subQuestion":        r.SubQuestion,
		"toolName":           r.ToolName,
		"knowledgeScopeCode": r.KnowledgeBaseId,
		"knowledgeScopeName": r.KnowledgeBaseName,
		"structureNodeId":    r.StructureNodeId,
		"canonicalPath":      r.CanonicalPath,
		"itemIndex":          r.ItemIndex,
		"title":              r.Title,
		"url":                r.Url,
	}
}

// ToRefSnapshotList 将引用列表转换为快照列表
func ToRefSnapshotList(references []*SearchReference) []map[string]any {
	refDetails := make([]map[string]any, len(references))
	for i, ref := range references {
		refDetails[i] = ref.ToSnapshot()
	}
	return refDetails
}

// ToEvidenceAnchor 将搜索引用转换为证据锚点
func (r *SearchReference) ToEvidenceAnchor(maxSnippetChars int) *EvidenceAnchor {
	if r == nil || !r.HasUsableAnchor() {
		return nil
	}
	return &EvidenceAnchor{
		DocumentId:      r.DocumentId,
		DocumentName:    r.DocumentName,
		StructureNodeId: r.StructureNodeId,
		SectionPath:     r.SectionPath,
		CanonicalPath:   r.CanonicalPath,
		ItemIndex:       r.ItemIndex,
		ParentChunkId:   r.ParentBlockId,
		ChunkId:         r.ChunkId,
		SourceType:      r.SourceType,
		Channel:         r.Channel,
		Snippet:         utils.ClipHead(r.Snippet, maxSnippetChars),
		Score:           r.Score,
	}
}
