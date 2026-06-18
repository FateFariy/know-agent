package logic

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
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
	retrievalHintPattern = regexp.MustCompile(`[a-zA-Z0-9._-]{2,}|[\p{Han}]{2,12}`)
)

// SessionMemoryLogicImpl 会话记忆逻辑实现
type SessionMemoryLogicImpl struct {
	historySummary          config.HistorySummaryConf
	repo                    adapter.ChatRepository
	chatModel               *ObservedChatModelImpl[*schema.AgenticMessage]
	promptTemplate          PromptTemplateLogic
	refreshingMu            sync.Mutex
	refreshing              map[string]struct{}
	rewriteHistoryTurns     int
	questionHistoryMaxChars int
}

// NewSessionMemoryLogic 创建会话记忆逻辑实例
func NewSessionMemoryLogic(svcCtx *svc.ServiceContext, repo adapter.ChatRepository, chanMode *ObservedChatModelImpl[*schema.AgenticMessage], promptTemplate PromptTemplateLogic) *SessionMemoryLogicImpl {
	return &SessionMemoryLogicImpl{
		repo:                    repo,
		refreshing:              make(map[string]struct{}),
		historySummary:          svcCtx.Config.Memory.HistorySummary,
		rewriteHistoryTurns:     svcCtx.Config.Memory.RewriteHistoryTurns,
		questionHistoryMaxChars: svcCtx.Config.Memory.QuestionHistoryMaxChars,
		chatModel:               chanMode,
		promptTemplate:          promptTemplate,
	}
}

