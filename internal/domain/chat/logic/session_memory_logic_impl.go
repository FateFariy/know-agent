package logic

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// 常量定义
const (
	maxSectionItems              = 6
	maxItemLength                = 80
	maxGoalLength                = 120
	maxQuestionLength            = 160
	maxAnswerLength              = 320
	maxAnswerContextAnswerLength = 220
	businessStatusYes            = 1
)

var (
	jsonObjectPattern    = regexp.MustCompile(`\{.*\}`)
	retrievalHintPattern = regexp.MustCompile(`[a-zA-Z0-9._-]{2,}|[\p{Han}]{2,12}`)
)

// SessionMemoryLogicImpl 会话记忆逻辑实现
type SessionMemoryLogicImpl struct {
	historySummary          config.HistorySummaryConf
	repo                    adapter.ChatRepository
	refreshingMu            sync.Mutex
	refreshing              map[string]struct{}
	rewriteHistoryTurns     int
	questionHistoryMaxChars int
}

// NewSessionMemoryLogic 创建会话记忆逻辑实例
func NewSessionMemoryLogic(svcCtx *svc.ServiceContext, repo adapter.ChatRepository) *SessionMemoryLogicImpl {
	return &SessionMemoryLogicImpl{
		repo:                    repo,
		refreshing:              make(map[string]struct{}),
		historySummary:          svcCtx.Config.Memory.HistorySummary,
		rewriteHistoryTurns:     svcCtx.Config.Memory.RewriteHistoryTurns,
		questionHistoryMaxChars: svcCtx.Config.Memory.QuestionHistoryMaxChars,
	}
}

// LoadMemoryContext 加载会话记忆上下文
func (s *SessionMemoryLogicImpl) LoadMemoryContext(ctx context.Context, conversationId string) (*vo.MemoryContext, error) {
	if strings.TrimSpace(conversationId) == "" {
		return &vo.MemoryContext{}, nil
	}

	// 获取最近对话
	recentExchanges, err := s.repo.ListRecentExchanges(ctx, conversationId, s.rewriteHistoryTurns*3)
	if err != nil {
		return nil, err
	}

	// 查询现有摘要
	summaryState, err := s.repo.SelectMemorySummary(ctx, conversationId)
	if err != nil {
		return nil, err
	}

	// 如果有摘要，刷新（如果需要）
	if summaryState != nil {
		summaryState = s.refreshSummaryIfNecessary(ctx, conversationId, summaryState)
	}

	summaryPayload := s.readSummaryPayload(summaryState)

	recentTranscript := s.renderRecentTranscript(recentExchanges, s.rewriteHistoryTurns, s.historySummary.RecentTranscriptMaxChars)
	answerRecentTranscript := s.renderRecentQuestionTranscript(recentExchanges, s.rewriteHistoryTurns, s.historySummary.RecentTranscriptMaxChars)
	longTermSummary := ""
	if summaryState != nil {
		longTermSummary = strings.TrimSpace(summaryState.SummaryText)
	}

	return &vo.MemoryContext{
		AssembledHistory:         s.assembleHistory(longTermSummary, recentTranscript),
		LongTermSummary:          longTermSummary,
		RecentTranscript:         recentTranscript,
		QuestionRecentTranscript: answerRecentTranscript,
		Summary:                  summaryPayload,
		IsCompressed:             longTermSummary != "",
		CoveredExchangeId:        s.defaultLong(summaryState),
		CoveredExchangeCount:     s.safeIntValue(summaryState),
		CompressionCount:         s.safeCompressionCount(summaryState),
	}, nil
}

// RefreshConversationSummaryAsync 异步刷新会话摘要
func (s *SessionMemoryLogicImpl) RefreshConversationSummaryAsync(ctx context.Context, conversationId string) {
	if strings.TrimSpace(conversationId) == "" {
		return
	}

	s.refreshingMu.Lock()
	if _, exists := s.refreshing[conversationId]; exists {
		s.refreshingMu.Unlock()
		return
	}
	s.refreshing[conversationId] = struct{}{}
	s.refreshingMu.Unlock()

	go func() {
		defer func() {
			s.refreshingMu.Lock()
			delete(s.refreshing, conversationId)
			s.refreshingMu.Unlock()
		}()

		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("异步刷新会话摘要失败, conversationId=%s, err=%v", conversationId, r)
			}
		}()

		s.refreshSummaryIfNecessary(ctx, conversationId, nil)
	}()
}

