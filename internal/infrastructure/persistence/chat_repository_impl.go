package persistence

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/slice"
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
	*transactionManager
}

func NewChatRepository(svcCtx *svc.ServiceContext) *ChatRepositoryImpl {
	return &ChatRepositoryImpl{
		transactionManager: &transactionManager{db: svcCtx.Db},
	}
}

// ========== 对话轮次（Exchange）相关 ==========

// InsertExchange 插入对话记录
func (r *ChatRepositoryImpl) InsertExchange(ctx context.Context, exchange *entity.ChatExchange) error {
	return r.dbWithContext(ctx).Create(convert.ToChatExchangeModel(exchange)).Error
}

// StartExchange 创建对话记录
func (r *ChatRepositoryImpl) StartExchange(ctx context.Context, dialogue *entity.ChatDialogue) (*entity.ChatExchange, error) {
	chatExchange := &entity.ChatExchange{
		ID:             utils.GetSnowflakeNextID(),
		ConversationId: dialogue.ConversationId,
		Question:       dialogue.Question,
		TurnStatus:     vo.ChatTurnStatusRunning,
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
	}
	return chatExchange, r.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dialogue.SessionStatus = vo.ChatSessionStatusRunning
		if err := r.UpsertDialogue(ctx, dialogue); err != nil {
			return err
		}
		return r.dbWithContext(ctx).Create(convert.ToChatExchangeModel(chatExchange)).Error
	})
}

// UpdateExchangeById 更新对话记录
func (r *ChatRepositoryImpl) UpdateExchangeById(ctx context.Context, exchange *entity.ChatExchange) error {
	return r.dbWithContext(ctx).Model(&model.ChatExchange{}).
		Where("id = ?", exchange.ID).
		Updates(convert.ToChatExchangeModel(exchange)).Error
}

// ListExchanges 列出对话的所有交换记录
func (r *ChatRepositoryImpl) ListExchanges(ctx context.Context, conversationId string) ([]*entity.ChatExchange, error) {
	var exchanges []*entity.ChatExchange
	err := r.dbWithContext(ctx).
		Model(&model.ChatExchange{}).
		Where("conversation_id = ?", conversationId).
		Order("create_time ASC, id ASC").
		Find(&exchanges).Error
	if err != nil {
		return nil, err
	}
	return exchanges, nil
}

// ListExchangesAfter 列出某个记录之后的记录
func (r *ChatRepositoryImpl) ListExchangesAfter(ctx context.Context, conversationId string, afterExchangeId int64) ([]*entity.ChatExchange, error) {
	var exchanges []*entity.ChatExchange
	query := r.dbWithContext(ctx).Model(&model.ChatExchange{}).Where("conversation_id = ?", conversationId)
	if afterExchangeId > -1 {
		query = query.Where("id > ?", afterExchangeId)
	}
	if err := query.Order("create_time ASC, id ASC").Find(&exchanges).Error; err != nil {
		return nil, err
	}
	return exchanges, nil
}

// ListRecentExchanges 列出最近的记录
func (r *ChatRepositoryImpl) ListRecentExchanges(ctx context.Context, conversationId string, limit int) ([]*entity.ChatExchange, error) {
	if limit <= 0 {
		return nil, nil
	}
	var exchanges []*entity.ChatExchange
	err := r.dbWithContext(ctx).Model(&model.ChatExchange{}).
		Where("conversation_id = ?", conversationId).
		Order("create_time DESC, id DESC").
		Limit(limit).Find(&exchanges).Error
	if err != nil {
		return nil, err
	}
	slices.Reverse(exchanges)
	return exchanges, nil
}

// ========== 会话（Dialogue）相关 ==========

// UpdateDialogueByConversationId 根据对话ID更新对话记录
func (r *ChatRepositoryImpl) UpdateDialogueByConversationId(ctx context.Context, dialogue *entity.ChatDialogue) error {
	return r.dbWithContext(ctx).Model(&model.ChatDialogue{}).
		Where("conversation_id = ?", dialogue.ConversationId).
		Updates(convert.ToChatDialogueModel(dialogue)).Error
}

// UpsertDialogue 创建或更新会话
func (r *ChatRepositoryImpl) UpsertDialogue(ctx context.Context, dialogue *entity.ChatDialogue) error {
	var chatDialogue *model.ChatDialogue
	err := r.dbWithContext(ctx).
		Where("conversation_id = ?", dialogue.ConversationId).
		Order("id DESC").
		First(&chatDialogue).Error

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		chatDialogue = convert.ToChatDialogueModel(dialogue)
		return r.dbWithContext(ctx).Create(chatDialogue).Error
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
		return r.dbWithContext(ctx).Model(&model.ChatDialogue{}).
			Where("id = ?", chatDialogue.ID).
			Updates(updates).Error
	}
	return nil
}

