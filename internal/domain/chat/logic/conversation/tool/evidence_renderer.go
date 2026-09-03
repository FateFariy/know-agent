package tool

import (
	"fmt"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// EvidenceRenderer 把检索结果渲染为带编号引用的证据文本
type EvidenceRenderer struct {
	totalEvidenceBudget          int // 总证据预算（字符数）
	perSubQuestionEvidenceBudget int // 每个子问题证据预算（字符数）
}

// NewEvidenceRenderer 创建证据渲染器
func NewEvidenceRenderer(svcCtx *svc.ServiceContext) *EvidenceRenderer {
	return &EvidenceRenderer{
		totalEvidenceBudget:          svcCtx.Config.Chat.Rag.TotalEvidenceMaxChars,
		perSubQuestionEvidenceBudget: svcCtx.Config.Chat.Rag.PerSubQuestionEvidenceMaxChars,
	}
}

// RenderEvidence 渲染检索结果（多个子问题 → 多个证据块，块内引用支持预算裁剪与去重复用）。
func (r *EvidenceRenderer) RenderEvidence(retrievalCtx *vo.RetrievalResult) string {
	if retrievalCtx == nil || len(retrievalCtx.SubQuestionEvidenceList) == 0 {
		return ""
	}
	budget := newPromptBudget(r.totalEvidenceBudget, r.perSubQuestionEvidenceBudget)
	var b strings.Builder
	for _, subQuestion := range retrievalCtx.SubQuestionEvidenceList {
		refs := r.renderSubQuestionReferences(subQuestion.References, budget)
		b.WriteString(fmt.Sprintf("## 子问题 %d: %s\n", subQuestion.SubQuestionIndex, subQuestion.SubQuestion))
		b.WriteString(refs)
		b.WriteString("\n\n")
	}
	return utils.Trim(b.String())
}

// renderSubQuestionReferences 渲染单个子问题的引用列表（复用 + 预算裁剪）
func (r *EvidenceRenderer) renderSubQuestionReferences(references []*vo.SearchReference, budget *promptBudget) string {
	renderedKeys := make(map[string]struct{})
	if len(references) == 0 {
		return "- 当前子问题没有检索到证据\n"
	}
	budget.resetSubQuestionBudget()
	var b strings.Builder
	for _, ref := range references {
		if ref == nil {
			continue
		}
		if _, exists := renderedKeys[ref.UniqueKey()]; exists {
			reuse := fmt.Sprintf("- 复用证据 [%s]\n", utils.Trim(ref.ReferenceId))
			if budget.tryConsume(utils.Len(reuse)) {
				b.WriteString(reuse)
			}
			continue
		}

		var block string
		if strings.EqualFold(ref.SourceType, "WEB") {
			block = fmt.Sprintf("[%s] 网页：%s；链接：%s\n摘要：%s\n\n",
				ref.ReferenceId,
				utils.BlankToDefault(ref.Title, "网页来源"),
				utils.BlankToDefault(ref.Url, "未知"),
				utils.ClipHead(ref.Snippet, 900),
			)
		} else {
			docName := utils.Trim(utils.BlankToDefault(ref.DocumentName, ref.Title))
			block = fmt.Sprintf("[%s] 文档：%s；章节：%s\n内容：%s\n\n",
				ref.ReferenceId,
				utils.BlankToDefault(docName, "文档来源"),
				utils.BlankToDefault(ref.SectionPath, "未识别"),
				utils.ClipHead(ref.Snippet, 1100),
			)
		}
		if budget.tryConsume(utils.Len(block)) {
			b.WriteString(block)
			renderedKeys[ref.UniqueKey()] = struct{}{}
			budget.markRendered(ref.ReferenceSummary("已纳入证据"))
		} else {
			budget.markOmitted(ref.ReferenceSummary("超出证据预算，已省略"))
			omitted := "- 其余证据因上下文预算限制已省略\n"
			b.WriteString(omitted)
			break
		}
	}
	return utils.Trim(b.String())
}

// -------------------- PromptBudget --------------------

// promptBudget 证据文本预算
type promptBudget struct {
	totalBudget              int      // 总预算
	perSubQuestionBudget     int      // 每个子问题预算
	remainingTotal           int      // 剩余总预算
	remainingSubQuestion     int      // 剩余子问题预算
	renderedReferenceCount   int      // 已渲染引用数量
	omittedReferenceCount    int      // 已省略引用数量
	renderedReferenceDetails []string // 已渲染引用详情列表
	omittedReferenceDetails  []string // 已省略引用详情列表
}

// newPromptBudget 创建预算对象
func newPromptBudget(totalBudget, perSubQuestionBudget int) *promptBudget {
	total := max(totalBudget, 0)
	perSQ := max(perSubQuestionBudget, 0)
	return &promptBudget{
		totalBudget:          total,
		perSubQuestionBudget: perSQ,
		remainingTotal:       total,
		remainingSubQuestion: perSQ,
	}
}

// resetSubQuestionBudget 切换到下一个子问题时重置子问题预算
func (p *promptBudget) resetSubQuestionBudget() {
	p.remainingSubQuestion = p.perSubQuestionBudget
}

// tryConsume 尝试消费 size 个字符，返回是否成功
func (p *promptBudget) tryConsume(size int) bool {
	if p.totalBudget <= 0 || p.perSubQuestionBudget <= 0 {
		return false
	}
	if size > p.remainingTotal || size > p.remainingSubQuestion {
		return false
	}
	p.remainingTotal -= size
	p.remainingSubQuestion -= size
	return true
}

// markRendered 标记一条引用已渲染
func (p *promptBudget) markRendered(detail string) {
	p.renderedReferenceCount++
	if detail != "" {
		p.renderedReferenceDetails = append(p.renderedReferenceDetails, detail)
	}
}

// markOmitted 标记一条引用已省略
func (p *promptBudget) markOmitted(detail string) {
	p.omittedReferenceCount++
	if detail != "" {
		p.omittedReferenceDetails = append(p.omittedReferenceDetails, detail)
	}
}
