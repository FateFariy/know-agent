package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChatRepository 聊天仓储接口
type ChatRepository interface {
	// Do 运行一个事务，事务中执行 fn 函数
	Do(ctx context.Context, fn func(ctx context.Context) error) error

	// ========== 对话轮次（Exchange）相关 ==========

	// InsertExchange 插入对话记录
	InsertExchange(ctx context.Context, exchange *entity.ChatExchange) error

	// UpdateExchangeById 根据ID更新对话记录
	UpdateExchangeById(ctx context.Context, exchange *entity.ChatExchange) error

	// ListExchanges 列出对话的所有记录
	ListExchanges(ctx context.Context, conversationId string) ([]*entity.ChatExchange, error)

	// SelectExchangeById 根据ID查询对话记录
	SelectExchangeById(ctx context.Context, exchangeId int64) (*entity.ChatExchange, error)

	// ListExchangesAfter 列出某个记录之后的记录
	ListExchangesAfter(ctx context.Context, conversationId string, afterExchangeId int64) ([]*entity.ChatExchange, error)

	// ListRecentExchanges 列出最近的记录
	ListRecentExchanges(ctx context.Context, conversationId string, limit int) ([]*entity.ChatExchange, error)

	// ========== 会话（Dialogue）相关 ==========

	// UpdateDialogueByConversationId 根据对话ID更新对话记录
	UpdateDialogueByConversationId(ctx context.Context, dialogue *entity.ChatDialogue) error

	// UpsertDialogue 创建或更新会话
	UpsertDialogue(ctx context.Context, dialogue *entity.ChatDialogue) error

	// RefreshSessionScope 刷新会话范围（更新会话状态、模式、文档选择）
	RefreshSessionScope(ctx context.Context, dialogue *entity.ChatDialogue) error

	// ========== 会话归档与查询 ==========

	// SelectSessionRecord 获取会话记录
	SelectSessionRecord(ctx context.Context, conversationId string) (*vo.ConversationArchiveRecord, error)

	// ListSessionRecordPage 列出会话记录分页
	ListSessionRecordPage(ctx context.Context, pageNo, pageSize, chatMode, latestTurnStatus int, keyword string) ([]*vo.ConversationArchiveRecord, int64, error)

	// DeleteSession 删除会话及所有记录
	DeleteSession(ctx context.Context, conversationId string) (int64, int64, error)

	// ========== 会话记忆摘要相关 ==========

	// SelectMemorySummary 查询会话记忆摘要
	SelectMemorySummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// InsertMemorySummary 插入会话记忆摘要
	InsertMemorySummary(ctx context.Context, summary *entity.ChatMemorySummary) error

	// UpdateMemorySummaryById 更新会话记忆摘要
	UpdateMemorySummaryById(ctx context.Context, summary *entity.ChatMemorySummary) error

	// DeleteMemorySummary 删除会话记忆摘要
	DeleteMemorySummary(ctx context.Context, conversationId string) error

	// ========== 会话阶段追踪相关 ==========

	// InsertStage 创建阶段记录
	InsertStage(ctx context.Context, stage *entity.ChatExchangeTraceStage) error

	// UpdateStageById 更新阶段记录
	UpdateStageById(ctx context.Context, stage *entity.ChatExchangeTraceStage) error

	// SelectStages 查询阶段记录
	SelectStages(ctx context.Context, conversationId string, exchangeId int64) ([]*entity.ChatExchangeTraceStage, error)

	// DeleteStage 删除阶段记录
	DeleteStage(ctx context.Context, conversationId string) error

	// ========== 检索结果相关 ==========

	// InsertChannelExecutions 插入渠道执行记录
	InsertChannelExecutions(ctx context.Context, executions []*vo.ChatChannelExecution) error

	// InsertRetrievalResults 插入检索结果
	InsertRetrievalResults(ctx context.Context, results []*vo.ChatRetrievalResult) error

	// SelectRetrievalResults 查询检索结果
	SelectRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*vo.ChatRetrievalResult, error)

	// SelectChannelExecutions 查询渠道执行记录
	SelectChannelExecutions(ctx context.Context, conversationId string, exchangeId int64) ([]*vo.ChatChannelExecution, error)

	// DeleteRetrievalResultsByConversationId 删除会话所有检索结果
	DeleteRetrievalResultsByConversationId(ctx context.Context, conversationId string) error

	// DeleteChannelExecutionsByConversationId 删除会话所有渠道执行记录
	DeleteChannelExecutionsByConversationId(ctx context.Context, conversationId string) error
}
