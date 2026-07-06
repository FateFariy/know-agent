package rag

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// PromptBuilder RAG 提示词组装实现
//
// 负责：
//  1. 基于执行计划（ConversationExecutionPlan）构建 system / user prompt
//  2. 对证据块进行预算裁剪（总预算 + 每个子问题预算）
//  3. 复用已渲染引用（避免重复输出相同证据块）
//  4. 统计渲染/省略引用详情，供上层跟踪。
type PromptBuilder struct {
	promptRenderer               logic.PromptTemplateLogic
	totalEvidenceBudget          int    // 总证据预算（字符数）
	perSubQuestionEvidenceBudget int    // 每个子问题的证据预算（字符数）
	systemPrompt                 string // 系统提示词
}

// NewPromptBuilder 创建 RAG 提示词组装实现
func NewPromptBuilder(svcCtx *svc.ServiceContext, promptRenderer logic.PromptTemplateLogic) *PromptBuilder {
	return &PromptBuilder{
		promptRenderer:               promptRenderer,
		totalEvidenceBudget:          svcCtx.Config.Chat.Rag.TotalEvidenceMaxChars,
		perSubQuestionEvidenceBudget: svcCtx.Config.Chat.Rag.PerSubQuestionEvidenceMaxChars,
		systemPrompt:                 svcCtx.Config.Chat.Rag.SystemPrompt,
	}
}

// Assemble 全量组装（返回 system + user + 预算/引用统计）
func (s *PromptBuilder) Assemble(_ context.Context, plan *vo.ConversationExecutionPlan, retrievalCtx *vo.RagRetrievalContext) (*vo.RagPromptAssemblyResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan not is nil")
	}
	budget := newPromptBudget(s.totalEvidenceBudget, s.perSubQuestionEvidenceBudget)

	userPrompt, _ := s.promptRenderer.Render(prompt.RagAnswerUser, map[string]any{
		"currentDate":          plan.CurrentDateText,
		"originalQuestion":     plan.OriginalQuestion,
		"hasRetrievalQuestion": s.hasRetrievalQuestion(plan),
		"retrievalQuestion":    plan.RetrievalQuestion,
		"hasHistoryContext":    s.hasHistoryContext(plan),
		"historyContext":       s.buildHistoryContext(plan),
		"hasSubQuestions":      len(plan.RetrievalSubQuestions) > 1,
		"subQuestions":         s.buildSubQuestions(plan),
		"evidenceBlocks":       s.buildEvidenceBlocks(retrievalCtx, budget),
	})

	return &vo.RagPromptAssemblyResult{
		SystemPrompt:             s.buildSystemPrompt(),
		UserPrompt:               strutil.Trim(userPrompt),
		TotalBudget:              budget.totalBudget,
		PerSubQuestionBudget:     budget.perSubQuestionBudget,
		RenderedReferenceCount:   budget.renderedReferenceCount,
		OmittedReferenceCount:    budget.omittedReferenceCount,
		RenderedReferenceDetails: append([]string{}, budget.renderedReferenceDetails...),
		OmittedReferenceDetails:  append([]string{}, budget.omittedReferenceDetails...),
	}, nil
}

// hasRetrievalQuestion 是否有检索问题
func (s *PromptBuilder) hasRetrievalQuestion(plan *vo.ConversationExecutionPlan) bool {
	return strutil.IsNotBlank(plan.RetrievalQuestion) && plan.RetrievalQuestion != plan.OriginalQuestion
}

// hasHistoryContext 是否有历史上下文
func (s *PromptBuilder) hasHistoryContext(plan *vo.ConversationExecutionPlan) bool {
	if plan.QuestionHistoryContext == nil {
		return false
	}
	return strutil.IsNotBlank(plan.QuestionHistoryContext.RenderedText)
}

// buildHistoryContext 构建历史上下文
func (s *PromptBuilder) buildHistoryContext(plan *vo.ConversationExecutionPlan) string {
	if s.hasHistoryContext(plan) {
		return strutil.Trim(plan.QuestionHistoryContext.RenderedText)
	}
	return ""
}

// buildSubQuestions 构建子问题列表
func (s *PromptBuilder) buildSubQuestions(plan *vo.ConversationExecutionPlan) string {
	if len(plan.RetrievalSubQuestions) < 2 {
		return ""
	}
	var b strings.Builder
	for idx, q := range plan.RetrievalSubQuestions {
		b.WriteString(strconv.Itoa(idx + 1))
		b.WriteString(". ")
		b.WriteString(strutil.Trim(q))
		b.WriteString("\n")
	}
	return strutil.Trim(b.String())
}

