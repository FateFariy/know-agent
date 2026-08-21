package rewrite

import (
	"context"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// QueryRewriteImpl 问题改写逻辑实现
type QueryRewriteImpl struct {
	chatModel       model.ChatModel
	promptTemplate  adapter.PromptRenderer
	maxSubQuestions int
	options         []model.Option
}

// NewQueryRewriteImpl 创建问题改写逻辑实例
func NewQueryRewriteImpl(svcCtx *svc.ServiceContext, chatModel model.ChatModel,
	promptTemplate adapter.PromptRenderer) *QueryRewriteImpl {
	return &QueryRewriteImpl{
		chatModel:       chatModel,
		promptTemplate:  promptTemplate,
		maxSubQuestions: svcCtx.Config.Chat.Rewrite.MaxSubQuestions,
		options: []model.Option{
			model.WithTemperature(svcCtx.Config.Chat.Rewrite.Temperature),
			model.WithTopP(svcCtx.Config.Chat.Rewrite.TopP),
		},
	}
}

// Rewrite 改写问题（结合历史上下文）
// 流程：空问题直接返回 -> 判断是否需要改写 -> 不需要则规则改写 -> 需要则LLM改写 -> 规范化结果
func (q *QueryRewriteImpl) Rewrite(ctx context.Context, question, historySummary string) (*vo.QuestionRewriteResult, error) {
	oq := NewOriginalQuestion(question, historySummary)

	// 空问题直接返回空结果
	if oq.IsBlank() {
		return vo.NewQuestionRewriteResult("", []string{}), nil
	}

	// 预计算兜底结果，用于快速返回
	fallback := q.fallback(oq)

	// 判断是否需要LLM改写（短问题或有明确多问题特征时）
	if !oq.NeedsRewrite(8, 18) {
		logx.Infof("RAG 改写跳过: question='%s', rewritten='%s', subQuestions=%v",
			oq.Question(), fallback.RewrittenQuestion, fallback.SubQuestions)
		return fallback, nil
	}

	// 构建提示词变量
	templateVars := map[string]any{
		"history":  utils.BlankToDefault(historySummary, "无历史上下文"),
		"question": question,
	}

	// 渲染提示词
	promptText, err := q.promptTemplate.Render(enum.ChatQueryRewrite, templateVars)
	if err != nil {
		logx.Warnf("RAG 改写失败，回退到规则改写: question='%s', err=%v", question, err)
		return fallback, nil
	}

	// 调用LLM生成改写结果
	raw, err := q.chatModel.GenerateWithTrace(ctx, enum.ChatStageRewrite, "", promptText, q.options...)
	if err != nil {
		logx.Warnf("RAG 改写失败，回退到规则改写: question='%s', err=%v", question, err)
		return fallback, nil
	}

	// 解析LLM输出
	payload := &parsedRewritePayload{}
	if err = utils.Unmarshal(raw, payload); err != nil {
		// LLM结果无效，回退到规则改写
		logx.Errorf("RAG 改写结果不可用，回退到规则改写: question='%s', raw='%s'", question, strutil.Trim(raw))
		return fallback, nil
	}

	// 规范化改写结果
	result := q.normalizeRewriteResult(oq, payload)
	if result != nil && strutil.IsNotBlank(result.RewrittenQuestion) {
		result.RawModelOutput = raw
		logx.Infof("RAG 改写完成: question='%s', rewritten='%s', subQuestions=%v",
			question, result.RewrittenQuestion, result.SubQuestions)
		return result, nil
	}

	// LLM结果无效，回退到规则改写
	logx.Errorf("RAG 改写结果不可用，回退到规则改写: question='%s', raw='%s'", question, strutil.Trim(raw))

	return fallback, nil
}

// fallback 兜底改写
func (q *QueryRewriteImpl) fallback(oq *OriginalQuestion) *vo.QuestionRewriteResult {
	if oq.IsExplicitMultiQuestion() {
		return vo.NewQuestionRewriteResult(oq.Question(), oq.SplitByRules(q.maxSubQuestions))
	}
	return vo.NewQuestionRewriteResult(oq.Question(), []string{oq.Question()})
}

// normalizeRewriteResult 规范化 LLM 改写输出，生成最终的 QuestionRewriteResult
func (q *QueryRewriteImpl) normalizeRewriteResult(oq *OriginalQuestion, parsed *parsedRewritePayload) *vo.QuestionRewriteResult {
	if parsed == nil {
		return nil
	}

	// 确定改写后的问题（优先使用LLM改写结果，否则回退到原问题）
	rewrite := strutil.Trim(utils.BlankToDefault(parsed.Rewrite, oq.Question()))
	if rewrite == "" {
		return nil
	}

	// 处理子问题列表：去空格、过滤空白、去重、限制数量
	subQuestions := utils.FilterMapUniqueLimit(parsed.SubQuestions, q.maxSubQuestions, func(item string) (string, string, bool) {
		trim := strings.TrimSpace(item)
		return trim, trim, trim != ""
	})

	// 判断是否为显式多问题及是否需要拆分
	explicitMultiQuestion := oq.IsExplicitMultiQuestion()

	// 拆分决策：仅当显式多问题且LLM明确要求拆分时才保留子问题
	if !parsed.ShouldSplit || !explicitMultiQuestion {
		// 不满足拆分条件，收敛为单一改写问题
		if parsed.ShouldSplit && len(subQuestions) > 1 {
			logx.Infof("RAG 改写子问题收敛: question='%s', rewrite='%s', originalSubQuestionCount=%d, reason='llm-split-rejected-by-conservative-structure-check'",
				oq, rewrite, len(subQuestions))
		}
		subQuestions = []string{rewrite}
	} else if len(subQuestions) == 0 {
		// 需要拆分但LLM未提供子问题，回退到规则拆分
		fallbackSplit := oq.SplitByRules(q.maxSubQuestions)
		if len(fallbackSplit) > 1 {
			subQuestions = fallbackSplit
		} else {
			subQuestions = []string{rewrite}
		}
	}

	// 子问题与改写问题不一致时的修正
	if len(subQuestions) == 1 && subQuestions[0] != rewrite && !parsed.ShouldSplit {
		subQuestions = []string{rewrite}
	}

	return vo.NewQuestionRewriteResult(rewrite, subQuestions)
}

type parsedRewritePayload struct {
	Rewrite      string   `json:"rewrite"`
	ShouldSplit  bool     `json:"should_split"`
	SubQuestions []string `json:"sub_questions"`
}
