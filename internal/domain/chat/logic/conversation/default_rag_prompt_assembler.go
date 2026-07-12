package conversation

import (
	"context"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// DefaultRagPromptAssembler 默认的 RAG Prompt 装配器。
// 基于检索得到的证据列表与原始问题，组装适合大模型的 system/user Prompt。
type DefaultRagPromptAssembler struct {
	maxReferenceBudget int
}

// NewDefaultRagPromptAssembler 创建默认 RAG Prompt 装配器（默认预算 30）。
func NewDefaultRagPromptAssembler() *DefaultRagPromptAssembler {
	return &DefaultRagPromptAssembler{maxReferenceBudget: 30}
}

var _ RagPromptAssembler = (*DefaultRagPromptAssembler)(nil)

// Assemble 组装 RAG Prompt。
// 1. systemPrompt：作为角色与规则指示；
// 2. userPrompt：由检索证据 + 用户原始问题拼装。
// 同时会按照预算过滤、记录引用信息供前端回溯。
func (a *DefaultRagPromptAssembler) Assemble(
	ctx context.Context,
	plan *vo.ConversationExecutionPlan,
	retrievalCtx *vo.RagRetrievalContext,
) (*vo.RagPromptAssemblyResult, error) {
	if plan == nil || retrievalCtx == nil {
		return nil, fmt.Errorf("invalid value")
	}

	references := retrievalCtx.FlattenReferences()
	budget := a.maxReferenceBudget
	if len(references) < budget {
		budget = len(references)
	}

	rendered := make([]string, 0, budget)
	renderedDetails := make([]string, 0, budget)
	for i := 0; i < budget; i++ {
		ref := references[i]
		docName := strutil.BlankToDefault(ref.DocumentName, ref.Title)
		header := fmt.Sprintf("[%d] %s", i+1, docName)
		if strutil.IsNotBlank(ref.SectionPath) {
			header = header + "（" + ref.SectionPath + "）"
		}
		body := strutil.BlankToDefault(ref.Content, ref.Title)
		rendered = append(rendered, fmt.Sprintf("%s\n%s", header, body))
		renderedDetails = append(renderedDetails, header)
	}

	omittedDetails := make([]string, 0, len(references)-budget)
	for i := budget; i < len(references); i++ {
		ref := references[i]
		docName := strutil.BlankToDefault(ref.DocumentName, ref.Title)
		omittedDetails = append(omittedDetails, fmt.Sprintf("[%d] %s", i+1, docName))
	}

	evidence := strings.Join(rendered, "\n\n")
	question := plan.UserQuestion
	if strutil.IsBlank(question) {
		question = retrievalCtx.RetrievalQuestion
	}

	systemPrompt := "你是一个专业的文档助手，会根据用户提供的文档证据生成准确、简洁的回答。回答需要引用原文中的章节与编号，并保持客观中立。"
	userPrompt := fmt.Sprintf(
		"## 文档证据\n%s\n\n## 问题\n%s\n\n请基于以上文档证据，为用户的问题提供准确、简洁的回答。若证据不足以回答，明确告知用户。",
		evidence, question,
	)

	return &vo.RagPromptAssemblyResult{
		SystemPrompt:             systemPrompt,
		UserPrompt:               userPrompt,
		TotalBudget:              a.maxReferenceBudget,
		PerSubQuestionBudget:     a.maxReferenceBudget,
		RenderedReferenceCount:   len(rendered),
		OmittedReferenceCount:    len(omittedDetails),
		RenderedReferenceDetails: renderedDetails,
		OmittedReferenceDetails:  omittedDetails,
	}, nil
}