// GetConversationSummary 获取会话摘要
func (s *SessionMemoryLogicImpl) GetConversationSummary(ctx context.Context, conversationId string) (*vo.ConversationMemorySummaryView, error) {
	if strings.TrimSpace(conversationId) == "" {
		return s.emptySummaryView(""), nil
	}

	summary, err := s.repo.SelectMemorySummary(ctx, conversationId)
	if err != nil {
		return nil, err
	}

	return s.toSummaryView(conversationId, summary), nil
}

// RebuildConversationSummary 重建会话摘要
func (s *SessionMemoryLogicImpl) RebuildConversationSummary(ctx context.Context, conversationId string) (*vo.ConversationMemorySummaryView, error) {
	if strings.TrimSpace(conversationId) == "" {
		return s.emptySummaryView(""), nil
	}

	s.refreshingMu.Lock()
	if _, exists := s.refreshing[conversationId]; exists {
		s.refreshingMu.Unlock()
		return s.GetConversationSummary(ctx, conversationId)
	}
	s.refreshing[conversationId] = struct{}{}
	s.refreshingMu.Unlock()

	defer func() {
		s.refreshingMu.Lock()
		delete(s.refreshing, conversationId)
		s.refreshingMu.Unlock()
	}()

	// 删除现有摘要
	if err := s.repo.DeleteMemorySummary(ctx, conversationId); err != nil {
		return nil, err
	}

	// 重新生成
	rebuiltState := s.refreshSummaryIfNecessary(ctx, conversationId, nil)
	return s.toSummaryView(conversationId, rebuiltState), nil
}

// DeleteConversationSummary 删除会话摘要
func (s *SessionMemoryLogicImpl) DeleteConversationSummary(ctx context.Context, conversationId string) error {
	if strings.TrimSpace(conversationId) == "" {
		return nil
	}
	return s.repo.DeleteMemorySummary(ctx, conversationId)
}

// refreshSummaryIfNecessary 刷新摘要（如果需要）
func (s *SessionMemoryLogicImpl) refreshSummaryIfNecessary(ctx context.Context, conversationId string, currentState *entity.ChatMemorySummary) *entity.ChatMemorySummary {
	coveredExchangeId := utils.Ternary(currentState == nil, 0, currentState.CoveredExchangeId)
	incrementalExchanges, err := s.repo.ListExchangesAfter(ctx, conversationId, coveredExchangeId)
	if err != nil {
		logx.Errorf("查询增量对话失败, conversationId=%s, err=%v", conversationId, err)
		return currentState
	}

	// 过滤已完成的对话
	stableExchanges := slice.Filter(incrementalExchanges, func(i int, item *entity.ChatExchange) bool {
		return item.TurnStatus == vo.ChatTurnStatusCompleted && strings.TrimSpace(item.Question) != ""
	})

	// 检查是否需要压缩
	overflowCount := len(stableExchanges) - s.historySummary.KeepRecentTurns
	if overflowCount <= 0 {
		return currentState
	}

	overflowExchanges := stableExchanges[:overflowCount]
	workingState := currentState
	compressionBatchTurns := s.historySummary.CompressionBatchTurns

	for start := 0; start < len(overflowExchanges); start += compressionBatchTurns {
		end := start + compressionBatchTurns
		if end > len(overflowExchanges) {
			end = len(overflowExchanges)
		}
		batch := overflowExchanges[start:end]

		// 使用回退合并策略
		existingPayload := s.readSummaryPayload(workingState)
		mergedPayload := s.fallbackMerge(existingPayload, batch)

		lastExchange := batch[len(batch)-1]
		workingState = s.saveSummarySnapshot(ctx, conversationId, workingState, mergedPayload,
			lastExchange.ID,
			s.safeInt(workingState)+len(batch),
			s.resolveSourceTime(lastExchange))
	}

	return workingState
}

