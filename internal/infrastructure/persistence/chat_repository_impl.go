package persistence

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/infrastructure/model"
	"github.com/swiftbit/know-agent/internal/svc"
)

var _ adapter.ChatRepository = (*ChatRepositoryImpl)(nil)

type ChatRepositoryImpl struct {
	db *gorm.DB
}

func NewChatRepository(svcCtx *svc.ServiceContext) *ChatRepositoryImpl {
	return &ChatRepositoryImpl{
		db: svcCtx.Db,
	}
}

// StartExchange 创建对话交换记录
func (r *ChatRepositoryImpl) StartExchange(ctx context.Context, dialogue *entity.ChatDialogue) (*entity.ChatExchange, error) {
	chatExchange := &entity.ChatExchange{
		ConversationId: dialogue.ConversationId,
		Question:       dialogue.Question,
		TurnStatus:     vo.ChatExchangeStatusRunning,
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
	}
	return chatExchange, r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dialogue.SessionStatus = vo.ChatSessionStatusRunning
		if err := r.upsertDialogue(ctx, dialogue); err != nil {
			return err
		}
		return r.db.WithContext(ctx).Create(convert.ToChatExchangeModel(chatExchange)).Error
	})
}

// CompleteExchange 完成对话交换记录
func (r *ChatRepositoryImpl) CompleteExchange(ctx context.Context, exchange *entity.ChatExchange) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"answer":                 exchange.Answer,
			"thinking_steps":         exchange.ThinkingSteps,
			"reference_list":         exchange.ReferenceList,
			"recommendation_list":    exchange.RecommendationList,
			"used_tool_list":         exchange.UsedToolList,
			"debug_trace_json":       exchange.DebugTraceJson,
			"turn_status":            exchange.TurnStatus,
			"error_message":          exchange.ErrorMessage,
			"first_response_time_ms": exchange.FirstResponseTimeMs,
			"total_response_time_ms": exchange.TotalResponseTimeMs,
		}
		if err := r.db.WithContext(ctx).Model(&model.ChatExchange{}).
			Where("id = ? AND conversation_id = ?", exchange.ID, exchange.ConversationId).
			Updates(updates).Error; err != nil {
			return err
		}

		return r.db.WithContext(ctx).Model(&model.ChatDialogue{}).
			Where("conversation_id = ?", exchange.ConversationId).
			Update("session_status", vo.ChatSessionStatusIdle).Error
	})
}

// SelectExchange 获取单个对话交换记录
func (r *ChatRepositoryImpl) SelectExchange(ctx context.Context, conversationId string, exchangeId int64) (*entity.ChatExchange, error) {
	var chatExchange *entity.ChatExchange
	err := r.db.WithContext(ctx).
		Model(&model.ChatExchange{}).
		Where("id = ? AND conversation_id = ?", exchangeId, conversationId).
		First(&chatExchange).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return chatExchange, nil
}

// ListExchanges 列出对话的所有交换记录
func (r *ChatRepositoryImpl) ListExchanges(ctx context.Context, conversationId string) ([]*entity.ChatExchange, error) {
	var exchanges []*entity.ChatExchange
	err := r.db.WithContext(ctx).
		Model(&model.ChatExchange{}).
		Where("conversation_id = ?", conversationId).
		Order("create_time ASC, id ASC").
		Find(&exchanges).Error
	if err != nil {
		return nil, err
	}
	return exchanges, nil
}

// ListExchangesAfter 列出某个交换之后的记录
func (r *ChatRepositoryImpl) ListExchangesAfter(ctx context.Context, conversationId string, afterExchangeId int64) ([]*entity.ChatExchange, error) {
	var exchanges []*entity.ChatExchange
	query := r.db.WithContext(ctx).Model(&model.ChatExchange{}).Where("conversation_id = ?", conversationId)
	if afterExchangeId > 0 {
		query = query.Where("id > ?", afterExchangeId)
	}
	if err := query.Order("create_time ASC, id ASC").Find(&exchanges).Error; err != nil {
		return nil, err
	}
	return exchanges, nil
}