// LoadMemoryContext 加载会话记忆上下文
func (s *SessionMemoryLogicImpl) LoadMemoryContext(ctx context.Context, conversationId string, tracer *vo.ConversationTrace) (*vo.MemoryContext, error) {
	if strutil.IsBlank(conversationId) {
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

	// 刷新摘要（如果需要）
	summaryState, err = s.refreshSummaryIfNecessary(ctx, conversationId, summaryState, tracer)
	if err != nil {
		return nil, err
	}

	summaryPayload := s.readSummary(summaryState)

	recentTranscript := s.renderRecentTranscript(recentExchanges, s.rewriteHistoryTurns, s.historySummary.RecentTranscriptMaxChars)
	answerRecentTranscript := s.renderRecentQuestionTranscript(recentExchanges, s.rewriteHistoryTurns, s.historySummary.RecentTranscriptMaxChars)
	longTermSummary := ""
	if summaryState != nil {
		longTermSummary = strutil.Trim(summaryState.SummaryText)
	}

	return &vo.MemoryContext{
		AssembledHistory:         s.assembleHistory(longTermSummary, recentTranscript),
		LongTermSummary:          longTermSummary,
		RecentTranscript:         recentTranscript,
		QuestionRecentTranscript: answerRecentTranscript,
		Summary:                  summaryPayload,
		IsCompressed:             strutil.IsNotBlank(longTermSummary),
		CoveredExchangeId:        s.defaultLong(summaryState),
		CoveredExchangeCount:     s.safeIntValue(summaryState),
		CompressionCount:         s.safeCompressionCount(summaryState),
	}, nil
}

// RefreshConversationSummaryAsync 异步刷新会话摘要
func (s *SessionMemoryLogicImpl) RefreshConversationSummaryAsync(ctx context.Context, conversationId string) {
	if strutil.IsBlank(conversationId) {
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

		s.refreshSummaryIfNecessary(ctx, conversationId, nil, tracer)
	}()
}

// GetConversationSummary 获取会话摘要
func (s *SessionMemoryLogicImpl) GetConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error) {
	if strutil.IsBlank(conversationId) {
		return nil, nil
	}

	summary, err := s.repo.SelectMemorySummary(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	summary.IsCompressed = strutil.IsNotBlank(summary.SummaryText)

	return summary, nil
}

// RebuildConversationSummary 重建会话摘要
func (s *SessionMemoryLogicImpl) RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error) {
	if strutil.IsBlank(conversationId) {
		return &entity.ChatMemorySummary{}, nil
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
	rebuiltState, err := s.refreshSummaryIfNecessary(ctx, conversationId, nil, tracer)
	if err != nil {
		return nil, err
	}
	return s.toSummaryView(conversationId, rebuiltState), nil
}

// DeleteConversationSummary 删除会话摘要
func (s *SessionMemoryLogicImpl) DeleteConversationSummary(ctx context.Context, conversationId string) error {
	if strutil.IsBlank(conversationId) {
		return nil
	}
	return s.repo.DeleteMemorySummary(ctx, conversationId)
}

// refreshSummaryIfNecessary 刷新摘要（如果需要）
func (s *SessionMemoryLogicImpl) refreshSummaryIfNecessary(ctx context.Context, conversationId string,
	currentState *entity.ChatMemorySummary, tracer *vo.ConversationTrace) (*entity.ChatMemorySummary, error) {
	// 获取增量对话（只拉取摘要尚未覆盖的新增轮次，避免重复压缩）
	coveredExchangeId := utils.Ternary(currentState == nil, 0, currentState.CoveredExchangeId)
	incrementalExchanges, err := s.repo.ListExchangesAfter(ctx, conversationId, coveredExchangeId)
	if err != nil {
		logx.Errorf("查询增量对话失败, conversationId=%s, err=%v", conversationId, err)
		return currentState, err
	}

	// 过滤已完成的对话（只有已完成的对话才参与摘要提取）
	stableExchanges := slice.Filter(incrementalExchanges, func(i int, item *entity.ChatExchange) bool {
		return item.TurnStatus == vo.ChatTurnStatusCompleted && strutil.IsNotBlank(item.Question)
	})

	// 检查是否需要压缩（超出保留窗口的对话需要压缩）
	overflowCount := len(stableExchanges) - s.historySummary.KeepRecentTurns
	if overflowCount <= 0 {
		return currentState, nil
	}

	// 分批压缩溢出对话（按批次大小分批处理，避免单次压缩内容过多）
	overflowExchanges := stableExchanges[:overflowCount]
	workingState := currentState
	compressionBatchTurns := s.historySummary.CompressionBatchTurns

	for start := 0; start < len(overflowExchanges); start += compressionBatchTurns {
		end := min(start+compressionBatchTurns, len(overflowExchanges))
		batch := overflowExchanges[start:end]

		// 优先使用LLM合并摘要，失败时回退到规则合并
		oldSummary := s.readSummary(workingState)
		newSummary, err := s.mergeSummaryByLLM(ctx, oldSummary, batch, tracer)
		if err != nil {
			logx.Errorf("LLM合并会话长期摘要失败，回退到规则压缩, conversationId=%s, err=%v", conversationId, err)
			newSummary = s.fallbackMerge(oldSummary, batch)
		}

		// 保存摘要快照（记录已覆盖的对话ID和数量）
		lastExchange := batch[len(batch)-1]
		coveredExchangeCount := utils.Ternary(workingState == nil, 0, workingState.CoveredExchangeCount+len(batch))
		workingState, err = s.saveSummarySnapshot(ctx, lastExchange, newSummary, coveredExchangeCount)
		if err != nil {
			logx.Errorf("保存会话摘要快照失败, conversationId=%s, err=%v", conversationId, err)
			return currentState, err
		}
	}

	return workingState, nil
}

// mergeSummaryByLLM 由大模型合并摘要
func (s *SessionMemoryLogicImpl) mergeSummaryByLLM(ctx context.Context, oldSummary *entity.ConversationSummary,
	batch []*entity.ChatExchange, tracer *vo.ConversationTrace) (*entity.ConversationSummary, error) {
	// 渲染系统提示词
	systemPrompt, err := s.promptTemplate.Render(prompt.ConversationSummarySystem, nil)
	if err != nil {
		return nil, err
	}

	// 序列化现有摘要（规范化后转为JSON）
	serializeSummary, err := s.serializeSummary(s.normalizeSummary(copySummary(oldSummary)))
	if err != nil {
		return nil, err
	}

	// 构建用户提示词变量（包含现有摘要和新对话批次）
	variables := map[string]any{
		"existingSummaryJson":  serializeSummary,
		"newConversationBatch": s.renderCompressionTranscript(batch),
	}
	userPrompt, err := s.promptTemplate.Render(prompt.ConversationSummaryMerge, variables)
	if err != nil {
		return nil, err
	}

	// 调用LLM生成合并后的摘要
	content, err := s.chatModel.Generate(ctx, vo.ChatStageSummary, systemPrompt, userPrompt, tracer)
	newSummary := s.deserializeSummary(content)
	if newSummary == nil {
		return nil, err
	}

	// 规范化摘要（限制字段长度、去重）
	return s.normalizeSummary(newSummary), nil
}

// fallbackMerge 回退合并策略（当LLM合并失败时使用规则合并）
func (s *SessionMemoryLogicImpl) fallbackMerge(oldSummary *entity.ConversationSummary, batch []*entity.ChatExchange) *entity.ConversationSummary {
	newSummary := copySummary(oldSummary)
	batchHighlight := s.renderFallbackBatchHighlight(batch)

	// 合并摘要文本（用分号连接旧摘要和批次高亮）
	newSummary.Summary = s.joinNonBlank(newSummary.Summary, batchHighlight, ";")
	newSummary.Summary = s.clipText(newSummary.Summary, s.historySummary.SummaryMaxChars)

	// 设置会话目标（若尚未设置，则取最后一条问题作为目标）
	lastQuestion := batch[len(batch)-1].Question
	if strutil.IsBlank(newSummary.ConversationGoal) && strutil.IsNotBlank(lastQuestion) {
		newSummary.ConversationGoal = s.clipText(lastQuestion, maxGoalLength)
	}

	// 累积待处理问题（保留旧问题，追加批次中的新问题）
	pendingQuestions := make([]string, 0, len(batch)+len(newSummary.PendingQuestions))
	pendingQuestions = append(pendingQuestions, oldSummary.PendingQuestions...)
	for _, exchange := range batch {
		if strutil.IsNotBlank(exchange.Question) {
			pendingQuestions = append(pendingQuestions, s.clipText(exchange.Question, maxItemLength))
		}
	}
	newSummary.PendingQuestions = s.deduplicateAndLimit(pendingQuestions)

	// 累积检索提示（从最后一条问题中提取关键词）
	retrievalHints := make([]string, 0, len(oldSummary.RetrievalHints))
	retrievalHints = append(retrievalHints, oldSummary.RetrievalHints...)
	if len(batch) > 0 && strutil.IsNotBlank(lastQuestion) {
		retrievalHints = append(retrievalHints, s.extractRetrievalHints(lastQuestion)...)
	}
	newSummary.RetrievalHints = s.deduplicateAndLimit(retrievalHints)

	return s.normalizeSummary(newSummary)
}

// saveSummarySnapshot 保存摘要快照
func (s *SessionMemoryLogicImpl) saveSummarySnapshot(ctx context.Context, lastExchange *entity.ChatExchange,
	summary *entity.ConversationSummary, coveredExchangeCount int) (*entity.ChatMemorySummary, error) {
	// 获取当前摘要
	latestState, err := s.repo.SelectMemorySummary(ctx, lastExchange.ConversationId)
	if err != nil {
		return nil, err
	}

	// 检查是否需要更新
	if latestState != nil {
		exchangeId := latestState.CoveredExchangeId
		if (exchangeId == lastExchange.ID && strutil.IsNotBlank(latestState.SummaryText)) || exchangeId > lastExchange.ID {
			return latestState, nil
		}
	}

	summaryText := s.buildLongTermSummaryText(summary)
	summaryJson, err := s.serializeSummary(summary)
	if err != nil {
		return nil, err
	}

	if latestState == nil {
		// 插入新记录
		newState := &entity.ChatMemorySummary{
			ID:                   utils.GetSnowflakeNextID(),
			ConversationId:       lastExchange.ConversationId,
			CoveredExchangeId:    lastExchange.ID,
			CoveredExchangeCount: coveredExchangeCount,
			CompressionCount:     1,
			SummaryVersion:       1,
			SummaryText:          summaryText,
			SummaryJson:          summaryJson,
			LastSourceUpdateTime: lastExchange.UpdateTime,
			UpdateTime:           time.Now(),
		}
		if err = s.repo.InsertMemorySummary(ctx, newState); err != nil {
			return nil, err
		}
		return newState, nil
	}

	// 更新现有记录
	latestState.CoveredExchangeId = lastExchange.ID
	latestState.CoveredExchangeCount = max(latestState.CoveredExchangeCount, coveredExchangeCount)
	latestState.CompressionCount++
	latestState.SummaryVersion++
	latestState.SummaryText = summaryText
	latestState.SummaryJson = summaryJson
	latestState.LastSourceUpdateTime = lastExchange.UpdateTime
	latestState.UpdateTime = time.Now()
	if err = s.repo.UpdateMemorySummaryById(ctx, latestState); err != nil {
		return nil, err
	}

	return latestState, nil
}

// readSummary 读取摘要
func (s *SessionMemoryLogicImpl) readSummary(summaryState *entity.ChatMemorySummary) *entity.ConversationSummary {
	if summaryState == nil {
		return &entity.ConversationSummary{}
	}

	if strutil.IsNotBlank(summaryState.SummaryJson) {
		summary := s.deserializeSummary(summaryState.SummaryJson)
		if summary != nil {
			return s.normalizeSummary(summary)
		}
	}

	return s.normalizeSummary(&entity.ConversationSummary{Summary: summaryState.SummaryText})
}

// deserializeSummary 反序列化摘要
func (s *SessionMemoryLogicImpl) deserializeSummary(raw string) *entity.ConversationSummary {
	raw = extractJsonObject(raw)
	summary := &entity.ConversationSummary{}
	if err := json.Unmarshal([]byte(raw), summary); err != nil {
		logx.Debugf("反序列化会话长期摘要 JSON 失败: %s, err=%v", raw, err)
		return nil
	}

	return summary
}

// normalizeSummary 规范化摘要
func (s *SessionMemoryLogicImpl) normalizeSummary(payload *entity.ConversationSummary) *entity.ConversationSummary {
	summary := s.clipText(strutil.Trim(payload.Summary), s.historySummary.SummaryMaxChars)
	summaryEntity := &entity.ConversationSummary{
		ConversationGoal: s.clipText(strutil.Trim(payload.ConversationGoal), maxGoalLength),
		StableFacts:      s.deduplicateAndLimit(payload.StableFacts),
		UserPreferences:  s.deduplicateAndLimit(payload.UserPreferences),
		ResolvedPoints:   s.deduplicateAndLimit(payload.ResolvedPoints),
		PendingQuestions: s.deduplicateAndLimit(payload.PendingQuestions),
		RetrievalHints:   s.deduplicateAndLimit(payload.RetrievalHints),
	}
	summaryEntity.Summary = utils.Ternary(strutil.IsNotBlank(summary), summary, s.synthesizeSummaryFromSections(summaryEntity))
	return summaryEntity
}

// buildLongTermSummaryText 构建长期摘要文本
func (s *SessionMemoryLogicImpl) buildLongTermSummaryText(payload *entity.ConversationSummary) string {
	normalized := s.normalizeSummary(payload)
	var builder strings.Builder

	s.appendSection(&builder, "长期会话摘要", normalized.Summary)
	s.appendSection(&builder, "会话目标", normalized.ConversationGoal)
	s.appendBulletSection(&builder, "已确认事实", normalized.StableFacts)
	s.appendBulletSection(&builder, "用户偏好与约束", normalized.UserPreferences)
	s.appendBulletSection(&builder, "已解决问题", normalized.ResolvedPoints)
	s.appendBulletSection(&builder, "待跟进问题", normalized.PendingQuestions)
	s.appendBulletSection(&builder, "检索提示", normalized.RetrievalHints)

	return s.clipText(strutil.Trim(builder.String()), 1024)
}

// renderCompressionTranscript 渲染压缩对话记录
func (s *SessionMemoryLogicImpl) renderCompressionTranscript(batch []*entity.ChatExchange) string {
	var builder strings.Builder
	for _, exchange := range batch {
		if strutil.IsNotBlank(exchange.Question) {
			builder.WriteString("用户：")
			builder.WriteString(s.clipText(exchange.Question, maxQuestionLength))
			builder.WriteString("\n")
		}
		if strutil.IsNotBlank(exchange.Answer) {
			builder.WriteString("助手：")
			builder.WriteString(s.clipText(exchange.Answer, maxAnswerLength))
			builder.WriteString("\n")
		}
		if exchange.TurnStatus == vo.ChatTurnStatusStopped && strutil.IsNotBlank(exchange.ErrorMessage) {
			builder.WriteString("补充说明：本轮被停止，说明=")
			builder.WriteString(s.clipText(exchange.ErrorMessage, maxItemLength))
			builder.WriteString("\n")
		}
	}
	return strutil.Trim(builder.String())
}

// renderFallbackBatchHighlight 渲染回退批次高亮
func (s *SessionMemoryLogicImpl) renderFallbackBatchHighlight(batch []*entity.ChatExchange) string {
	var highlights []string
	for _, exchange := range batch {
		if strutil.IsNotBlank(exchange.Question) {
			highlights = append(highlights, "用户关注："+s.clipText(exchange.Question, maxItemLength))
		}
		if strutil.IsNotBlank(exchange.Answer) {
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
			builder.WriteString(s.clipText(exchange.Question, maxQuestionLength))
			builder.WriteString("\n")
		}
		if exchange.TurnStatus == vo.ChatTurnStatusCompleted && strutil.IsNotBlank(exchange.Answer) {
			builder.WriteString("助手：")
			builder.WriteString(s.clipText(exchange.Answer, maxAnswerLength))
			builder.WriteString("\n")
		}
	}

	return s.clipRecentTranscript(builder.String(), maxChars)
}

// renderRecentQuestionTranscript 渲染最近问题记录
func (s *SessionMemoryLogicImpl) renderRecentQuestionTranscript(exchanges []*entity.ChatExchange, keepRecentTurns, maxChars int) string {
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
		builder.WriteString(s.clipText(exchange.Question, maxQuestionLength))
		builder.WriteString("\n")
	}

	return s.clipRecentTranscript(builder.String(), maxChars)
}

