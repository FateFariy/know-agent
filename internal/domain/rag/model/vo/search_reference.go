package vo

import (
	"fmt"
	"strconv"

	"github.com/duke-git/lancet/v2/convertor"

	"github.com/swiftbit/know-agent/common/utils"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

type SearchReference struct {
	ReferenceId           string  // 参考ID
	SourceType            string  // 来源类型
	Title                 string  // 标题
	Url                   string  // URL地址
	Snippet               string  // 摘要
	DocumentId            int64   // 文档ID
	DocumentName          string  // 文档名称
	ChunkId               int64   // 块ID
	ParentBlockId         int64   // 父块ID
	ParentBlockNo         int     // 父块序号
	ChunkNo               int     // 块序号
	SectionPath           string  // 节点路径
	StructureNodeNodeId   int     // 结构节点ID
	StructureNodeNodeType int     // 结构节点类型
	CanonicalPath         string  // 规范路径
	ItemIndex             int     // 项索引
	Score                 float64 // 分数
	SubQuestionIndex      int     // 子问题索引
	SubQuestion           string  // 子问题内容
	Channel               string  // 渠道名称
	ToolName              string  // 工具名称
	KnowledgeScopeCode    string  // 知识范围代码
	KnowledgeScopeName    string  // 知识范围名称
}

// NewSearchReference 创建新的 SearchReference 实例
func NewSearchReference(document *klvo.Document, subQuestionIndex int, referenceNumber int, subQuestion string) *SearchReference {
	if document == nil {
		return &SearchReference{}
	}

	meta := document.Meta
	if meta == nil {
		meta = make(map[string]interface{})
	}

	sourceType := utils.BlankToDefault(convertor.ToString(meta[klvo.MetaSourceType]), "DOCUMENT")
	ref := &SearchReference{
		ReferenceId:      strconv.Itoa(referenceNumber),
		SourceType:       sourceType,
		Snippet:          document.Content,
		SubQuestionIndex: subQuestionIndex,
		SubQuestion:      subQuestion,
		Score:            document.Score,
		Channel:          convertor.ToString(meta[klvo.MetaChannel]),
	}
	if sourceType == "WEB" {
		ref.Title = utils.BlankToDefault(convertor.ToString(meta[klvo.MetaTitle]), "网页来源")
		ref.Url = convertor.ToString(meta[klvo.MetaURL])
		ref.ToolName = utils.BlankToDefault(convertor.ToString(meta[klvo.MetaToolName]), "tavily_search")
		return ref
	}
	ref.Title = utils.BlankToDefault(convertor.ToString(meta[klvo.MetaDocumentName]), "文档片段")
	ref.DocumentId, _ = convertor.ToInt(meta[klvo.MetaDocumentID])
	ref.DocumentName = convertor.ToString(meta[klvo.MetaDocumentName])
	ref.ParentBlockId, _ = convertor.ToInt(meta[klvo.MetaParentBlockID])
	ref.ParentBlockNo = toInt(meta[klvo.MetaParentBlockNo])
	ref.ChunkId, _ = convertor.ToInt(meta[klvo.MetaChunkID])
	ref.ChunkNo = toInt(meta[klvo.MetaChunkNo])
	ref.SectionPath = convertor.ToString(meta[klvo.MetaSectionPath])
	ref.StructureNodeNodeId = toInt(meta[klvo.MetaStructureNodeID])
	ref.StructureNodeNodeType = toInt(meta[klvo.MetaStructureNodeType])
	ref.CanonicalPath = convertor.ToString(meta[klvo.MetaCanonicalPath])
	ref.ItemIndex = toInt(meta[klvo.MetaItemIndex])
	ref.KnowledgeScopeCode = convertor.ToString(meta[klvo.MetaKnowledgeScopeCode])
	ref.KnowledgeScopeName = convertor.ToString(meta[klvo.MetaKnowledgeScopeName])
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

func toInt(value any) int {
	i, _ := convertor.ToInt(value)
	return int(i)
}
