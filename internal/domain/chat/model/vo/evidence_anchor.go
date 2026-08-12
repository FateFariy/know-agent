package vo

import (
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
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
	ParentBlockId     int64   // 父块ID
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
		a.ParentBlockId != 0 ||
		a.ChunkId != 0 ||
		strutil.IsNotBlank(a.SectionPath)
}

type EvidenceAnchors []*EvidenceAnchor

// RenderStructuredContext 渲染结构化上下文（证据锚点列表）
func (anchors EvidenceAnchors) RenderStructuredContext(budget int) string {
	if len(anchors) == 0 || budget <= 0 {
		return ""
	}
	builder := strings.Builder{}
	appendAnchorField := func(name string, value any) {
		text := utils.ToString(value)
		if text == "" {
			return
		}
		builder.WriteString(name)
		builder.WriteString(": ")
		builder.WriteString(text)
		builder.WriteByte('\n')
	}
	builder.WriteString("上一轮可继承证据锚点（仅用于解析指代和限定范围，不作为事实证据）：\n")
	for _, anchor := range anchors {
		if anchor != nil {
			builder.WriteString("- 文档: ")
			builder.WriteString(utils.BlankToDefault(anchor.DocumentName, "-"))
			builder.WriteByte('\n')
			appendAnchorField("  章节", anchor.SectionPath)
			appendAnchorField("  canonicalPath", anchor.CanonicalPath)
			appendAnchorField("  structureNodeId", anchor.StructureNodeId)
			appendAnchorField("  parentBlockId", anchor.ParentBlockId)
			appendAnchorField("  chunkId", anchor.ChunkId)
			appendAnchorField("  itemIndex", anchor.ItemIndex)
			appendAnchorField("  snippet", utils.ClipHead(anchor.Snippet, 300))
		}
	}
	return utils.ClipHead(strings.TrimSpace(builder.String()), budget)
}