// fallbackMerge 回退合并策略
func (s *SessionMemoryLogicImpl) fallbackMerge(existingPayload vo.ConversationSummary, batch []*entity.ChatExchange) vo.ConversationSummary {
	mergedPayload := s.copyPayload(existingPayload)
	batchHighlight := s.renderFallbackBatchHighlight(batch)

	// 合并摘要
	var mergedSummary string
	if existingPayload.Summary != "" && batchHighlight != "" {
		mergedSummary = existingPayload.Summary + "；" + batchHighlight
	} else if existingPayload.Summary != "" {
		mergedSummary = existingPayload.Summary
	} else {
		mergedSummary = batchHighlight
	}
	mergedPayload.Summary = s.clipText(mergedSummary, 1024)

	// 设置会话目标
	if mergedPayload.ConversationGoal == "" && len(batch) > 0 && batch[len(batch)-1].Question != "" {
		mergedPayload.ConversationGoal = s.clipText(batch[len(batch)-1].Question, maxGoalLength)
	}

	// 添加待处理问题
	pendingQuestions := make([]string, 0)
	pendingQuestions = append(pendingQuestions, existingPayload.PendingQuestions...)
	for _, exchange := range batch {
		if exchange.Question != "" {
			pendingQuestions = append(pendingQuestions, s.clipText(exchange.Question, maxItemLength))
		}
	}
	mergedPayload.PendingQuestions = s.deduplicateAndLimit(pendingQuestions)

	// 添加检索提示
	retrievalHints := make([]string, 0)
	retrievalHints = append(retrievalHints, existingPayload.RetrievalHints...)
	if len(batch) > 0 && batch[len(batch)-1].Question != "" {
		retrievalHints = append(retrievalHints, s.extractRetrievalHints(batch[len(batch)-1].Question)...)
	}
	mergedPayload.RetrievalHints = s.deduplicateAndLimit(retrievalHints)

	return s.normalizePayload(mergedPayload)
}

// saveSummarySnapshot 保存摘要快照
func (s *SessionMemoryLogicImpl) saveSummarySnapshot(ctx context.Context, conversationId string,
	currentState *entity.ChatMemorySummary, payload vo.ConversationSummary,
	coveredExchangeId int64, coveredExchangeCount int, lastSourceEditTime time.Time) *entity.ChatMemorySummary {

	latestState, err := s.repo.SelectMemorySummary(ctx, conversationId)
	if err != nil {
		logx.Errorf("查询最新摘要失败, conversationId=%s, err=%v", conversationId, err)
		return currentState
	}

	latestCoveredExchangeId := int64(0)
	if latestState != nil {
		latestCoveredExchangeId = latestState.CoveredExchangeId
	}

	// 检查是否需要更新
	if latestCoveredExchangeId > coveredExchangeId {
		return latestState
	}

	if latestState != nil && latestCoveredExchangeId == coveredExchangeId && latestState.SummaryText != "" {
		return latestState
	}

	summaryText := s.buildLongTermSummaryText(payload)
	summaryJson := s.writePayloadJson(payload)

	if latestState == nil {
		// 插入新记录
		newState := &entity.ChatMemorySummary{
			ID:                   utils.GetSnowflakeNextID(),
			ConversationId:       conversationId,
			CoveredExchangeId:    coveredExchangeId,
			CoveredExchangeCount: coveredExchangeCount,
			CompressionCount:     1,
			SummaryVersion:       1,
			SummaryText:          summaryText,
			SummaryJson:          summaryJson,
			LastSourceEditTime:   lastSourceEditTime,
			Status:               businessStatusYes,
			CreateTime:           time.Now(),
			UpdateTime:           time.Now(),
		}
		if err := s.repo.InsertMemorySummary(ctx, newState); err != nil {
			logx.Errorf("插入摘要失败, conversationId=%s, err=%v", conversationId, err)
			return currentState
		}
		return newState
	}

	// 更新现有记录
	latestState.CoveredExchangeId = coveredExchangeId
	if coveredExchangeCount > latestState.CoveredExchangeCount {
		latestState.CoveredExchangeCount = coveredExchangeCount
	}
	latestState.CompressionCount++
	latestState.SummaryVersion++
	latestState.SummaryText = summaryText
	latestState.SummaryJson = summaryJson
	latestState.LastSourceEditTime = lastSourceEditTime
	latestState.UpdateTime = time.Now()

	if err := s.repo.UpdateMemorySummary(ctx, latestState); err != nil {
		logx.Errorf("更新摘要失败, conversationId=%s, err=%v", conversationId, err)
		return currentState
	}

	return latestState
}

