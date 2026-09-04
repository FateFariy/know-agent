package recommend

import (
	"context"
	"strings"

	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	maxRecommendations = 3
)

// QuestionRecommendImpl 追问推荐实现
type QuestionRecommendImpl struct {
	promptTemplate      adapter.PromptRenderer
	chatModel           model.ChatModel
	historyPreviewTurns int
}

func NewQuestionRecommendImpl(svcCtx *svc.ServiceContext, promptTemplate adapter.PromptRenderer, chatModel model.ChatModel) *QuestionRecommendImpl {
	return &QuestionRecommendImpl{
		chatModel:           chatModel,
		promptTemplate:      promptTemplate,
		historyPreviewTurns: svcCtx.Config.Chat.Recommendation.HistoryPreviewTurns,
	}
}

// Generate 生成推荐追问
func (r *QuestionRecommendImpl) Generate(ctx context.Context, question, answer string, recentExchanges []*entity.ChatExchange) ([]string, error) {
	if strutil.IsBlank(answer) {
		return []string{}, nil
	}
	result, err := r.generateRecommendations(ctx, question, answer, recentExchanges)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// generateRecommendations 生成推荐追问
func (r *QuestionRecommendImpl) generateRecommendations(ctx context.Context, question, answer string, recentExchanges []*entity.ChatExchange) ([]string, error) {
	// 构建最近上下文
	recentContext := r.buildRecentContext(recentExchanges)

	// 渲染提示词模板
	userPrompt, err := r.promptTemplate.Render(enum.RecommendationUser, map[string]any{
		"recentContext": recentContext,
		"question":      question,
		"answer":        answer,
	})
	if err != nil {
		return nil, err
	}

	// 调用LLM生成推荐
	content, err := r.chatModel.GenerateWithTrace(ctx, enum.ChatStageRecommend, "", userPrompt, model.WithThink(false))
	if strutil.IsBlank(content) {
		return nil, err
	}

	// 解析JSON数组
	var result []string
	if err = utils.Unmarshal(content, &result); err != nil {
		logx.Warnf("解析推荐问题失败: content=%s, err=%v", content, err)
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
func (r *QuestionRecommendImpl) buildRecentContext(recentExchanges []*entity.ChatExchange) string {
	if len(recentExchanges) == 0 {
		return ""
	}

	var sb strings.Builder
	historyTurns := max(r.historyPreviewTurns, 3)
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