// ListRecentExchanges 列出最近的交换记录
func (r *ChatRepositoryImpl) ListRecentExchanges(ctx context.Context, conversationId string, limit int) ([]*entity.ChatExchange, error) {
	if limit <= 0 {
		return []*entity.ChatExchange{}, nil
	}
	var exchanges []*entity.ChatExchange
	err := r.db.WithContext(ctx).Model(&model.ChatExchange{}).
		Where("conversation_id = ?", conversationId).
		Order("create_time DESC, id DESC").
		Limit(limit).Find(&exchanges).Error
	if err != nil {
		return nil, err
	}
	slices.Reverse(exchanges)
	return exchanges, nil
}

// upsertDialogue 创建或更新会话
func (r *ChatRepositoryImpl) upsertDialogue(ctx context.Context, dialogue *entity.ChatDialogue) error {
	var chatDialogue *model.ChatDialogue
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", dialogue.ConversationId).
		Order("id DESC").
		First(&chatDialogue).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		chatDialogue = convert.ToChatDialogueModel(dialogue)
		return r.db.WithContext(ctx).Create(chatDialogue).Error
	}

	// Check if update needed
	needUpdate := chatDialogue.SessionStatus != dialogue.SessionStatus ||
		chatDialogue.ChatMode != dialogue.ChatMode ||
		chatDialogue.SelectedDocumentId != dialogue.SelectedDocumentId ||
		chatDialogue.SelectedDocumentName != dialogue.SelectedDocumentName

	if needUpdate {
		updates := map[string]interface{}{
			"session_status":         dialogue.SessionStatus,
			"chat_mode":              dialogue.ChatMode,
			"selected_document_id":   dialogue.SelectedDocumentId,
			"selected_document_name": dialogue.SelectedDocumentName,
		}
		return r.db.WithContext(ctx).Model(&model.ChatDialogue{}).
			Where("id = ?", chatDialogue.ID).
			Updates(updates).Error
	}
	return nil
}

// RefreshSessionScope 刷新会话范围（更新会话状态、模式、文档选择）
func (r *ChatRepositoryImpl) RefreshSessionScope(ctx context.Context, dialogue *entity.ChatDialogue) error {
	dialogue.SessionStatus = vo.ChatSessionStatusRunning
	return r.upsertDialogue(ctx, dialogue)
}

// SelectDialogue 获取会话
func (r *ChatRepositoryImpl) SelectDialogue(ctx context.Context, conversationId string) (*entity.ChatDialogue, error) {
	var dialogue *entity.ChatDialogue
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationId).
		Order("id DESC").
		First(dialogue).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return dialogue, nil
}

// ListDialogues 列出所有会话
func (r *ChatRepositoryImpl) ListDialogues(ctx context.Context) ([]*entity.ChatDialogue, error) {
	var dialogues []*entity.ChatDialogue
	err := r.db.WithContext(ctx).
		Order("update_time DESC, id DESC").
		Find(&dialogues).Error
	if err != nil {
		return nil, err
	}
	return dialogues, nil
}

// ListDialoguePage 分页查询会话
func (r *ChatRepositoryImpl) ListDialoguePage(ctx context.Context, pageNo, pageSize int, keyword string, chatMode, latestTurnStatus int) ([]*entity.ChatDialogue, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.ChatDialogue{}).Scopes(utils.Paginate(pageNo, pageSize))

	if chatMode > 0 {
		query = query.Where("chat_mode = ?", chatMode)
	}

	if keyword != "" {
		keyword = strings.TrimSpace(keyword)
		query = query.Where("(conversation_id LIKE ? OR selected_document_name LIKE ?)",
			"%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var dialogues []*entity.ChatDialogue
	if err = query.Order("update_time DESC, id DESC").Find(&dialogues).Error; err != nil {
		return nil, 0, err
	}
	return dialogues, total, nil
}

// DeleteSession 删除会话及所有交换记录
func (r *ChatRepositoryImpl) DeleteSession(ctx context.Context, conversationId string) (int64, int64, error) {
	var exchangeCount, dialogueCount int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := r.db.WithContext(ctx).Where("conversation_id = ?", conversationId).Delete(&model.ChatExchange{})
		if res.Error != nil {
			return res.Error
		}
		exchangeCount = res.RowsAffected

		res = r.db.WithContext(ctx).Where("conversation_id = ?", conversationId).Delete(&model.ChatDialogue{})
		dialogueCount = res.RowsAffected
		return res.Error
	})

	return dialogueCount, exchangeCount, err
}
