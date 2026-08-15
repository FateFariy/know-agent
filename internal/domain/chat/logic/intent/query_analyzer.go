package intent

import (
	"regexp"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

var (
	// 显式日期模式
	explicitDatePattern = regexp.MustCompile(`(\d{4}[-/.年]\d{1,2}[-/.月]\d{1,2}日?)|(\d{1,2}月\d{1,2}日)`)

	// 相对时间关键词
	relativeTimeKeywords = []string{
		"今天", "今日", "明天", "明日", "昨天", "昨日", "后天", "前天",
		"现在", "当前", "目前", "此刻", "实时", "最新", "刚刚",
		"本周", "这周", "本月", "这个月", "今年", "本年度", "本季度",
		"周几", "星期几", "几号", "日期", "几月几号",
	}

	// 实时信息关键词
	realTimeInformationKeywords = []string{
		"天气", "气温", "温度", "降雨", "下雨", "下雪", "空气质量", "aqi",
		"限号", "限行", "尾号限行",
		"汇率", "金价", "黄金价格", "银价", "油价",
		"股价", "行情", "大盘", "指数",
		"新闻", "头条", "热搜", "热榜",
		"路况", "拥堵",
		"票房", "排片",
		"航班", "班次", "列车", "高铁", "火车", "地铁运营",
		"比分", "赛果", "赛程", "比赛结果",
		"预警", "台风",
	}

	// 日历关键词
	calendarKeywords = []string{
		"周几", "星期几", "几号", "日期", "几月几号", "星期", "周",
	}

	// 历史提示关键词
	historicalHints = []string{
		"历史", "过去", "去年", "前年", "上周", "上个月", "上月", "上一周",
		"上一月", "往年", "历年", "当时", "之前", "回顾", "曾经",
	}
)

// QueryAnalyzer 封装查询字符串，提供各种分析方法
type QueryAnalyzer struct {
	query string
}

func NewQueryAnalyzer(query string) *QueryAnalyzer {
	return &QueryAnalyzer{query: utils.Trim(query)}
}

// RequiresCurrentDateAnchoring 判断是否需要当前日期锚定
func (q *QueryAnalyzer) RequiresCurrentDateAnchoring() bool {
	query := strings.ToLower(q.query)
	if query == "" {
		return false
	}
	if q.hasHistoricalIntent() && !q.hasRelativeTimeReference() && !q.hasCalendarQuestion() {
		return false
	}
	return q.hasRelativeTimeReference() || q.hasRealTimeInfo() || q.hasCalendarQuestion()
}

// RequiresRealTimeSearch 判断是否需要实时搜索
func (q *QueryAnalyzer) RequiresRealTimeSearch() bool {
	query := strings.ToLower(q.query)
	if query == "" {
		return false
	}
	if q.hasHistoricalIntent() || q.containsExplicitDate() || q.hasCalendarQuestion() {
		return false
	}
	return q.hasRealTimeInfo() || q.containsAny("最新", "实时", "当前", "现在", "目前", "刚刚")
}

// BuildEffectiveSearchQuery 构建有效搜索查询，需要传入当前日期字符串
func (q *QueryAnalyzer) BuildEffectiveSearchQuery(currentDate string) string {
	query := q.query
	if query == "" || currentDate == "" {
		return query
	}
	if !q.RequiresCurrentDateAnchoring() {
		return query
	}
	if q.containsExplicitDate() || strings.Contains(query, currentDate) || q.hasHistoricalIntent() {
		return query
	}
	return query + " " + currentDate + " " + q.deriveTemporalHint()
}

func (q *QueryAnalyzer) containsExplicitDate() bool {
	return explicitDatePattern.MatchString(q.query)
}

func (q *QueryAnalyzer) hasRelativeTimeReference() bool {
	return q.containsAny(relativeTimeKeywords...)
}

func (q *QueryAnalyzer) hasCalendarQuestion() bool {
	return q.containsAny(calendarKeywords...)
}

func (q *QueryAnalyzer) hasRealTimeInfo() bool {
	return q.containsAny(realTimeInformationKeywords...)
}

func (q *QueryAnalyzer) hasHistoricalIntent() bool {
	return q.containsAny(historicalHints...)
}

func (q *QueryAnalyzer) deriveTemporalHint() string {
	if q.containsAny("明天", "明日") {
		return "明天"
	}
	if q.containsAny("昨天", "昨日", "前天") {
		return "昨天"
	}
	if q.containsAny("本周", "这周") {
		return "本周"
	}
	if q.containsAny("本月", "这个月") {
		return "本月"
	}
	if q.containsAny("今年", "本年度", "本季度") {
		return "今年"
	}
	if q.containsAny("最新", "实时", "当前", "现在", "目前", "刚刚") {
		return "最新"
	}
	return "今天"
}

// containsAny 检测查询字符串是否包含任意候选词
func (q *QueryAnalyzer) containsAny(candidates ...string) bool {
	return utils.ContainsAny(candidates, q.query)
}
