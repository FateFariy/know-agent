package graph

import (
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
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

// RenderAnswer 渲染图谱回答
func (r *DefaultAnswerRender) RenderAnswer(mode vo.ExecutionMode, decision *vo.DocumentNavigationDecision, result *entity.GraphQueryResult) string {
	if result == nil || result.TargetSection == nil {
		return ""
	}
	if mode != nil && mode.Value() == vo.ExecutionModeGraphThenEvidence.Value() {
		return r.renderGraphThenEvidence(decision, result)
	}
	return r.renderGraphOnly(decision, result)
}

// renderGraphOnly 图谱直答模式
func (r *DefaultAnswerRender) renderGraphOnly(decision *vo.DocumentNavigationDecision, result *entity.GraphQueryResult) string {
	var action string
	var question string
	if decision != nil {
		action = decision.NavigationAction
		if decision.RetrievalPlan != nil {
			question = decision.RetrievalPlan.RetrievalQuestion
		}
	}
	if action == vo.DocumentNavigationActionSectionAdjacencyLookup || r.asksAdjacency(question) {
		return r.renderAdjacency(result)
	}
	if r.asksChildren(question) || len(result.Children) > 0 {
		return r.renderChildren(result.TargetSection, result.Children)
	}
	return result.TargetSection.DisplayTitle()
}

// renderGraphThenEvidence 图谱定位后取证模式
func (r *DefaultAnswerRender) renderGraphThenEvidence(decision *vo.DocumentNavigationDecision, result *entity.GraphQueryResult) string {
	if result.TargetItem != nil {
		item := result.TargetItem
		return fmt.Sprintf("“%s”中的第%d步是：%s", result.TargetSection.DisplayTitle(), item.ItemIndex, item.DisplayText())
	}
	if len(result.MatchedItems) > 0 {
		var builder strings.Builder
		builder.WriteString("在“")
		builder.WriteString(result.TargetSection.DisplayTitle())
		builder.WriteString("”中命中了以下步骤：")
		for _, item := range result.MatchedItems {
			builder.WriteString("\n")
			builder.WriteString(r.formatItem(item))
		}
		return builder.String()
	}
	targetSection := result.TargetSection
	if strutil.IsNotBlank(targetSection.ContentText) {
		return "“" + targetSection.DisplayTitle() + "”中的相关内容如下：\n" + strutil.Trim(targetSection.ContentText)
	}
	return targetSection.DisplayTitle()
}

// renderAdjacency 渲染相邻关系（父章节、上一节、下一节）
func (r *DefaultAnswerRender) renderAdjacency(result *entity.GraphQueryResult) string {
	var builder strings.Builder
	targetSection := result.TargetSection
	parentSection := result.ParentSection
	builder.WriteString("目标章节是：“")
	builder.WriteString(targetSection.DisplayTitle())
	builder.WriteString("”。")
	if parentSection != nil {
		builder.WriteString("\n它属于：“")
		builder.WriteString(parentSection.DisplayTitle())
		builder.WriteString("”。")
	}
	builder.WriteString("\n上一节：")
	builder.WriteString(r.formatSectionOrFallback(result.PreviousSibling))
	builder.WriteString("\n下一节：")
	builder.WriteString(r.formatSectionOrFallback(result.NextSibling))
	return builder.String()
}

// renderChildren 渲染子章节列表
func (r *DefaultAnswerRender) renderChildren(targetSection *entity.GraphSection, children []*entity.GraphSection) string {
	var builder strings.Builder
	builder.WriteString("“")
	builder.WriteString(targetSection.DisplayTitle())
	builder.WriteString("”包含以下章节：")
	if len(children) == 0 {
		builder.WriteString("\n未找到直接子章节。")
		return builder.String()
	}
	for _, child := range children {
		builder.WriteString("\n- ")
		builder.WriteString(child.DisplayTitle())
	}
	return builder.String()
}

// formatItem 格式化步骤项
func (r *DefaultAnswerRender) formatItem(item *entity.GraphItem) string {
	if item == nil {
		return ""
	}
	if item.ItemIndex > 0 {
		return fmt.Sprintf("第%d步：%s", item.ItemIndex, item.DisplayText())
	}
	return item.DisplayText()
}

// formatItemIndex 格式化步骤索引
func (r *DefaultAnswerRender) formatItemIndex(idx *int) string {
	if idx == nil {
		return ""
	}
	return fmt.Sprintf("%d", *idx)
}

// formatSectionOrFallback 格式化章节或返回默认值
func (r *DefaultAnswerRender) formatSectionOrFallback(section *entity.GraphSection) string {
	if section == nil {
		return "未找到相邻章节"
	}
	return "“" + section.DisplayTitle() + "”"
}

// asksAdjacency 判断问题是否询问相邻关系
func (r *DefaultAnswerRender) asksAdjacency(question string) bool {
	if strutil.IsBlank(question) {
		return false
	}
	return strutil.ContainsAny(question, []string{"上一节", "下一节", "前一节", "属于哪个章节"})
}

// asksChildren 判断问题是否询问子章节
func (r *DefaultAnswerRender) asksChildren(question string) bool {
	if strutil.IsBlank(question) {
		return false
	}
	return strutil.ContainsAny(question, []string{"包含哪些章节", "都包含哪些章节", "有哪些小节", "有哪些章节"})
}
