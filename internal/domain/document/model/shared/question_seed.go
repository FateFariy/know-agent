package shared

import (
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

// QuestionSeed 表示构建假设性问答所需的输入上下文
type QuestionSeed struct {
	title     string
	chunkType string
	keywords  []string
}

func NewQuestionSeed(title, chunkType string, keywords []string) *QuestionSeed {
	return &QuestionSeed{
		title:     strings.TrimSpace(title),
		chunkType: chunkType,
		keywords:  utils.FilterBlank(keywords),
	}
}

// Build 执行问题构建，返回问题列表
func (q *QuestionSeed) Build() []string {
	seen := make(map[string]bool, 4)
	questions := make([]string, 0, 4)
	add := func(question string) {
		if !seen[question] && len(questions) < 4 {
			seen[question] = true
			questions = append(questions, question)
		}
	}

	// 确定主题词：优先使用标题，否则取关键词列表的第一个
	topic := strings.TrimSpace(q.title)
	if topic == "" {
		if len(q.keywords) > 0 {
			topic = q.keywords[0]
		}
	}

	// 基于主题生成通用问题
	if topic != "" {
		add("关于" + topic + "的核心内容是什么？")
		add(topic + "有哪些要求或注意事项？")
	}

	// 基于块类型生成特定问题
	upperType := strings.ToUpper(q.chunkType)
	if upperType == "TABLE" {
		add("这个表格说明了什么？")
	}
	if upperType == "IMAGE" || upperType == "FIGURE" {
		add("这张图片说明了什么？")
	}

	return questions
}
