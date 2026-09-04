package middleware

import (
	"context"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation/tool"
	"github.com/swiftbit/know-agent/internal/svc"
)

// CacheInjectMiddleware 缓存复用注入中间件（语义缓存「仅复用检索」场景）。
//
// 语义缓存命中「仅复用检索」策略（ReuseRetrievalOnly）时，答案仍需重新生成，
// 因此 AgentStage 会照常启动 deep agent；本中间件在 agent 启动前（BeforeAgent）
// 把缓存回填的检索结果渲染成证据注入 system 指令，并提示不要重复检索，使 agent
// 直接基于缓存证据作答：
//   - 真正复用缓存检索结果，避免再次调用 search_knowledge_base 工具重查知识库；
//   - 引用编号与前端展示的参考文献同源（均来自同一 RetrievalResult），编号可对齐。
//
// 命中「复用答案」策略（ReuseAnswerAndRetrieval）时 AgentStage 已短路发布草稿、
// agent 不会启动，本中间件不生效；未命中时返回原指令，无副作用。
// 须注册在 MemoryLoadMiddleware / KnowledgeRouteMiddleware 之后，保证指令追加语义。
type CacheInjectMiddleware struct {
	BaseAgentMiddleware
	renderer *tool.EvidenceRenderer
}

// NewCacheInjectMiddleware 创建缓存复用注入中间件
func NewCacheInjectMiddleware(svcCtx *svc.ServiceContext) *CacheInjectMiddleware {
	return &CacheInjectMiddleware{renderer: tool.NewEvidenceRenderer(svcCtx)}
}

// Name 中间件名称
func (m *CacheInjectMiddleware) Name() string { return "cache-inject" }

// BeforeAgent agent 启动前把缓存检索证据注入 system 指令
func (m *CacheInjectMiddleware) BeforeAgent(_ context.Context, convCtx *conversation.Context, input *BeforeAgentInput) (*BeforeAgentOutput, error) {
	if input == nil {
		return nil, nil
	}
	instruction := input.Instruction
	if hint := m.buildEvidenceHint(convCtx); hint != "" {
		instruction = appendInstruction(instruction, hint)
	}
	return &BeforeAgentOutput{Instruction: instruction}, nil
}

// buildEvidenceHint 命中「仅复用检索」时渲染缓存检索结果并组装提示指令；无可用证据时返回空串
func (m *CacheInjectMiddleware) buildEvidenceHint(convCtx *conversation.Context) string {
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

// appendInstruction 将提示段落按换行语义追加到指令末尾
func appendInstruction(instruction, hint string) string {
	if instruction != "" && !strings.HasSuffix(instruction, "\n") {
		instruction += "\n"
	}
	return instruction + hint
}
