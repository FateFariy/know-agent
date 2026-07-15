package rewrite

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

var (
	numberedMultiQuestionPattern = regexp.MustCompile(`(^|\s)(\d+[)\.、]|[A-Za-z][)])`)
	multiLinePattern             = regexp.MustCompile(`\n+`)
)

// QueryRewriteLogicImpl 问题改写逻辑实现
type QueryRewriteLogicImpl struct {
	chatModel       *logic.ChatModelImpl[*schema.AgenticMessage]
	promptTemplate  logic.PromptTemplateLogic
	maxSubQuestions int
	options         []model.Option
}

// NewQueryRewriteLogicImpl 创建问题改写逻辑实例
func NewQueryRewriteLogicImpl(svcCtx *svc.ServiceContext, chatModel *logic.ChatModelImpl[*schema.AgenticMessage],
	promptTemplate logic.PromptTemplateLogic) *QueryRewriteLogicImpl {
	return &QueryRewriteLogicImpl{
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
func (q *QueryRewriteLogicImpl) Rewrite(ctx context.Context, question, historySummary string, trace *vo.ConversationTrace) (*vo.QuestionRewriteResult, error) {
	// 空问题直接返回空结果
	question = strutil.Trim(question)
	if strutil.IsBlank(question) {
		return vo.NewQuestionRewriteResult("", []string{}), nil
	}

	// 预计算兜底结果，用于快速返回
	fallback := q.fallback(question)

	// 判断是否需要LLM改写（短问题或有明确多问题特征时跳过）
	if !q.needsRewrite(question, historySummary) {
		logx.Infof("RAG 改写跳过: question='%s', rewritten='%s', subQuestions=%v",
			question, fallback.RewrittenQuestion, fallback.SubQuestions)
		return fallback, nil
	}

	// 构建提示词变量
	templateVars := map[string]any{
		"history":  utils.BlankToDefault(historySummary, "无历史上下文"),
		"question": question,
	}

	// 渲染提示词
	promptText, err := q.promptTemplate.Render(prompt.ChatQueryRewrite, templateVars)
	if err != nil {
		Warnf("RAG 改写失败，回退到规则改写: question='%s', err=%v", question, err)
		return fallback, nil
	}

	// 调用LLM生成改写结果
	raw, err := q.chatModel.GenerateWithTrace(ctx, vo.ChatStageRewrite, "", promptText, trace, q.options...)
	if err != nil {
		Warnf("RAG 改写失败，回退到规则改写: question='%s', err=%v", question, err)
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
	result := q.normalizeRewriteResult(question, payload)
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
func (q *QueryRewriteLogicImpl) fallback(normalizedQuestion string) *vo.QuestionRewriteResult {
	if q.looksLikeExplicitMultiQuestion(normalizedQuestion) {
		return vo.NewQuestionRewriteResult(normalizedQuestion, q.ruleBasedSplit(normalizedQuestion))
	}
	return vo.NewQuestionRewriteResult(normalizedQuestion, []string{normalizedQuestion})
}

// needsRewrite 是否需要改写
func (q *QueryRewriteLogicImpl) needsRewrite(question, historySummary string) bool {
	if strutil.IsBlank(historySummary) {
		return utf8.RuneCountInString(question) < 8 || q.looksLikeExplicitMultiQuestion(question)
	}
	return utf8.RuneCountInString(question) < 18 || q.looksLikeExplicitMultiQuestion(question)
}

// looksLikeExplicitMultiQuestion 是否显式多问题
func (q *QueryRewriteLogicImpl) looksLikeExplicitMultiQuestion(question string) bool {
	normalized := strutil.Trim(question)
	if strutil.IsBlank(normalized) {
		return false
	}

	questionMarkCount := strings.Count(normalized, "?") + strings.Count(normalized, "？")
	if questionMarkCount >= 2 {
		return true
	}

	if strings.Contains(normalized, "；") || strings.Contains(normalized, ";") {
		return true
	}

	if multiLinePattern.MatchString(normalized) {
		nonBlankLines := slices.DeleteFunc(strings.Split(normalized, "\n"), func(item string) bool {
			return strutil.IsBlank(item)
		})
		if len(nonBlankLines) >= 2 {
			return true
		}
	}

	if numberedMultiQuestionPattern.MatchString(normalized) {
		return true
	}

	return strings.Contains(normalized, "分别")
}

// normalizeRewriteResult 规范化改写结果
func (q *QueryRewriteLogicImpl) normalizeRewriteResult(originalQuestion string, parsed *parsedRewritePayload) *vo.QuestionRewriteResult {
	if parsed == nil {
		return nil
	}

	// 确定改写后的问题（优先使用LLM改写结果，否则回退到原问题）
	rewrite := strutil.Trim(utils.BlankToDefault(parsed.Rewrite, originalQuestion))
	if strutil.IsBlank(rewrite) {
		return nil
	}

	// 处理子问题列表：去空格、过滤空白、去重、限制数量
	subQuestions := stream.FromSlice(parsed.SubQuestions).
		Map(func(item string) string { return strutil.Trim(item) }).
		Filter(func(item string) bool { return strutil.IsNotBlank(item) }).
		Distinct().Limit(q.maxSubQuestions).ToSlice()

	// 判断是否为显式多问题及是否需要拆分
	explicitMultiQuestion := q.looksLikeExplicitMultiQuestion(originalQuestion)

	// 拆分决策：仅当显式多问题且LLM明确要求拆分时才保留子问题
	if !parsed.ShouldSplit || !explicitMultiQuestion {
		// 不满足拆分条件，收敛为单一改写问题
		if parsed.ShouldSplit && len(subQuestions) > 1 {
			logx.Infof("RAG 改写子问题收敛: question='%s', rewrite='%s', originalSubQuestionCount=%d, reason='llm-split-rejected-by-conservative-structure-check'",
				originalQuestion, rewrite, len(subQuestions))
		}
		subQuestions = []string{rewrite}
	} else if len(subQuestions) == 0 {
		// 需要拆分但LLM未提供子问题，回退到规则拆分
		fallbackSplit := q.ruleBasedSplit(originalQuestion)
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

// ruleBasedSplit 基于规则(?？；;\n)进行拆分
func (q *QueryRewriteLogicImpl) ruleBasedSplit(question string) []string {
	splitPattern := regexp.MustCompile(`[?？；;\n]+`)
	parts := splitPattern.Split(question, -1)
	result := stream.FromSlice(parts).
		Map(func(item string) string { return strutil.Trim(item) }).
		Filter(func(item string) bool { return strutil.IsNotBlank(item) }).
		Distinct().Limit(q.maxSubQuestions).ToSlice()

	if len(result) == 0 {
		return []string{question}
	}

	return result
}

func Warnf(format string, args ...any) {
	logx.Alert(fmt.Sprintf(format, args...))
}

type parsedRewritePayload struct {
	Rewrite      string   `json:"rewrite"`
	ShouldSplit  bool     `json:"should_split"`
	SubQuestions []string `json:"sub_questions"`
}