// readSummaryPayload 读取摘要负载
func (s *SessionMemoryLogicImpl) readSummaryPayload(summaryState *entity.ChatMemorySummary) vo.ConversationSummary {
	if summaryState == nil {
		return vo.ConversationSummary{}
	}

	if summaryState.SummaryJson != "" {
		payload := s.parseSummaryPayload(summaryState.SummaryJson)
		if payload.Summary != "" || len(payload.PendingQuestions) > 0 {
			return s.normalizePayload(payload)
		}
	}

	return s.normalizePayload(vo.ConversationSummary{
		Summary: summaryState.SummaryText,
	})
}

// parseSummaryPayload 解析摘要负载
func (s *SessionMemoryLogicImpl) parseSummaryPayload(raw string) vo.ConversationSummary {
	if strings.TrimSpace(raw) == "" {
		return vo.ConversationSummary{}
	}

	jsonStr := s.extractJsonObject(raw)
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		logx.Debugf("解析会话长期摘要JSON失败: %s, err=%v", raw, err)
		return vo.ConversationSummary{}
	}

	payload := vo.ConversationSummary{
		Summary:          s.getString(data, "summary"),
		ConversationGoal: s.getString(data, "conversation_goal"),
		StableFacts:      s.getStringSlice(data, "stable_facts"),
		UserPreferences:  s.getStringSlice(data, "user_preferences"),
		ResolvedPoints:   s.getStringSlice(data, "resolved_points"),
		PendingQuestions: s.getStringSlice(data, "pending_questions"),
		RetrievalHints:   s.getStringSlice(data, "retrieval_hints"),
	}

	return s.normalizePayload(payload)
}

// renderCompressionTranscript 渲染压缩对话记录
func (s *SessionMemoryLogicImpl) renderCompressionTranscript(batch []*entity.ChatExchange) string {
	var builder strings.Builder
	for _, exchange := range batch {
		if exchange.Question != "" {
			builder.WriteString("用户：")
			builder.WriteString(s.clipText(exchange.Question, maxQuestionLength))
			builder.WriteString("\n")
		}
		if exchange.Answer != "" {
			builder.WriteString("助手：")
			builder.WriteString(s.clipText(exchange.Answer, maxAnswerLength))
			builder.WriteString("\n")
		}
		if exchange.TurnStatus == vo.ChatTurnStatusStopped && exchange.ErrorMessage != "" {
			builder.WriteString("补充说明：本轮被停止，说明=")
			builder.WriteString(s.clipText(exchange.ErrorMessage, maxItemLength))
			builder.WriteString("\n")
		}
	}
	return strings.TrimSpace(builder.String())
}

// renderFallbackBatchHighlight 渲染回退批次高亮
func (s *SessionMemoryLogicImpl) renderFallbackBatchHighlight(batch []*entity.ChatExchange) string {
	var highlights []string
	for _, exchange := range batch {
		if exchange.Question != "" {
			highlights = append(highlights, "用户关注："+s.clipText(exchange.Question, maxItemLength))
		}
		if exchange.Answer != "" {
			highlights = append(highlights, "已有结论："+s.clipText(exchange.Answer, maxItemLength))
		}
		if len(highlights) >= 4 {
			break
		}
	}
	return strings.Join(highlights, "；")
}