// serializeSummary 序列化摘要
func (s *SessionMemoryLogicImpl) serializeSummary(summary *entity.ConversationSummary) (string, error) {
	data, err := json.Marshal(summary)
	if err != nil {
		logx.Errorf("序列化会话长期摘要失败, err=%v", err)
		return "{}", err
	}
	return string(data), nil
}

// synthesizeSummaryFromSections 从各部分合成摘要
func (s *SessionMemoryLogicImpl) synthesizeSummaryFromSections(payload *entity.ConversationSummary) string {
	var parts []string
	if strutil.IsNotBlank(payload.ConversationGoal) {
		parts = append(parts, "目标："+s.clipText(payload.ConversationGoal, maxItemLength))
	}
	if len(payload.StableFacts) > 0 {
		parts = append(parts, "事实："+strings.Join(payload.StableFacts, "；"))
	}
	if len(payload.PendingQuestions) > 0 {
		parts = append(parts, "待跟进："+strings.Join(payload.PendingQuestions, "；"))
	}
	return s.clipText(strings.Join(parts, "；"), s.historySummary.SummaryMaxChars)
}

// extractRetrievalHints 提取检索提示
func (s *SessionMemoryLogicImpl) extractRetrievalHints(question string) []string {
	if strutil.IsBlank(question) {
		return []string{}
	}

	matches := retrievalHintPattern.FindAllString(question, -1)
	hints := make([]string, 0, len(matches))
	for _, match := range matches {
		hint := strutil.Trim(match)
		if len(hint) >= 2 && !isNoiseHint(hint) {
			hints = append(hints, s.clipText(hint, maxItemLength))
		}
		if len(hints) >= maxSectionItems {
			break
		}
	}
	s.deduplicateAndLimit(hints)

	return hints
}

