package support

import (
	"regexp"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
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

// RequiresCurrentDateAnchoring 判断是否需要当前日期锚定
func RequiresCurrentDateAnchoring(query string) bool {
	query = strings.ToLower(strutil.Trim(query))
	if query == "" {
		return false
	}
	if hasHistoricalIntent(query) && !hasRelativeTimeReference(query) && !hasCalendarQuestion(query) {
		return false
	}
	return hasRelativeTimeReference(query) || hasRealTimeInfo(query) || hasCalendarQuestion(query)
}

// RequiresRealTimeSearch 判断是否需要实时搜索
func RequiresRealTimeSearch(query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return false
	}
	if hasHistoricalIntent(query) || containsExplicitDate(query) || hasCalendarQuestion(query) {
		return false
	}
	return hasRealTimeInfo(query) || containsAny(query, "最新", "实时", "当前", "现在", "目前", "刚刚")
}

// BuildEffectiveSearchQuery 构建有效搜索查询
func BuildEffectiveSearchQuery(query, currentDate string) string {
	query = strings.TrimSpace(query)
	if query == "" || currentDate == "" {
		return query
	}
	if !RequiresCurrentDateAnchoring(query) {
		return query
	}
	if containsExplicitDate(query) || strings.Contains(query, currentDate) || hasHistoricalIntent(query) {
		return query
	}
	return query + " " + currentDate + " " + deriveTemporalHint(query)
}

// containsExplicitDate 判断是否包含显式日期
func containsExplicitDate(query string) bool {
	return explicitDatePattern.MatchString(query)
}

// hasRelativeTimeReference 判断是否有相对时间引用
func hasRelativeTimeReference(query string) bool {
	return containsAny(query, relativeTimeKeywords...)
}

// hasCalendarQuestion 判断是否为日历问题
func hasCalendarQuestion(query string) bool {
	return containsAny(query, calendarKeywords...)
}

// hasRealTimeInfo 判断是否为实时信息
func hasRealTimeInfo(query string) bool {
	return containsAny(query, realTimeInformationKeywords...)
}

// hasHistoricalIntent 判断是否有历史意图
func hasHistoricalIntent(query string) bool {
	return containsAny(query, historicalHints...)
}

// deriveTemporalHint 推导时间提示
func deriveTemporalHint(query string) string {
	if containsAny(query, "明天", "明日") {
		return "明天"
	}
	if containsAny(query, "昨天", "昨日", "前天") {
		return "昨天"
	}
	if containsAny(query, "本周", "这周") {
		return "本周"
	}
	if containsAny(query, "本月", "这个月") {
		return "本月"
	}
	if containsAny(query, "今年", "本年度", "本季度") {
		return "今年"
	}
	if containsAny(query, "最新", "实时", "当前", "现在", "目前", "刚刚") {
		return "最新"
	}
	return "今天"
}

// containsAny 检测字符串是否包含任意元素
func containsAny(query string, candidates ...string) bool {
	return strutil.ContainsAny(query, candidates)
}
