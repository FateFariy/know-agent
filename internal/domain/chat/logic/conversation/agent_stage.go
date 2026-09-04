package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// AgentRunner 执行一次 deep agent 会话的领域入口
type AgentRunner interface {
	Run(ctx context.Context, convCtx *Context) error
}

// AgentStage Agentic RAG 编排阶段
type AgentStage struct {
	runner AgentRunner
}

var _ Stage = (*AgentStage)(nil)

// NewAgentStage 创建 AgentStage
func NewAgentStage(runner AgentRunner) *AgentStage {
	return &AgentStage{runner: runner}
}

// Name 阶段名称
func (s *AgentStage) Name() string { return enum.ConversationTraceStageAgent.Name }

// Order 阶段顺序
func (s *AgentStage) Order() int { return enum.ConversationTraceStageAgent.Order }

// Execute 执行 AgentStage。
//
// 缓存策略：
//   - 命中「复用答案」策略且存在可复用草稿：直接发布缓存答案，不再运行 agent；
//   - 其余情况（含「仅复用检索」）：运行 deep agent 完成决策/检索/生成。
func (s *AgentStage) Execute(ctx context.Context, convCtx *Context) error {
	if convCtx.cache.IsCacheHit() &&
		convCtx.cache.ReuseStrategy() == enum.ReuseAnswerAndRetrieval &&
		utils.IsNotBlank(convCtx.cache.CacheEntry().AnswerDraft) {
		return convCtx.PublishText(convCtx.cache.CacheEntry().AnswerDraft)
	}
	return s.runner.Run(ctx, convCtx)
}
