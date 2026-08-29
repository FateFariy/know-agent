package vo

import (
	"encoding/json"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
)

// ConversationSummary 会话摘要
type ConversationSummary struct {
	Summary          string   `json:"summary"`           // 摘要
	ConversationGoal string   `json:"conversation_goal"` // 会话目标
	StableFacts      []string `json:"stable_facts"`      // 稳定事实
	UserPreferences  []string `json:"user_preferences"`  // 用户偏好
	ResolvedPoints   []string `json:"resolved_points"`   // 解决的点
	PendingQuestions []string `json:"pending_questions"` // 待解决的问题
	RetrievalHints   []string `json:"retrieval_hints"`   // 检索提示
}

// CopySummary 复制摘要
func (s *ConversationSummary) CopySummary() *ConversationSummary {
	if s == nil {
		return &ConversationSummary{}
	}
	return &ConversationSummary{
		Summary:          s.Summary,
		ConversationGoal: s.ConversationGoal,
		StableFacts:      append([]string{}, s.StableFacts...),
		UserPreferences:  append([]string{}, s.UserPreferences...),
		ResolvedPoints:   append([]string{}, s.ResolvedPoints...),
		PendingQuestions: append([]string{}, s.PendingQuestions...),
		RetrievalHints:   append([]string{}, s.RetrievalHints...),
	}
}

func (s *ConversationSummary) Normalize(maxChars, maxGoalLength, maxItemLength, maxSectionItems int) {
	if s == nil {
		return
	}
	s.ConversationGoal = utils.ClipTail(strutil.Trim(s.ConversationGoal), maxGoalLength)
	s.StableFacts = s.deduplicateAndLimit(s.StableFacts, maxItemLength, maxSectionItems)
	s.UserPreferences = s.deduplicateAndLimit(s.UserPreferences, maxItemLength, maxSectionItems)
	s.ResolvedPoints = s.deduplicateAndLimit(s.ResolvedPoints, maxItemLength, maxSectionItems)
	s.PendingQuestions = s.deduplicateAndLimit(s.PendingQuestions, maxItemLength, maxSectionItems)
	s.RetrievalHints = s.deduplicateAndLimit(s.RetrievalHints, maxItemLength, maxSectionItems)
	s.Summary = utils.ClipTail(strutil.Trim(s.Summary), maxChars)
	if strutil.IsBlank(s.Summary) {
		s.Summary = s.synthesizeSummaryFromSections(maxItemLength)
	}
}

// BuildLongText 构建用于持久化的长文本摘要（包含各结构化字段）
func (s *ConversationSummary) BuildLongText(maxChars, maxGoalLength, maxItemLength, maxSectionItems int) string {
	if s == nil {
		return ""
	}
	s.Normalize(maxChars, maxGoalLength, maxItemLength, maxSectionItems)
	var builder strings.Builder

	appendSection := func(title, content string) {
		if strutil.IsBlank(content) {
			return
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString("【")
		builder.WriteString(title)
		builder.WriteString("】\n")
		builder.WriteString(strutil.Trim(content))
		builder.WriteString("\n")
	}
	appendBulletSection := func(title string, values []string) {
		if len(values) == 0 {
			return
		}
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString("【")
		builder.WriteString(title)
		builder.WriteString("】\n")
		for _, v := range values {
			builder.WriteString("- ")
			builder.WriteString(v)
			builder.WriteString("\n")
		}
	}

	appendSection("长期会话摘要", s.Summary)
	appendSection("会话目标", s.ConversationGoal)
	appendBulletSection("已确认事实", s.StableFacts)
	appendBulletSection("用户偏好与约束", s.UserPreferences)
	appendBulletSection("已解决问题", s.ResolvedPoints)
	appendBulletSection("待跟进问题", s.PendingQuestions)
	appendBulletSection("检索提示", s.RetrievalHints)

	return utils.ClipTail(strutil.Trim(builder.String()), maxChars)
}

// synthesizeSummaryFromSections 从各部分合成摘要文本
func (s *ConversationSummary) synthesizeSummaryFromSections(maxItemLength int) string {
	var parts []string
	if strutil.IsNotBlank(s.ConversationGoal) {
		parts = append(parts, "目标："+utils.ClipTail(s.ConversationGoal, maxItemLength))
	}
	if len(s.StableFacts) > 0 {
		parts = append(parts, "事实："+strings.Join(s.StableFacts, "；"))
	}
	if len(s.PendingQuestions) > 0 {
		parts = append(parts, "待跟进："+strings.Join(s.PendingQuestions, "；"))
	}
	return strings.Join(parts, "；")
}

// deduplicateAndLimit 去重并限制切片长度和单个元素长度
func (s *ConversationSummary) deduplicateAndLimit(values []string, maxItemLength, maxSectionItems int) []string {
	return utils.FilterMapUniqueLimit(values, maxSectionItems, func(item string) (string, string, bool) {
		tail := utils.ClipTail(strutil.Trim(item), maxItemLength)
		return tail, tail, strutil.IsNotBlank(tail)
	})
}

// Marshal 序列化为 JSON 字符串
func (s *ConversationSummary) Marshal() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "{}", err
	}
	return string(data), nil
}

func (s *ConversationSummary) Unmarshal(raw string) error {
	if err := utils.Unmarshal(raw, s); err != nil {
		logx.Errorf("反序列化会话长期摘要 JSON 失败: %s, err=%v", raw, err)
		return err
	}

	return nil
}
