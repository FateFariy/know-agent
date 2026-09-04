package vo

import (
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

const (
	maxFollowUpHintAnchors   = 3   // 追问提示最多展示的锚点数
	followUpHintSnippetChars = 120 // 追问提示中锚点片段最大字符数
)

type EvidenceAnchor struct {
	DocumentId        int64   // 文档ID
	DocumentName      string  // 文档名称
	TaskID            int64   // 任务ID
	KnowledgeBaseId   int64   // 知识库ID
	KnowledgeBaseName string  // 知识库名称
	StructureNodeId   int64   // 结构节点ID
	SectionPath       string  // 章节路径
	CanonicalPath     string  // 规范路径
	ItemIndex         int     // 条目索引
	ParentChunkId     int64   // 父块ID
	ChunkId           int64   // 分块ID
	SourceType        string  // 来源类型
	Channel           string  // 来源渠道
	Snippet           string  // 内容片段
	Score             float64 // 相关性得分
}

// HasAnchorIdentity 检查锚点是否有可用的标识字段
func (a *EvidenceAnchor) HasAnchorIdentity() bool {
	if a == nil {
		return false
	}
	return a.DocumentId != 0 ||
		a.StructureNodeId != 0 ||
		a.ParentChunkId != 0 ||
		a.ChunkId != 0 ||
		strutil.IsNotBlank(a.SectionPath)
}

// AnchorHint 生成单个锚点的提示字符串
func (a *EvidenceAnchor) AnchorHint() string {
	if a == nil {
		return ""
	}
	var builder strings.Builder
	appendHintPart := func(name string, value any) {
		trimmed := utils.Trim(utils.ToString(value))
		if trimmed == "" {
			return
		}
		if builder.Len() > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(name)
		builder.WriteString("=")
		builder.WriteString(trimmed)
	}

	appendHintPart("documentId", a.DocumentId)
	appendHintPart("sectionPath", a.SectionPath)
	appendHintPart("structureNodeId", a.StructureNodeId)
	appendHintPart("parentBlockId", a.ParentChunkId)
	appendHintPart("chunkId", a.ChunkId)
	return utils.Trim(builder.String())
}

func (a *EvidenceAnchor) ToRetrievalContextAnchor() *RetrievalContextAnchor {
	if a == nil || a.DocumentId == 0 {
		return nil
	}
	return &RetrievalContextAnchor{
		DocumentId:      a.DocumentId,
		SectionPath:     a.SectionPath,
		StructureNodeId: a.StructureNodeId,
		ParentChunkId:   a.ParentChunkId,
		ChunkId:         a.ChunkId,
		Source:          "FINAL_EVIDENCE_ANCHOR",
	}
}

type EvidenceAnchors []*EvidenceAnchor

// RenderFollowUpHint 渲染上一轮证据落点的精简提示，供 agent 指令注入。
// 仅用于解析追问中的指代与定位检索，不作为当前回答的事实来源。
func (anchors EvidenceAnchors) RenderFollowUpHint() string {
	if len(anchors) == 0 {
		return ""
	}
	var b strings.Builder
	for i, a := range anchors {
		if i >= maxFollowUpHintAnchors {
			break
		}
		if a == nil {
			continue
		}
		b.WriteString("- 文档：")
		b.WriteString(utils.BlankToDefault(utils.Trim(a.DocumentName), "未命名"))
		if utils.IsNotBlank(a.SectionPath) {
			b.WriteString("；章节：")
			b.WriteString(utils.Trim(a.SectionPath))
		}
		if a.ItemIndex > 0 {
			b.WriteString("；条目：")
			b.WriteString(utils.ToString(a.ItemIndex))
		}
		if utils.IsNotBlank(a.Snippet) {
			b.WriteString("\n  片段：")
			b.WriteString(utils.ClipHead(utils.Trim(a.Snippet), followUpHintSnippetChars))
		}
		b.WriteByte('\n')
	}
	body := utils.Trim(b.String())
	if body == "" {
		return ""
	}
	return "上一轮回答的证据落点（仅用于解析“那个”“上面 [编号]”等指代，不作为事实来源）：\n" + body
}