// buildSystemPrompt 构建 system prompt
func (s *PromptBuilder) buildSystemPrompt() string {
	if strutil.IsNotBlank(s.systemPrompt) {
		return strutil.Trim(s.systemPrompt)
	}
	rendered, _ := s.promptRenderer.Render(prompt.RagAnswerSystem, nil)
	return strutil.Trim(rendered)
}

// buildEvidenceBlocks 组装证据块（每个子问题对应一个块）
func (s *PromptBuilder) buildEvidenceBlocks(retrievalCtx *vo.RagRetrievalContext, budget *promptBudget) string {
	if retrievalCtx == nil || len(retrievalCtx.SubQuestionEvidenceList) == 0 {
		return s.renderNoEvidenceBlock()
	}
	var b strings.Builder
	for _, subQuestion := range retrievalCtx.SubQuestionEvidenceList {
		refs := s.renderSubQuestionReferences(subQuestion.References, budget)
		block, _ := s.promptRenderer.Render(prompt.RagAnswerSubQuestionEvidence, map[string]any{
			"subQuestionIndex": subQuestion.SubQuestionIndex,
			"subQuestion":      strutil.Trim(subQuestion.SubQuestion),
			"references":       refs,
		})
		b.WriteString(strutil.Trim(block))
		b.WriteString("\n\n")
	}
	return strutil.Trim(b.String())
}

// renderSubQuestionReferences 渲染单个子问题的引用列表（复用 + 预算裁剪）
func (s *PromptBuilder) renderSubQuestionReferences(references []*vo.SearchReference, budget *promptBudget) string {
	renderedKeys := make(map[string]struct{})
	if len(references) == 0 {
		return s.renderNoEvidenceBlock()
	}
	budget.resetSubQuestionBudget()
	var b strings.Builder
	for _, ref := range references {
		if ref == nil {
			continue
		}
		if _, exists := renderedKeys[ref.UniqueKey()]; exists {
			reuse, _ := s.promptRenderer.Render(prompt.RagAnswerReuseReference, map[string]any{
				"referenceId": strutil.Trim(ref.ReferenceId),
			})
			reuse = reuse + "\n"
			if budget.tryConsume(utf8.RuneCountInString(reuse)) {
				b.WriteString(reuse)
			}
			continue
		}

		var block string
		if strings.EqualFold(ref.SourceType, "WEB") {
			rendered, _ := s.promptRenderer.Render(prompt.RagAnswerWebReference, map[string]any{
				"referenceId": ref.ReferenceId,
				"title":       utils.BlankToDefault(ref.Title, "网页来源"),
				"url":         utils.BlankToDefault(ref.Url, "未知"),
				"snippet":     utils.ClipHead(ref.Snippet, 900),
			})
			block = rendered + "\n\n"
		} else {
			docName := strutil.Trim(utils.BlankToDefault(ref.DocumentName, ref.Title))
			rendered, _ := s.promptRenderer.Render(prompt.RagAnswerDocumentReference, map[string]any{
				"referenceId":  ref.ReferenceId,
				"documentName": utils.BlankToDefault(docName, "文档来源"),
				"sectionPath":  utils.BlankToDefault(ref.SectionPath, "未识别"),
				"snippet":      utils.ClipHead(ref.Snippet, 1100),
			})
			block = rendered + "\n\n"
		}
		if budget.tryConsume(utf8.RuneCountInString(block)) {
			b.WriteString(block)
			renderedKeys[ref.UniqueKey()] = struct{}{}
			budget.markRendered(ref.ReferenceSummary("已纳入 Prompt"))
		} else {
			budget.markOmitted(ref.ReferenceSummary("超出上下文预算，已省略"))
			omitted, _ := s.promptRenderer.Render(prompt.RagAnswerOmittedEvidence, nil)
			b.WriteString(omitted)
			b.WriteString("\n")
			break
		}
	}
	return strutil.Trim(b.String())
}

// renderNoEvidenceBlock 渲染无证据块
func (s *PromptBuilder) renderNoEvidenceBlock() string {
	rendered, _ := s.promptRenderer.Render(prompt.RagAnswerNoEvidence, nil)
	return rendered + "\n"
}

// -------------------- PromptBudget --------------------

// promptBudget prompt 组装预算
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

// tryConsume 尝试消费 size 个字符，返回是否成功；
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
