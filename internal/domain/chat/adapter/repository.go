package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
)

// ChatRepository 聊天仓储接口
type ChatRepository interface {
	// StartExchange 创建对话交换记录
	StartExchange(ctx context.Context, dialogue *entity.ChatDialogue) (*entity.ChatExchange, error)

	// CompleteExchange 完成对话交换记录
	CompleteExchange(ctx context.Context, exchange *entity.ChatExchange) error

	// SelectExchange 获取单个对话交换记录
	SelectExchange(ctx context.Context, conversationId string, exchangeId int64) (*entity.ChatExchange, error)

	// ListExchanges 列出对话的所有交换记录
	ListExchanges(ctx context.Context, conversationId string) ([]*entity.ChatExchange, error)

	// ListExchangesAfter 列出某个交换之后的记录
	ListExchangesAfter(ctx context.Context, conversationId string, afterExchangeId int64) ([]*entity.ChatExchange, error)

	// ListRecentExchanges 列出最近的交换记录
	ListRecentExchanges(ctx context.Context, conversationId string, limit int) ([]*entity.ChatExchange, error)

	// RefreshSessionScope 刷新会话范围（更新会话状态、模式、文档选择）
	RefreshSessionScope(ctx context.Context, dialogue *entity.ChatDialogue) error

	// SelectDialogue 获取会话
	SelectDialogue(ctx context.Context, conversationId string) (*entity.ChatDialogue, error)

	// ListDialogues 列出所有会话
	ListDialogues(ctx context.Context) ([]*entity.ChatDialogue, error)

	// ListDialoguePage 分页查询会话
	ListDialoguePage(ctx context.Context, pageNo, pageSize int, keyword string, chatMode, latestTurnStatus int) ([]*entity.ChatDialogue, int64, error)

	// DeleteSession 删除会话及所有交换记录
	DeleteSession(ctx context.Context, conversationId string) (int64, int64, error)
}