// RefreshSessionScope 刷新会话范围（更新会话状态、模式、文档选择）
func (r *ChatRepositoryImpl) RefreshSessionScope(ctx context.Context, dialogue *entity.ChatDialogue) error {
	dialogue.SessionStatus = vo.ChatSessionStatusRunning
	return r.UpsertDialogue(ctx, dialogue)
}

// ========== 会话归档与查询 ==========

// SelectSessionRecord 获取会话
func (r *ChatRepositoryImpl) SelectSessionRecord(ctx context.Context, conversationId string) (*vo.ConversationArchiveRecord, error) {
	dialogue := &entity.ChatDialogue{}
	err := r.dbWithContext(ctx).
		Where("conversation_id = ?", conversationId).
		Order("id DESC").
		First(dialogue).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var chatExchanges []*entity.ChatExchange
	if err = r.dbWithContext(ctx).Model(&model.ChatExchange{}).
		Where("conversation_id = ?", conversationId).
		Order("create_time ASC, id ASC").Find(&chatExchanges).Error; err != nil {
		return nil, err
	}

	return r.toChatArchiveRecord(dialogue, chatExchanges), nil
}

// ListSessionRecordPage 列出会话记录分页
func (r *ChatRepositoryImpl) ListSessionRecordPage(ctx context.Context, keyword string, pageNo, pageSize, chatMode, latestTurnStatus int) ([]*vo.ConversationArchiveRecord, int64, error) {
	query := r.buildListDialoguePageQuery(ctx, keyword, chatMode, latestTurnStatus)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var dialogues []*entity.ChatDialogue
	if err := query.Scopes(utils.Paginate(pageNo, pageSize)).Find(&dialogues).Error; err != nil {
		return nil, 0, err
	}
	conversationIds := slice.Map(dialogues, func(index int, item *entity.ChatDialogue) string {
		return item.ConversationId
	})
	chatExchangesMap, err := r.selectLatestExchangesByConversationIds(ctx, conversationIds)
	if err != nil {
		return nil, 0, err
	}
	records := slice.Map(dialogues, func(index int, item *entity.ChatDialogue) *vo.ConversationArchiveRecord {
		return r.toChatArchiveRecord(item, []*entity.ChatExchange{chatExchangesMap[item.ConversationId]})
	})
	return records, total, nil
}

// DeleteSession 删除会话及所有记录
func (r *ChatRepositoryImpl) DeleteSession(ctx context.Context, conversationId string) (int64, int64, error) {
	var exchangeCount, dialogueCount int64
	err := r.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := r.dbWithContext(ctx).Where("conversation_id = ?", conversationId).Delete(&model.ChatExchange{})
		if res.Error != nil {
			return res.Error
		}
		exchangeCount = res.RowsAffected

		res = r.dbWithContext(ctx).Where("conversation_id = ?", conversationId).Delete(&model.ChatDialogue{})
		dialogueCount = res.RowsAffected
		return res.Error
	})

	return dialogueCount, exchangeCount, err
}

// ========== 会话记忆摘要相关 ==========

// SelectMemorySummary 查询会话记忆摘要
func (r *ChatRepositoryImpl) SelectMemorySummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error) {
	var summary *entity.ChatMemorySummary
	err := r.dbWithContext(ctx).Model(&model.ChatMemorySummary{}).
		Where("conversation_id = ?", conversationId).
		Order("id DESC").
		First(summary).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return summary, nil
}

// InsertMemorySummary 插入会话记忆摘要
func (r *ChatRepositoryImpl) InsertMemorySummary(ctx context.Context, summary *entity.ChatMemorySummary) error {
	return r.dbWithContext(ctx).Create(convert.ToChatMemorySummaryModel(summary)).Error
}

// UpdateMemorySummaryById 更新会话记忆摘要
func (r *ChatRepositoryImpl) UpdateMemorySummaryById(ctx context.Context, summary *entity.ChatMemorySummary) error {
	fields := "covered_exchange_id,covered_exchange_count,compression_count,summary_version,summary_text,summary_json,last_source_update_time,update_time"
	return r.dbWithContext(ctx).Select(fields).Where("id = ?", summary.ID).
		Updates(convert.ToChatMemorySummaryModel(summary)).Error
}

// DeleteMemorySummary 删除会话记忆摘要
func (r *ChatRepositoryImpl) DeleteMemorySummary(ctx context.Context, conversationId string) error {
	return r.dbWithContext(ctx).Where("conversation_id = ?", conversationId).Delete(&model.ChatMemorySummary{}).Error
}

// ========== 会话阶段追踪相关 ==========

// InsertStage 创建阶段记录
func (r *ChatRepositoryImpl) InsertStage(ctx context.Context, stage *entity.ChatExchangeTraceStage) error {
	return r.dbWithContext(ctx).Create(convert.ToChatExchangeTraceStageModel(stage)).Error
}

