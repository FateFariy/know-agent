package vo

import (
	"fmt"

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
	ParentChunkId     int64   `json:"parentChunkId"`     // 父块ID
	ParentChunkNo     int     `json:"parentChunkNo"`     // 父块序号
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
	KnowledgeBaseId   int64   `json:"knowledgeBaseId"`   // 知识库ID
	KnowledgeBaseName string  `json:"knowledgeBaseName"` // 知识库名称

	// todo 以下字段待完善
	FinalSelectionReason         string   `json:"finalSelectionReason"`
	EvidenceApplicabilityStatus  string   `json:"evidenceApplicabilityStatus"`
	EvidenceApplicabilityReason  string   `json:"evidenceApplicabilityReason"`
	ContextIdentity              string   `json:"contextIdentity"`
	CitationIdentity             string   `json:"citationIdentity"`
	CitationEvidenceType         string   `json:"citationEvidenceType"`
	ContextOnly                  bool     `json:"contextOnly"`
	SourceEvidenceResolved       bool     `json:"sourceEvidenceResolved"`
	TaskId                       int64    `json:"taskId"`
	ChunkType                    string   `json:"chunkType"`
	SourceAuthoredHeading        bool     `json:"sourceAuthoredHeading"`
	PageNo                       int      `json:"pageNo"`
	PageRange                    string   `json:"pageRange"`
	BboxJson                     string   `json:"bboxJson"`
	SourceBlockIds               string   `json:"sourceBlockIds"`
	TableId                      int64    `json:"tableId"`
	TableNo                      int      `json:"tableNo"`
	TableTitle                   string   `json:"tableTitle"`
	TableOperation               string   `json:"tableOperation"`
	TableMetricColumn            string   `json:"tableMetricColumn"`
	TableGroupByColumn           string   `json:"tableGroupByColumn"`
	TableMatchedRowCount         int      `json:"tableMatchedRowCount"`
	TableEvidenceRowIds          []int64  `json:"tableEvidenceRowIds"`
	TableEvidenceRowNos          []int    `json:"tableEvidenceRowNos"`
	TableEvidenceColumnIds       []int64  `json:"tableEvidenceColumnIds"`
	TableEvidenceColumnNos       []int    `json:"tableEvidenceColumnNos"`
	TableEvidenceColumnNames     []string `json:"tableEvidenceColumnNames"`
	TableEvidenceCellIds         []int64  `json:"tableEvidenceCellIds"`
	TableEvidenceCellCoordinates []string `json:"tableEvidenceCellCoordinates"`
	TableEvidenceCellBboxJsons   []string `json:"tableEvidenceCellBboxJsons"`

	KgEntityId                                 int64   `json:"kgEntityId"`
	KgEntityName                               string  `json:"kgEntityName"`
	KgCanonicalEntityKey                       string  `json:"kgCanonicalEntityKey"`
	KgCanonicalEntityName                      string  `json:"kgCanonicalEntityName"`
	KgCanonicalEntityCount                     int     `json:"kgCanonicalEntityCount"`
	KgCanonicalDocumentCount                   int     `json:"kgCanonicalDocumentCount"`
	KgRelatedEntityId                          int64   `json:"kgRelatedEntityId"`
	KgRelatedEntityName                        string  `json:"kgRelatedEntityName"`
	KgRelationId                               int64   `json:"kgRelationId"`
	KgRelationType                             string  `json:"kgRelationType"`
	KgRelationGroupKey                         string  `json:"kgRelationGroupKey"`
	KgRelationGroupRelationCount               int     `json:"kgRelationGroupRelationCount"`
	KgRelationGroupEvidenceCount               int     `json:"kgRelationGroupEvidenceCount"`
	KgRelationGroupDocumentCount               int     `json:"kgRelationGroupDocumentCount"`
	KgEvidenceId                               int64   `json:"kgEvidenceId"`
	KgGraphPath                                string  `json:"kgGraphPath"`
	KgHopCount                                 int     `json:"kgHopCount"`
	KgQueryPlanSource                          string  `json:"kgQueryPlanSource"`
	KgQueryPlanAnswerTypes                     string  `json:"kgQueryPlanAnswerTypes"`
	KgQueryPlanEntities                        string  `json:"kgQueryPlanEntities"`
	KgNhopSeedEntityId                         int64   `json:"kgNhopSeedEntityId"`
	KgNhopSeedEntityName                       string  `json:"kgNhopSeedEntityName"`
	KgNhopPath                                 string  `json:"kgNhopPath"`
	KgCrossDocumentCommunityKey                string  `json:"kgCrossDocumentCommunityKey"`
	KgCommunitySummaryOnly                     bool    `json:"kgCommunitySummaryOnly"`
	KgCrossDocumentCommunityEntityCount        int     `json:"kgCrossDocumentCommunityEntityCount"`
	KgCrossDocumentCommunityRelationGroupCount int     `json:"kgCrossDocumentCommunityRelationGroupCount"`
	KgCrossDocumentCommunityEvidenceCount      int     `json:"kgCrossDocumentCommunityEvidenceCount"`
	KgCrossDocumentCommunityDocumentCount      int     `json:"kgCrossDocumentCommunityDocumentCount"`
	KgCommunityRankScore                       float64 `json:"kgCommunityRankScore"`
	KgCommunityRankReasons                     string  `json:"kgCommunityRankReasons"`
	KgQualityScore                             float64 `json:"kgQualityScore"`
	KgQualityReasons                           string  `json:"kgQualityReasons"`
	KgNoiseReasons                             string  `json:"kgNoiseReasons"`
	KgPagerank                                 float64 `json:"kgPagerank"`
	KgRankPosition                             int     `json:"kgRankPosition"`
	KgDegree                                   int     `json:"kgDegree"`

	RaptorNodeId       int64  `json:"raptorNodeId"`
	RaptorNodeTitle    string `json:"raptorNodeTitle"`
	RaptorNodeLevel    int    `json:"raptorNodeLevel"`
	RaptorSummary      string `json:"raptorSummary"`
	RaptorSourceStatus string `json:"raptorSourceStatus"`

	QuoteText         string `json:"quoteText"`
	GenerationVisible bool   // 生成时是否可见
}

// UniqueKey 生成唯一键
func (r *SearchReference) UniqueKey() string {
	if r.ParentChunkId > 0 {
		return fmt.Sprintf("PARENT:%d", r.ParentChunkId)
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
		r.ParentChunkId != 0 ||
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
		"parentBlockId":      r.ParentChunkId,
		"parentBlockNo":      r.ParentChunkNo,
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
		ParentChunkId:   r.ParentChunkId,
		ChunkId:         r.ChunkId,
		SourceType:      r.SourceType,
		Channel:         r.Channel,
		Snippet:         utils.ClipHead(r.Snippet, maxSnippetChars),
		Score:           r.Score,
	}
}