// renderRecentTranscript 渲染最近对话记录
func (s *SessionMemoryLogicImpl) renderRecentTranscript(exchanges []*entity.ChatExchange, keepRecentTurns, maxChars int) string {
	// 判断是否应保留在最近窗口中
	renderable := slice.Filter(exchanges, func(i int, item *entity.ChatExchange) bool {
		question, answer := strings.TrimSpace(item.Question), strings.TrimSpace(item.Answer)
		return item != nil && item.TurnStatus != vo.ChatTurnStatusRunning && (question != "" || answer != "")
	})

	if len(renderable) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("【最近对话原文】\n")
	for i := 0; i < len(renderable) && i < keepRecentTurns; i++ {
		exchange := renderable[i]
		if exchange.Question != "" {
			builder.WriteString("用户：")
			builder.WriteString(s.clipText(exchange.Question, maxQuestionLength))
			builder.WriteString("\n")
		}
		if exchange.TurnStatus == vo.ChatTurnStatusCompleted && exchange.Answer != "" {
			builder.WriteString("助手：")
			builder.WriteString(s.clipText(exchange.Answer, maxAnswerLength))
			builder.WriteString("\n")
		}
	}

	return s.clipRecentTranscript(builder.String(), maxChars)
}

// renderQuestionRecentTranscript 渲染最近问题记录
func (s *SessionMemoryLogicImpl) renderRecentQuestionTranscript(exchanges []*entity.ChatExchange, keepRecentTurns, maxChars int) string {
	renderable := slice.Filter(exchanges, func(i int, item *entity.ChatExchange) bool {
		return item != nil && item.TurnStatus != vo.ChatTurnStatusRunning && strings.TrimSpace(item.Question) != ""
	})

	if len(renderable) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("【最近相关对话】\n")
	for i := 0; i < len(renderable) && i < keepRecentTurns; i++ {
		exchange := renderable[i]
		builder.WriteString("用户：")
		builder.WriteString(s.clipText(exchange.Question, maxQuestionLength))
		builder.WriteString("\n")
	}

	return s.clipRecentTranscript(builder.String(), maxChars)
}

// buildLongTermSummaryText 构建长期摘要文本
func (s *SessionMemoryLogicImpl) buildLongTermSummaryText(payload vo.ConversationSummary) string {
	normalized := s.normalizePayload(payload)
	var builder strings.Builder

	s.appendSection(&builder, "长期会话摘要", normalized.Summary)
	s.appendSection(&builder, "会话目标", normalized.ConversationGoal)
	s.appendBulletSection(&builder, "已确认事实", normalized.StableFacts)
	s.appendBulletSection(&builder, "用户偏好与约束", normalized.UserPreferences)
	s.appendBulletSection(&builder, "已解决问题", normalized.ResolvedPoints)
	s.appendBulletSection(&builder, "待跟进问题", normalized.PendingQuestions)
	s.appendBulletSection(&builder, "检索提示", normalized.RetrievalHints)

	return s.clipText(strings.TrimSpace(builder.String()), 1024)
}

// normalizePayload 规范化负载
func (s *SessionMemoryLogicImpl) normalizePayload(payload vo.ConversationSummary) vo.ConversationSummary {
	normalizedSummary := s.clipText(strings.TrimSpace(payload.Summary), 1024)
	if normalizedSummary == "" {
		normalizedSummary = s.synthesizeSummaryFromSections(payload)
	}

	return vo.ConversationSummary{
		Summary:          normalizedSummary,
		ConversationGoal: s.clipText(strings.TrimSpace(payload.ConversationGoal), maxGoalLength),
		StableFacts:      s.deduplicateAndLimit(payload.StableFacts),
		UserPreferences:  s.deduplicateAndLimit(payload.UserPreferences),
		ResolvedPoints:   s.deduplicateAndLimit(payload.ResolvedPoints),
		PendingQuestions: s.deduplicateAndLimit(payload.PendingQuestions),
		RetrievalHints:   s.deduplicateAndLimit(payload.RetrievalHints),
	}
}

// synthesizeSummaryFromSections 从各部分合成摘要
func (s *SessionMemoryLogicImpl) synthesizeSummaryFromSections(payload vo.ConversationSummary) string {
	var parts []string
	if payload.ConversationGoal != "" {
		parts = append(parts, "目标："+s.clipText(payload.ConversationGoal, maxItemLength))
	}
	if len(payload.StableFacts) > 0 {
		parts = append(parts, "事实："+strings.Join(payload.StableFacts, "；"))
	}
	if len(payload.PendingQuestions) > 0 {
		parts = append(parts, "待跟进："+strings.Join(payload.PendingQuestions, "；"))
	}
	return s.clipText(strings.Join(parts, "；"), 1024)
}

