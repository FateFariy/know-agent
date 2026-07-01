package vo

import (
	"fmt"
	"strconv"

	"github.com/duke-git/lancet/v2/convertor"

	"github.com/swiftbit/know-agent/common/utils"
)

type SearchReference struct {
	ReferenceId        string  // 参考ID
	SourceType         string  // 来源类型
	Title              string  // 标题
	Url                string  // URL地址
	Snippet            string  // 摘要
	DocumentId         int64   // 文档ID
	DocumentName       string  // 文档名称
	ChunkId            int64   // 块ID
	ParentBlockId      int64   // 父块ID
	ParentBlockNo      int     // 父块序号
	ChunkNo            int     // 块序号
	SectionPath        string  // 节点路径
	StructureNodeId    int64   // 结构节点ID
	StructureNodeType  int     // 结构节点类型
	CanonicalPath      string  // 规范路径
	ItemIndex          int     // 项索引
	Score              float64 // 分数
	SubQuestionIndex   int     // 子问题索引
	SubQuestion        string  // 子问题内容
	Channel            string  // 渠道名称
	ToolName           string  // 工具名称
	KnowledgeScopeCode string  // 知识范围代码
	KnowledgeScopeName string  // 知识范围名称
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