// extractJsonObject 提取JSON对象
func extractJsonObject(raw string) string {
	trimmed := strutil.Trim(raw)
	if strutil.IsBlank(trimmed) {
		return trimmed
	}

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || end < start {
		return trimmed
	}
	return trimmed[start : end+1]
}

// deduplicateAndLimit 去重并限制数量
func (s *SessionMemoryLogicImpl) deduplicateAndLimit(values []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range values {
		text := s.clipText(strutil.Trim(v), maxItemLength)
		if seen[text] || strutil.IsBlank(text) {
			continue
		}
		seen[text] = true
		result = append(result, text)
		if len(result) >= maxSectionItems {
			break
		}
	}
	return result
}

// appendSection 添加段落
func (s *SessionMemoryLogicImpl) appendSection(builder *strings.Builder, title, content string) {
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
	normalized := strutil.Trim(text)
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
	normalized := strutil.Trim(text)
	if len(normalized) <= maxChars {
		return normalized
	}
	return "…" + normalized[len(normalized)-maxChars+1:]
}

// assembleHistory 组装历史记录
func (s *SessionMemoryLogicImpl) assembleHistory(longTermSummary, recentTranscript string) string {
	return s.joinNonBlank(longTermSummary, recentTranscript, "\n\n")
}

// joinNonBlank 连接非空字符串
func (s *SessionMemoryLogicImpl) joinNonBlank(left, right, delimiter string) string {
	left = strutil.Trim(left)
	right = strutil.Trim(right)
	if strutil.IsBlank(left) {
		return right
	}
	if strutil.IsBlank(right) {
		return left
	}
	return left + delimiter + right
}