// deduplicateAndLimit 去重并限制数量
func (s *SessionMemoryLogicImpl) deduplicateAndLimit(values []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, v := range values {
		text := s.clipText(strings.TrimSpace(v), maxItemLength)
		if text == "" {
			continue
		}
		if _, exists := seen[text]; exists {
			continue
		}
		seen[text] = struct{}{}
		result = append(result, text)
		if len(result) >= maxSectionItems {
			break
		}
	}
	return result
}

// copyPayload 复制负载
func (s *SessionMemoryLogicImpl) copyPayload(payload vo.ConversationSummary) vo.ConversationSummary {
	return vo.ConversationSummary{
		Summary:          payload.Summary,
		ConversationGoal: payload.ConversationGoal,
		StableFacts:      append([]string(nil), payload.StableFacts...),
		UserPreferences:  append([]string(nil), payload.UserPreferences...),
		ResolvedPoints:   append([]string(nil), payload.ResolvedPoints...),
		PendingQuestions: append([]string(nil), payload.PendingQuestions...),
		RetrievalHints:   append([]string(nil), payload.RetrievalHints...),
	}
}

// writePayloadJson 写入负载JSON
func (s *SessionMemoryLogicImpl) writePayloadJson(payload vo.ConversationSummary) string {
	normalized := s.normalizePayload(payload)
	data, err := json.Marshal(normalized)
	if err != nil {
		logx.Errorf("序列化会话长期摘要失败, err=%v", err)
		return "{}"
	}
	return string(data)
}

// extractJsonObject 提取JSON对象
func (s *SessionMemoryLogicImpl) extractJsonObject(raw string) string {
	match := jsonObjectPattern.FindString(strings.TrimSpace(raw))
	if match != "" {
		return match
	}
	return strings.TrimSpace(raw)
}

// appendSection 添加段落
func (s *SessionMemoryLogicImpl) appendSection(builder *strings.Builder, title, content string) {
	if strings.TrimSpace(content) == "" {
		return
	}
	if builder.Len() > 0 {
		builder.WriteString("\n")
	}
	builder.WriteString("【")
	builder.WriteString(title)
	builder.WriteString("】\n")
	builder.WriteString(strings.TrimSpace(content))
	builder.WriteString("\n")
}

