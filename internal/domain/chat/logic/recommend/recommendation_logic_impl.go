package recommend

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	maxRecommendations = 3
)

// RecommendationLogicImpl 推荐追问业务逻辑实现
type RecommendationLogicImpl struct {
	properties     config.RecommendationConf
	promptTemplate logic.PromptTemplateLogic
	chatModel      *logic.ChatModelImpl[*schema.AgenticMessage]
}

// NewRecommendationLogicImpl 创建推荐追问逻辑实例
func NewRecommendationLogicImpl(svcCtx *svc.ServiceContext, promptTemplate logic.PromptTemplateLogic, chatModel *logic.ChatModelImpl[*schema.AgenticMessage]) *RecommendationLogicImpl {
	return &RecommendationLogicImpl{
		properties:     svcCtx.Config.Chat.Recommendation,
		promptTemplate: promptTemplate,
		chatModel:      chatModel,
	}
}

// GenerateRecommendations 生成推荐追问
func (r *RecommendationLogicImpl) GenerateRecommendations(ctx context.Context, question, answer string, recentExchanges []*entity.ChatExchange, trace *vo.ConversationTrace) []string {
	// 检查是否启用推荐且回答不为空
	if !r.properties.Enabled || strutil.IsBlank(answer) {
		return []string{}
	}

	// 使用通道处理超时
	resultChan := make(chan []string, 1)
	errChan := make(chan error, 1)
	timeoutCtx, cancelFunc := context.WithTimeout(ctx, r.properties.Timeout)
	defer close(resultChan)
	defer close(errChan)
	defer cancelFunc()

	go func() {
		result, err := r.generateRecommendations(timeoutCtx, question, answer, recentExchanges, trace)
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- result
	}()

	// 等待结果或超时
	var result []string
	select {
	case result = <-resultChan:
	case err := <-errChan:
		Warnf("生成推荐问题失败: %v", err)
	case <-timeoutCtx.Done():
		Warnf("生成推荐问题超时: %v", r.properties.Timeout)
	case <-ctx.Done():
		Warnf("生成推荐问题被取消: %v", ctx.Err())
	}
	return result
}

// generateRecommendations 生成推荐追问
func (r *RecommendationLogicImpl) generateRecommendations(ctx context.Context, question, answer string, recentExchanges []*entity.ChatExchange, trace *vo.ConversationTrace) ([]string, error) {
	// 构建最近上下文
	recentContext := r.buildRecentContext(recentExchanges)

	// 渲染提示词模板
	userPrompt, err := r.promptTemplate.Render(prompt.RecommendationUser, map[string]any{
		"recentContext": recentContext,
		"question":      question,
		"answer":        answer,
	})
	if err != nil {
		return nil, err
	}

	// 调用LLM生成推荐
	content, err := r.chatModel.GenerateWithTrace(ctx, vo.ChatStageRecommend, "", userPrompt, trace)
	if strutil.IsBlank(content) {
		return nil, err
	}

	// 解析JSON数组
	var result []string
	if err = utils.Unmarshal(content, &result); err != nil {
		Warnf("解析推荐问题失败: content=%s, err=%v", content, err)
		return nil, err
	}

	// 去重并限制数量
	result = stream.FromSlice(result).
		Filter(func(item string) bool { return strutil.IsNotBlank(item) }).
		Map(func(item string) string { return strutil.Trim(item) }).
		Distinct().Limit(maxRecommendations).ToSlice()

	return result, nil
}

// buildRecentContext 构建最近对话上下文
func (r *RecommendationLogicImpl) buildRecentContext(recentExchanges []*entity.ChatExchange) string {
	if len(recentExchanges) == 0 {
		return ""
	}

	var sb strings.Builder
	historyTurns := max(r.properties.HistoryPreviewTurns, 3)
	startIndex := max(len(recentExchanges)-historyTurns, 0)
	for i := startIndex; i < len(recentExchanges); i++ {
		exchange := recentExchanges[i]
		sb.WriteString("用户：")
		sb.WriteString(exchange.Question)
		sb.WriteString("\n")
		if strutil.IsNotBlank(exchange.Answer) {
			sb.WriteString("助手：")
			sb.WriteString(exchange.Answer)
			sb.WriteString("\n")
		}
	}

	return strutil.Trim(sb.String())
}

func Warnf(format string, v ...any) {
	logx.Alert(fmt.Sprintf(format, v...))
}
