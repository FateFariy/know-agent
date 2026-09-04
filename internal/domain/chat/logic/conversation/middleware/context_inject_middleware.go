package middleware

import (
	"context"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation/tool"
	"github.com/swiftbit/know-agent/internal/svc"
)

// ContextInjectMiddleware 动态上下文注入中间件：在 Agent 启动前把作答所需的预置上下文
// 统一追加进 system 指令，集中管理指令注入逻辑：
//
//  1. 会话记忆上下文：当前日期、时效/实时检索规则判断、历史摘要、上一轮证据锚点（追问承接）；
//  2. 语义缓存「仅复用检索」（ReuseRetrievalOnly）命中的缓存检索证据，提示 agent
//     直接基于缓存证据作答、无需重复调用检索工具。
//
// 记忆装载与执行计划组装已由链上的 MemoryLoadStage 在查询改写之前完成，本中间件不再做
// 数据装载（避免重复装载覆盖查询改写已写回的 RewriteQuestion），仅负责指令注入。
// 命中「复用答案」策略（ReuseAnswerAndRetrieval）时 AgentStage 已短路发布草稿、
// agent 不会启动，缓存证据注入不生效；未命中缓存时按原指令透传，无副作用。
type ContextInjectMiddleware struct {
	BaseAgentMiddleware
	renderer *tool.EvidenceRenderer
}

// NewContextInjectMiddleware 创建动态上下文注入中间件
func NewContextInjectMiddleware(svcCtx *svc.ServiceContext) *ContextInjectMiddleware {
	return &ContextInjectMiddleware{renderer: tool.NewEvidenceRenderer(svcCtx)}
}

// Name 中间件名称
func (m *ContextInjectMiddleware) Name() string { return "context-inject" }

// BeforeAgent agent 启动前把动态上下文逐段注入 system 指令
func (m *ContextInjectMiddleware) BeforeAgent(_ context.Context, convCtx *conversation.Context, input *BeforeAgentInput) (*BeforeAgentOutput, error) {
	if input == nil {
		return nil, nil
	}
	instruction := input.Instruction
	for _, paragraph := range m.buildInjectionParagraphs(convCtx) {
		if instruction != "" && !strings.HasSuffix(instruction, "\n") {
			instruction += "\n"
		}
		instruction = instruction + paragraph
	}
	return &BeforeAgentOutput{Instruction: instruction}, nil
}

// buildInjectionParagraphs 组装待注入的指令段落（每段语义独立、追加时以换行分隔）
func (m *ContextInjectMiddleware) buildInjectionParagraphs(convCtx *conversation.Context) []string {
	if convCtx == nil {
		return nil
	}
	var paragraphs []string
	if memoryHints := m.buildMemoryHints(convCtx); len(memoryHints) > 0 {
		paragraphs = append(paragraphs, strings.Join(memoryHints, "\n"))
	}
	if evidenceHint := m.buildCacheEvidenceHint(convCtx); evidenceHint != "" {
		paragraphs = append(paragraphs, evidenceHint)
	}
	return paragraphs
}

// buildMemoryHints 组装会话记忆相关提示
func (m *ContextInjectMiddleware) buildMemoryHints(convCtx *conversation.Context) []string {
	execPlan := convCtx.ExecutionPlan.Load()
	if execPlan == nil {
		return nil
	}
	question := utils.Trim(convCtx.Question)
	if question == "" {
		return nil
	}

	analyzer := NewQueryAnalyzer(question)
	execPlan.RequiresRealTimeSearch = analyzer.RequiresRealTimeSearch()
	execPlan.RequiresCurrentDateAnchoring = analyzer.RequiresCurrentDateAnchoring()

	var hints []string
	addHint := func(condition bool, content string) {
		if condition {
			hints = append(hints, content)
		}
	}
	addHint(utils.IsNotBlank(convCtx.CurrentDateText), "当前日期："+convCtx.CurrentDateText+"。")
	addHint(execPlan.RequiresCurrentDateAnchoring, "当前问题包含相对时间或强时效表达（如“今天、明天、现在、最新、本周、本月”等），涉及日期必须以此为准，不要用检索结果中的旧日期替代今天。")
	addHint(execPlan.RequiresRealTimeSearch, "当前问题需要核实最新信息，检索证据不足或日期滞后时请明确说明不确定性，不要编造。")
	addHint(utils.IsNotBlank(execPlan.HistorySummary), "相关会话背景：\n"+execPlan.HistorySummary)
	addHint(len(execPlan.RecentEvidenceAnchors) > 0, execPlan.RecentEvidenceAnchors.RenderFollowUpHint())

	return hints
}

// buildCacheEvidenceHint 命中「仅复用检索」时渲染缓存检索结果并组装提示；无可用证据时返回空串
func (m *ContextInjectMiddleware) buildCacheEvidenceHint(convCtx *conversation.Context) string {
	result := convCtx.ReusableCacheEvidence()
	if result == nil {
		return ""
	}
	evidence := utils.Trim(result.EvidenceText)
	if evidence == "" {
		evidence = utils.Trim(m.renderer.RenderEvidence(result))
	}
	if evidence == "" {
		return ""
	}
	return "已检索到相关知识库结果。如果检索结果可以直接回答，请直接使用下述内容回答，无需再调用重复检索:\n\n" + evidence
}
