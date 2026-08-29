package conversation

import (
	"context"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// EvidenceBudgetStage RAG 提示词组装实现
//
// 负责：
//  1. 基于执行计划（ConversationExecutionPlan）构建 system / user prompt
//  2. 对证据块进行预算裁剪（总预算 + 每个子问题预算）
//  3. 复用已渲染引用（避免重复输出相同证据块）
//  4. 统计渲染/省略引用详情，供上层跟踪。
type EvidenceBudgetStage struct {
	promptRenderer               adapter.PromptRenderer
	totalEvidenceBudget          int    // 总证据预算（字符数）
	perSubQuestionEvidenceBudget int    // 每个子问题的证据预算（字符数）
	systemPrompt                 string // 系统提示词
}

func NewEvidenceBudgetStage(svcCtx *svc.ServiceContext, promptRenderer adapter.PromptRenderer) *EvidenceBudgetStage {
	return &EvidenceBudgetStage{
		promptRenderer:               promptRenderer,
		totalEvidenceBudget:          svcCtx.Config.Chat.Rag.TotalEvidenceMaxChars,
		perSubQuestionEvidenceBudget: svcCtx.Config.Chat.Rag.PerSubQuestionEvidenceMaxChars,
		systemPrompt:                 svcCtx.Config.Chat.Rag.SystemPrompt,
	}
}

func (s *EvidenceBudgetStage) Name() string {
	return enum.ConversationTraceStageEvidenceBudget.Name
}

// Execute 执行证据预算与 Prompt 组装阶段，负责校验执行上下文、处理空证据兜底、发布引用、组装 Prompt 并填充调试轨迹
func (s *EvidenceBudgetStage) Execute(ctx context.Context, convCtx *Context) error {
	if convCtx.ChatMode == enum.ChatQueryModeOpenChat {
		return nil
	}

	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil || execPlan.RetrievalResult == nil {
		return nil
	}

	// 语义缓存命中：Prompt 已由缓存提供，跳过证据预算与 Prompt 组装
	if convCtx.IsCacheHit() {
		return nil
	}

	ctx = vo.OnStart(ctx, enum.ConversationTraceStageEvidenceBudget, s.Name(),
		&vo.StageInput{SummaryText: "正在组装证据与 Prompt 预算。"})

	// 空证据兜底：检索结果为空时直接返回无证据提示
	if execPlan.RetrievalResult.IsEmpty() {
		if err := convCtx.PublishThinking("当前没有足够证据，直接返回无证据兜底回复。"); err != nil {
			return err
		}
		_ = vo.OnEnd(ctx, &vo.StageOutput{SummaryText: "无 Source 证据，已完成空证据预算账本。"})
		return nil
	}

	// 发布检索引用（仅在存在引用时）
	if references := execPlan.RetrievalResult.FlattenReferences(); len(references) > 0 {
		if err := convCtx.PublishReferences(references); err != nil {
			return err
		}
	}

	// 组装 Prompt（系统提示 + 用户提示）
	promptResult, err := s.assemble(execPlan)
	if err != nil {
		logx.Errorf("Prompt 组装失败: conversationId=%s, err=%v", convCtx.ConversationId, err)
		vo.OnError(ctx, "证据预算与 Prompt 组装失败。", err)
		return err
	}

	// 填充调试轨迹，便于排查 RAG 提示词问题
	if debugTrace := convCtx.DebugTrace.Load(); debugTrace != nil {
		debugTrace.RagSystemPrompt = promptResult.SystemPrompt
		debugTrace.RagUserPrompt = promptResult.UserPrompt
	}
	execPlan.PromptAssemblyResult = promptResult

	_ = vo.OnEnd(ctx, &vo.StageOutput{
		SummaryText: "证据预算与 Prompt 组装完成。",
		Snapshot:    promptResult.ToSnapshot(execPlan.RetrievalResult),
	})

	return nil
}

// Assemble 全量组装（返回 system + user + 预算/引用统计）
func (s *EvidenceBudgetStage) assemble(plan *vo.ConversationExecutionPlan) (*vo.RagPromptAssemblyResult, error) {
	budget := newPromptBudget(s.totalEvidenceBudget, s.perSubQuestionEvidenceBudget)

	userPrompt, _ := s.promptRenderer.Render(enum.RagAnswerUser, map[string]any{
		"currentDate":          plan.CurrentDateText,
		"originalQuestion":     plan.OriginalQuestion,
		"hasRetrievalQuestion": plan.HasRetrievalQuestion(),
		"retrievalQuestion":    plan.RetrievalPlan.QuestionPlan.RetrievalQuestion,
		"hasHistoryContext":    plan.HasHistoryContext(),
		"historyContext":       utils.Trim(plan.QuestionHistoryContext.RenderedText),
		"hasSubQuestions":      len(plan.SubQuestions()) > 1,
		"subQuestions":         s.buildSubQuestions(plan),
		"evidenceBlocks":       s.buildEvidenceBlocks(plan.RetrievalResult, budget),
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

// buildSubQuestions 构建子问题列表
func (s *EvidenceBudgetStage) buildSubQuestions(plan *vo.ConversationExecutionPlan) string {
	if len(plan.SubQuestions()) < 2 {
		return ""
	}
	var b strings.Builder
	for idx, q := range plan.RetrievalPlan.QuestionPlan.SubQuestions {
		b.WriteString(strconv.Itoa(idx + 1))
		b.WriteString(". ")
		b.WriteString(utils.Trim(q))
		b.WriteString("\n")
	}
	return utils.Trim(b.String())
}

// buildSystemPrompt 构建 system prompt
func (s *EvidenceBudgetStage) buildSystemPrompt() string {
	if utils.IsNotBlank(s.systemPrompt) {
		return utils.Trim(s.systemPrompt)
	}
	rendered, _ := s.promptRenderer.Render(enum.RagAnswerSystem, nil)
	return utils.Trim(rendered)
}

// buildEvidenceBlocks 组装证据块（每个子问题对应一个块）
func (s *EvidenceBudgetStage) buildEvidenceBlocks(retrievalCtx *vo.RetrievalResult, budget *promptBudget) string {
	if retrievalCtx == nil || len(retrievalCtx.SubQuestionEvidenceList) == 0 {
		return s.renderNoEvidenceBlock()
	}
	var b strings.Builder
	for _, subQuestion := range retrievalCtx.SubQuestionEvidenceList {
		refs := s.renderSubQuestionReferences(subQuestion.References, budget)
		block, _ := s.promptRenderer.Render(enum.RagAnswerSubQuestionEvidence, map[string]any{
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
func (s *EvidenceBudgetStage) renderSubQuestionReferences(references []*vo.SearchReference, budget *promptBudget) string {
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
			reuse, _ := s.promptRenderer.Render(enum.RagAnswerReuseReference, map[string]any{
				"referenceId": utils.Trim(ref.ReferenceId),
			})
			reuse = reuse + "\n"
			if budget.tryConsume(utils.Len(reuse)) {
				b.WriteString(reuse)
			}
			continue
		}

		var block string
		if strings.EqualFold(ref.SourceType, "WEB") {
			rendered, _ := s.promptRenderer.Render(enum.RagAnswerWebReference, map[string]any{
				"referenceId": ref.ReferenceId,
				"title":       utils.BlankToDefault(ref.Title, "网页来源"),
				"url":         utils.BlankToDefault(ref.Url, "未知"),
				"snippet":     utils.ClipHead(ref.Snippet, 900),
			})
			block = rendered + "\n\n"
		} else {
			docName := strutil.Trim(utils.BlankToDefault(ref.DocumentName, ref.Title))
			rendered, _ := s.promptRenderer.Render(enum.RagAnswerDocumentReference, map[string]any{
				"referenceId":  ref.ReferenceId,
				"documentName": utils.BlankToDefault(docName, "文档来源"),
				"sectionPath":  utils.BlankToDefault(ref.SectionPath, "未识别"),
				"snippet":      utils.ClipHead(ref.Snippet, 1100),
			})
			block = rendered + "\n\n"
		}
		if budget.tryConsume(utils.Len(block)) {
			b.WriteString(block)
			renderedKeys[ref.UniqueKey()] = struct{}{}
			budget.markRendered(ref.ReferenceSummary("已纳入 Prompt"))
		} else {
			budget.markOmitted(ref.ReferenceSummary("超出上下文预算，已省略"))
			omitted, _ := s.promptRenderer.Render(enum.RagAnswerOmittedEvidence, nil)
			b.WriteString(omitted)
			b.WriteString("\n")
			break
		}
	}
	return strutil.Trim(b.String())
}

// renderNoEvidenceBlock 渲染无证据块
func (s *EvidenceBudgetStage) renderNoEvidenceBlock() string {
	rendered, _ := s.promptRenderer.Render(enum.RagAnswerNoEvidence, nil)
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