// isNoiseHint 判断是否为噪音提示
func isNoiseHint(value string) bool {
	noiseHints := map[string]bool{
		"请问": true, "帮我": true, "一下": true, "如何": true, "怎么": true,
		"什么": true, "哪个": true, "这个": true, "那个": true, "可以": true, "需要": true,
	}
	return noiseHints[value]
}

// resolveSourceTime 解析源时间
func resolveSourceTime(exchange *entity.ChatExchange) time.Time {
	if exchange == nil {
		return time.Now()
	}
	// 使用UpdateTime作为编辑时间，如果没有则使用CreateTime
	if !exchange.UpdateTime.IsZero() {
		return exchange.UpdateTime
	}
	return exchange.CreateTime
}

// safeInt 安全获取int值
func safeInt(summary *entity.ChatMemorySummary) int {
	if summary == nil {
		return 0
	}
	return summary.CoveredExchangeCount
}

// safeIntValue 安全获取CoveredExchangeCount
func safeIntValue(summary *entity.ChatMemorySummary) int {
	if summary == nil {
		return 0
	}
	return summary.CoveredExchangeCount
}

// safeCompressionCount 安全获取CompressionCount
func safeCompressionCount(summary *entity.ChatMemorySummary) int {
	if summary == nil {
		return 0
	}
	return summary.CompressionCount
}

// copySummary 复制摘要
func copySummary(summary *entity.ConversationSummary) *entity.ConversationSummary {
	if summary == nil {
		return &entity.ConversationSummary{}
	}
	return &entity.ConversationSummary{
		Summary:          summary.Summary,
		ConversationGoal: summary.ConversationGoal,
		StableFacts:      append([]string{}, summary.StableFacts...),
		UserPreferences:  append([]string{}, summary.UserPreferences...),
		ResolvedPoints:   append([]string{}, summary.ResolvedPoints...),
		PendingQuestions: append([]string{}, summary.PendingQuestions...),
		RetrievalHints:   append([]string{}, summary.RetrievalHints...),
	}
}