// appendBulletSection 添加项目符号段落
func (s *SessionMemoryLogicImpl) appendBulletSection(builder *strings.Builder, title string, values []string) {
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

// clipText 裁剪文本
func (s *SessionMemoryLogicImpl) clipText(text string, maxChars int) string {
	normalized := strings.TrimSpace(text)
	if len(normalized) <= maxChars {
		return normalized
	}
	if maxChars <= 1 {
		return ""
	}
	return normalized[:maxChars-1] + "…"
}

// clipRecentTranscript 裁剪最近对话记录
func (s *SessionMemoryLogicImpl) clipRecentTranscript(text string, maxChars int) string {
	normalized := strings.TrimSpace(text)
	if len(normalized) <= maxChars {
		return normalized
	}
	return "…" + normalized[len(normalized)-maxChars+1:]
}

// extractRetrievalHints 提取检索提示
func (s *SessionMemoryLogicImpl) extractRetrievalHints(question string) []string {
	if strings.TrimSpace(question) == "" {
		return []string{}
	}

	hints := make(map[string]struct{})
	matches := retrievalHintPattern.FindAllString(question, -1)
	for _, match := range matches {
		hint := strings.TrimSpace(match)
		if len(hint) >= 2 && !s.isNoiseHint(hint) {
			hints[s.clipText(hint, maxItemLength)] = struct{}{}
		}
		if len(hints) >= maxSectionItems {
			break
		}
	}

	result := make([]string, 0, len(hints))
	for hint := range hints {
		result = append(result, hint)
	}
	return result
}

// isNoiseHint 判断是否为噪音提示
func (s *SessionMemoryLogicImpl) isNoiseHint(value string) bool {
	noiseHints := map[string]bool{
		"请问": true, "帮我": true, "一下": true, "如何": true, "怎么": true,
		"什么": true, "哪个": true, "这个": true, "那个": true, "可以": true, "需要": true,
	}
	return noiseHints[value]
}

// resolveSourceTime 解析源时间
func (s *SessionMemoryLogicImpl) resolveSourceTime(exchange *entity.ChatExchange) time.Time {
	if exchange == nil {
		return time.Now()
	}
	// 使用UpdateTime作为编辑时间，如果没有则使用CreateTime
	if !exchange.UpdateTime.IsZero() {
		return exchange.UpdateTime
	}
	return exchange.CreateTime
}

// toSummaryView 转换为摘要视图
func (s *SessionMemoryLogicImpl) toSummaryView(conversationId string, summary *entity.ChatMemorySummary) *vo.ConversationMemorySummaryView {
	if summary == nil {
		return s.emptySummaryView(conversationId)
	}

	updateTime := summary.UpdateTime
	lastSourceUpdateTime := summary.LastSourceUpdateTime

	return &vo.ConversationMemorySummaryView{
		ConversationId:       conversationId,
		HasSummary:           summary.SummaryText != "",
		CoveredExchangeId:    summary.CoveredExchangeId,
		CoveredExchangeCount: summary.CoveredExchangeCount,
		CompressionCount:     summary.CompressionCount,
		SummaryVersion:       summary.SummaryVersion,
		SummaryText:          summary.SummaryText,
		Summary:              s.readSummaryPayload(summary),
		LastSourceUpdateTime: &lastSourceUpdateTime,
		UpdateTime:           &updateTime,
	}
}

// emptySummaryView 创建空摘要视图
func (s *SessionMemoryLogicImpl) emptySummaryView(conversationId string) *vo.ConversationMemorySummaryView {
	return &vo.ConversationMemorySummaryView{
		ConversationId:       conversationId,
		HasSummary:           false,
		CoveredExchangeId:    0,
		CoveredExchangeCount: 0,
		CompressionCount:     0,
		SummaryVersion:       0,
		SummaryText:          "",
		Summary:              vo.ConversationSummary{},
		LastSourceEditTime:   nil,
		UpdateTime:           nil,
	}
}

// safeInt 安全获取int值
func (s *SessionMemoryLogicImpl) safeInt(summary *entity.ChatMemorySummary) int {
	if summary == nil {
		return 0
	}
	return summary.CoveredExchangeCount
}

// defaultLong 安全获取CoveredExchangeId
func (s *SessionMemoryLogicImpl) defaultLong(summary *entity.ChatMemorySummary) int64 {
	if summary == nil {
		return 0
	}
	return summary.CoveredExchangeId
}

// safeIntValue 安全获取CoveredExchangeCount
func (s *SessionMemoryLogicImpl) safeIntValue(summary *entity.ChatMemorySummary) int {
	if summary == nil {
		return 0
	}
	return summary.CoveredExchangeCount
}

// safeCompressionCount 安全获取CompressionCount
func (s *SessionMemoryLogicImpl) safeCompressionCount(summary *entity.ChatMemorySummary) int {
	if summary == nil {
		return 0
	}
	return summary.CompressionCount
}

// assembleHistory 组装历史记录
func (s *SessionMemoryLogicImpl) assembleHistory(longTermSummary, recentTranscript string) string {
	return s.joinNonBlank(longTermSummary, recentTranscript, "\n\n")
}

// joinNonBlank 连接非空字符串
func (s *SessionMemoryLogicImpl) joinNonBlank(left, right, delimiter string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	return left + delimiter + right
}

// getString 从map获取字符串
func (s *SessionMemoryLogicImpl) getString(data map[string]interface{}, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

// getStringSlice 从map获取字符串切片
func (s *SessionMemoryLogicImpl) getStringSlice(data map[string]interface{}, key string) []string {
	if v, ok := data[key].([]interface{}); ok {
		var result []string
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				result = append(result, str)
			}
		}
		return result
	}
	return []string{}
}
