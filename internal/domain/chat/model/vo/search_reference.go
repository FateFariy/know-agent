package vo

import (
	"fmt"
	"strconv"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

type SearchReference struct {
	ReferenceId        string  `json:"referenceId,omitempty"`        // 参考ID
	SourceType         string  `json:"sourceType,omitempty"`         // 来源类型
	Title              string  `json:"title,omitempty"`              // 标题
	Url                string  `json:"url,omitempty"`                // URL地址
	Snippet            string  `json:"snippet,omitempty"`            // 摘要
	DocumentId         int64   `json:"documentId,omitempty"`         // 文档ID
	DocumentName       string  `json:"documentName,omitempty"`       // 文档名称
	ChunkId            int64   `json:"chunkId,omitempty"`            // 块ID
	ParentBlockId      int64   `json:"parentBlockId,omitempty"`      // 父块ID
	ParentBlockNo      int     `json:"parentBlockNo,omitempty"`      // 父块序号
	ChunkNo            int     `json:"chunkNo,omitempty"`            // 块序号
	SectionPath        string  `json:"sectionPath,omitempty"`        // 节点路径
	StructureNodeId    int64   `json:"structureNodeId,omitempty"`    // 结构节点ID
	StructureNodeType  int     `json:"structureNodeType,omitempty"`  // 结构节点类型
	CanonicalPath      string  `json:"canonicalPath,omitempty"`      // 规范路径
	ItemIndex          int     `json:"itemIndex,omitempty"`          // 项索引
	Score              float64 `json:"score,omitempty"`              // 分数
	SubQuestionIndex   int     `json:"subQuestionIndex,omitempty"`   // 子问题索引
	SubQuestion        string  `json:"subQuestion,omitempty"`        // 子问题内容
	Channel            string  `json:"channel,omitempty"`            // 渠道名称
	ToolName           string  `json:"toolName,omitempty"`           // 工具名称
	KnowledgeScopeCode string  `json:"knowledgeScopeCode,omitempty"` // 知识范围代码
	KnowledgeScopeName string  `json:"knowledgeScopeName,omitempty"` // 知识范围名称
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
	ref.KnowledgeScopeCode = chunk.KnowledgeScopeCode
	ref.KnowledgeScopeName = chunk.KnowledgeScopeName
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

// ReferenceSummary 生成引用摘要（用于 debug snapshot）
func (r *SearchReference) ReferenceSummary(suffix string) string {
	title := utils.BlankToDefault(r.DocumentName, r.Title)
	path := utils.BlankToDefault(r.SectionPath, r.Url)
	refID := utils.BlankToDefault(r.ReferenceId, "-")
	if strutil.IsBlank(path) {
		return "[" + refID + "] " + title + " | " + suffix
	}
	return "[" + refID + "] " + title + " | " + path + " | " + suffix
}
