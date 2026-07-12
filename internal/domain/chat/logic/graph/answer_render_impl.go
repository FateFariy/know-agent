package graph

import (
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// DefaultAnswerRender 默认的图谱回答渲染器。
// 依据 NavigationDecision 的章节/节点信息组装人类可读的一句话回答。
type DefaultAnswerRender struct {
}

// NewDefaultAnswerRender 创建默认图谱回答渲染器。
func NewDefaultAnswerRender() *DefaultAnswerRender {
	return &DefaultAnswerRender{}
}

var _ AnswerRender = (*DefaultAnswerRender)(nil)

func (r *DefaultAnswerRender) RenderAnswer(mode vo.ExecutionMode, decision *vo.DocumentNavigationDecision, result *entity.GraphQueryResult) string {
	if decision == nil || result == nil {
		return "没有查询到匹配的章节信息。"
	}
	parts := make([]string, 0, 4)
	switch mode {
	case vo.ExecutionModeGraphOnly:
		title := safeTitle(result.TargetSection)
		parts = append(parts, "你询问的内容位于章节 "+title+"。")
		if len(result.Children) > 0 {
			parts = append(parts, "该章节包含 "+strconv.Itoa(len(result.Children))+" 个子章节，可供进一步查阅。")
		}
		if strutil.IsNotBlank(decision.StructureAnchor.TargetSectionHint) {
			parts = append(parts, "相关提示："+decision.StructureAnchor.TargetSectionHint)
		}
	case vo.ExecutionModeGraphThenEvidence:
		title := safeTitle(result.TargetSection)
		parts = append(parts, "基于章节 "+title+" 的上下文，我将为你检索证据信息。")
		if decision.ItemAnchor != nil && decision.ItemAnchor.ItemIndex > 0 {
			parts = append(parts, "聚焦第 "+strconv.Itoa(decision.ItemAnchor.ItemIndex)+" 项内容。")
		}
	case vo.ExecutionModeRetrieval:
		title := safeTitle(result.TargetSection)
		if strutil.IsNotBlank(title) {
			parts = append(parts, "检索到相关章节 "+title+"，已为你整理相关证据。")
		} else {
			parts = append(parts, "根据你的问题，正在从相关文档中检索证据。")
		}
	default:
		parts = append(parts, "根据你的查询，正在从文档中查找相关章节与证据。")
	}
	return strings.Join(parts, " ")
}

func safeTitle(s *entity.GraphSection) string {
	if s == nil {
		return "未知章节"
	}
	if strutil.IsNotBlank(s.CanonicalPath) {
		return s.CanonicalPath
	}
	if strutil.IsNotBlank(s.NodeCode) && strutil.IsNotBlank(s.Title) {
		return s.NodeCode + " " + s.Title
	}
	if strutil.IsNotBlank(s.Title) {
		return s.Title
	}
	return "未知章节"
}