// UpdateStageById 更新阶段记录
func (r *ChatRepositoryImpl) UpdateStageById(ctx context.Context, stage *entity.ChatExchangeTraceStage) error {
	return r.dbWithContext(ctx).Model(&model.ChatExchangeTraceStage{}).
		Where("id = ?", stage.ID).
		Updates(stage).Error
}

// SelectStages 查询阶段记录
func (r *ChatRepositoryImpl) SelectStages(ctx context.Context, conversationId string, exchangeId int64) ([]*entity.ChatExchangeTraceStage, error) {
	var stages []*entity.ChatExchangeTraceStage
	if err := r.dbWithContext(ctx).
		Model(&model.ChatExchangeTraceStage{}).
		Where("conversation_id = ? AND exchange_id = ?", conversationId, exchangeId).
		Order("stage_order ASC, start_time ASC, id ASC").
		Find(&stages).Error; err != nil {
		return nil, err
	}
	return stages, nil
}

// DeleteStage 删除阶段记录
func (r *ChatRepositoryImpl) DeleteStage(ctx context.Context, conversationId string) error {
	return r.dbWithContext(ctx).
		Where("conversation_id = ?", conversationId).
		Delete(&model.ChatExchangeTraceStage{}).Error
}

// ========== 内部辅助方法 ==========

// buildListDialoguePageQuery 构建分页查询会话的查询条件
func (r *ChatRepositoryImpl) buildListDialoguePageQuery(ctx context.Context, keyword string, chatMode, latestTurnStatus int) *gorm.DB {
	query := r.dbWithContext(ctx).Model(&model.ChatDialogue{})

	if chatMode > 0 {
		query = query.Where("chat_mode = ?", chatMode)
	}

	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		likeKeyword := "%" + keyword + "%"
		subQuery := r.db.Session(&gorm.Session{NewDB: true, Context: ctx}).
			Table("chat_exchange AS e").
			Select("1").
			Where("conversation_id = e.conversation_id").
			Where("e.question LIKE ? OR e.answer LIKE ? OR e.error_message LIKE ?", likeKeyword, likeKeyword, likeKeyword)
		query = query.Where("(conversation_id LIKE ? OR selected_document_name LIKE ? OR EXISTS (?))", likeKeyword, likeKeyword, subQuery)
	}

	if latestTurnStatus > 1 {
		query.Where("session_status = ?", vo.ChatSessionStatusIdle)
		latestIdSubQuery := r.db.Session(&gorm.Session{NewDB: true, Context: ctx}).
			Table("chat_exchange AS latest").
			Select("latest.id").
			Where("latest.conversation_id = conversation_id").
			Order("latest.create_time DESC, latest.id DESC").
			Limit(1)
		existsQuery := r.db.Session(&gorm.Session{NewDB: true, Context: ctx}).
			Table("chat_exchange AS e").
			Select("1").
			Where("conversation_id = e.conversation_id").
			Where("e.id = (?)", latestIdSubQuery).
			Where("e.turn_status = ?", latestTurnStatus)
		query.Where("EXISTS (?)", existsQuery)
	} else if latestTurnStatus > 0 {
		query.Where("session_status = ?", vo.ChatSessionStatusRunning)
	}

	return query.Order("update_time DESC, id DESC")
}

// selectLatestExchangesByConversationIds 根据会话ID列表获取最新的对话交换记录
func (r *ChatRepositoryImpl) selectLatestExchangesByConversationIds(ctx context.Context, conversationIds []string) (map[string]*entity.ChatExchange, error) {
	var chatExchanges []*entity.ChatExchange
	if err := r.dbWithContext(ctx).Model(&model.ChatExchange{}).
		Where("conversation_id IN ?", conversationIds).
		Order("creat_time DESC, id DESC").Find(&chatExchanges).Error; err != nil {
		return nil, err
	}
	return utils.SliceToMapBy(chatExchanges, func(item *entity.ChatExchange) (string, *entity.ChatExchange) {
		return item.ConversationId, item
	}), nil
}

// toChatArchiveRecord 转换为会话记录
func (r *ChatRepositoryImpl) toChatArchiveRecord(dialogue *entity.ChatDialogue, chatExchanges []*entity.ChatExchange) *vo.ConversationArchiveRecord {
	return &vo.ConversationArchiveRecord{
		ConversationId:       dialogue.ConversationId,
		ChatMode:             dialogue.ChatMode,
		Running:              dialogue.SessionStatus == vo.ChatSessionStatusRunning,
		SelectedDocumentId:   dialogue.SelectedDocumentId,
		SelectedDocumentName: dialogue.SelectedDocumentName,
		CreatedAt:            dialogue.CreateTime,
		UpdatedAt:            dialogue.UpdateTime,
		Exchanges:            chatExchanges,
	}
}
