package strategy

import (
	"context"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// MemoryStrategy 记忆策略接口
type MemoryStrategy interface {
	// LoadMemoryContext 加载会话记忆上下文
	LoadMemoryContext(ctx context.Context, conversationId string, trace *vo.ConversationTrace) (*vo.MemoryContext, error)

	// GetStrategyType 获取记忆策略类型
	GetStrategyType() string
}

// baseMemoryStrategy 记忆策略基类（封装公共渲染和裁剪方法）
type baseMemoryStrategy struct {
}

const (
	maxQuestionLength = 160
	maxAnswerLength   = 320
)

func (b *baseMemoryStrategy) LoadMemoryContext(ctx context.Context, conversationId string, trace *vo.ConversationTrace) (*vo.MemoryContext, error) {
	panic("implement me")
}

func (b *baseMemoryStrategy) GetStrategyType() string {
	panic("implement me")
}

// renderRecentTranscript 渲染最近对话记录
func (b *baseMemoryStrategy) renderRecentTranscript(exchanges []*entity.ChatExchange, keepRecentTurns, maxChars int) string {
	// 过滤可渲染的对话（非进行中且有问答内容）
	renderable := slice.Filter(exchanges, func(i int, item *entity.ChatExchange) bool {
		return item != nil && item.TurnStatus != vo.ChatTurnStatusRunning && (strutil.IsNotBlank(item.Question) || strutil.IsNotBlank(item.Answer))
	})

	if len(renderable) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("【最近对话原文】\n")
	for i := 0; i < len(renderable) && i < keepRecentTurns; i++ {
		exchange := renderable[i]
		if strutil.IsNotBlank(exchange.Question) {
			builder.WriteString("用户：")
			builder.WriteString(utils.ClipTail(exchange.Question, maxQuestionLength))
			builder.WriteString("\n")
		}
		if exchange.TurnStatus == vo.ChatTurnStatusCompleted && strutil.IsNotBlank(exchange.Answer) {
			builder.WriteString("助手：")
			builder.WriteString(utils.ClipTail(exchange.Answer, maxAnswerLength))
			builder.WriteString("\n")
		}
	}

	return utils.ClipHead(builder.String(), maxChars)
}

// renderRecentQuestionTranscript 渲染最近问题记录
func (b *baseMemoryStrategy) renderRecentQuestionTranscript(exchanges []*entity.ChatExchange, keepRecentTurns, maxChars int) string {
	// 过滤有问题的对话（非进行中且有提问）
	renderable := slice.Filter(exchanges, func(i int, item *entity.ChatExchange) bool {
		return item != nil && item.TurnStatus != vo.ChatTurnStatusRunning && strutil.IsNotBlank(item.Question)
	})

	if len(renderable) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("【最近相关对话】\n")
	for i := 0; i < len(renderable) && i < keepRecentTurns; i++ {
		exchange := renderable[i]
		builder.WriteString("用户：")
		builder.WriteString(utils.ClipTail(exchange.Question, maxQuestionLength))
		builder.WriteString("\n")
	}

	return utils.ClipHead(builder.String(), maxChars)
}
