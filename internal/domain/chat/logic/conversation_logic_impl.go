package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/memory"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

const (
	chatRunningLeasePrefix = "conversation:running:"
)

// ConversationLogicImpl 聊天业务逻辑实现
type ConversationLogicImpl struct {
	repo            adapter.ChatRepository
	baseGateway     adapter.KnowledgeBaseGateway
	memoryManager   memory.SessionMemoryManager
	distributedLock adapter.DistributedLock
	checkPointStore adapter.CheckPointStore
	chain           *conversation.Chain
}

var _ ConversationLogic = (*ConversationLogicImpl)(nil)

// NewConversationLogicImpl 创建聊天逻辑实例
func NewConversationLogicImpl(repo adapter.ChatRepository, baseGateway adapter.KnowledgeBaseGateway,
	memoryManager memory.SessionMemoryManager, distributedLock adapter.DistributedLock,
	checkPointStore adapter.CheckPointStore, chain *conversation.Chain) *ConversationLogicImpl {
	return &ConversationLogicImpl{
		repo:            repo,
		baseGateway:     baseGateway,
		memoryManager:   memoryManager,
		distributedLock: distributedLock,
		checkPointStore: checkPointStore,
		chain:           chain,
	}
}

// OpenConversationStream 打开会话流
func (c *ConversationLogicImpl) OpenConversationStream(ctx context.Context, sink adapter.Sink, cmd *vo.ChatCommand) (err error) {
	cmdJSON, _ := json.Marshal(cmd)
	logx.Infof("====== request 内容：%s", string(cmdJSON))

	leaseKey := chatRunningLeasePrefix + cmd.ConversationId
	defer func() {
		if err != nil {
			c.releaseConversationLock(leaseKey)
			logx.Errorf("会话启动失败, conversationId=%s, question=%s, err=%v",
				cmd.ConversationId, cmd.Question, err)
			if err = sink.Error(err.Error(), cmd.ConversationId, 0); err != nil {
				return
			}
		}
	}()

	// 获取分布式租约
	if err = c.distributedLock.TryLock(ctx, leaseKey); err != nil {
		return fmt.Errorf("该会话当前正在执行中，请稍后再试")
	}

	// 构建对话上下文
	convCtx, err := c.buildConversationContext(ctx, cmd, sink)
	if err != nil {
		return err
	}
	// 运行会话链
	return c.chain.Run(ctx, convCtx)
}

// StopConversation 停止会话
func (c *ConversationLogicImpl) StopConversation(ctx context.Context, conversationId string) (bool, string, error) {
	responseMessage, stopped := c.chain.Stop(ctx, conversationId, "用户已停止生成")
	return stopped, responseMessage, nil
}

// GetSessionDetail 获取会话详情
func (c *ConversationLogicImpl) GetSessionDetail(ctx context.Context, conversationId string) (*aggregate.ConversationArchiveRecord, error) {
	record, err := c.repo.SelectSessionRecord(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	record.MemorySummary, err = c.memoryManager.GetConversationSummary(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	record.FillSummaryFields()

	return record, nil
}

// GetExchangeDetail 获取对话详情（含阶段追踪）
func (c *ConversationLogicImpl) GetExchangeDetail(ctx context.Context, conversationId string, exchangeId int64) (*entity.ChatExchange, []*entity.ChatExchangeTraceStage, error) {
	exchange, err := c.repo.SelectExchangeById(ctx, exchangeId)
	if err != nil {
		return nil, nil, err
	}

	stages, err := c.repo.SelectStages(ctx, conversationId, exchangeId)
	if err != nil {
		return nil, nil, err
	}
	return exchange, stages, nil
}

// ListSessions 获取会话列表（分页）
func (c *ConversationLogicImpl) ListSessions(ctx context.Context, pageNo, pageSize, chatMode, latestTurnStatus int, keyword string) ([]*aggregate.ConversationArchiveRecord, int64, error) {
	records, total, err := c.repo.ListSessionRecordPage(ctx, pageNo, pageSize, chatMode, latestTurnStatus, keyword)
	if err != nil {
		return nil, 0, err
	}
	for _, record := range records {
		record.FillSummaryFields()
		record.Exchanges = nil
	}

	return records, total, nil
}

// ResetConversation 重置会话：停止并清除所有相关落库数据
func (c *ConversationLogicImpl) ResetConversation(ctx context.Context, conversationId string) (*vo.ConversationReset, error) {
	// 停止正在运行的会话
	_, stoped := c.chain.Stop(ctx, conversationId, "会话被重置")

	var dialogueCount, exchangeCount int64
	var err error

	// 删除记忆摘要
	if err = c.memoryManager.DeleteConversationSummary(ctx, conversationId); err != nil {
		return nil, err
	}
	fn := func(txCtx context.Context) error {
		// 删除会话及关联 exchange
		dialogueCount, exchangeCount, err = c.repo.DeleteSession(ctx, conversationId)
		if err != nil {
			return err
		}
		// 删除阶段追踪
		if err = c.repo.DeleteStage(txCtx, conversationId); err != nil {
			return err
		}
		// 删除检索结果
		if err = c.repo.DeleteRetrievalResultsByConversationId(txCtx, conversationId); err != nil {
			return err
		}
		// 删除渠道执行记录
		return c.repo.DeleteChannelExecutionsByConversationId(txCtx, conversationId)
	}
	if err = c.repo.Do(ctx, fn); err != nil {
		return nil, err
	}
	// 删除检查点
	count, err := c.checkPointStore.Delete(ctx, conversationId)
	if err != nil {
		return nil, err
	}
	return &vo.ConversationReset{
		ConversationId:         conversationId,
		StoppedRunningTask:     stoped,
		RemovedDialogueCount:   int(dialogueCount),
		RemovedExchangeCount:   int(exchangeCount),
		RemovedCheckpointCount: count,
		Message:                "会话已重置",
	}, nil
}

// RebuildConversationSummary 重建会话摘要
func (c *ConversationLogicImpl) RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error) {
	summary, err := c.memoryManager.RebuildConversationSummary(ctx, conversationId)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

// GetRetrievalResults 获取检索结果
func (c *ConversationLogicImpl) GetRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*entity.ChatRetrievalResult, error) {
	return c.repo.SelectRetrievalResults(ctx, conversationId, exchangeId)
}

// GetChannelExecutions 获取渠道执行结果
func (c *ConversationLogicImpl) GetChannelExecutions(ctx context.Context, conversationId string, exchangeId int64) ([]*entity.ChatChannelExecution, error) {
	return c.repo.SelectChannelExecutions(ctx, conversationId, exchangeId)
}

// buildConversationContext 构建对话上下文：从 ChatCommand 提取问题与对话上下文，生成规范化的对话上下文
//
// 执行流程：
//  1. 规范化会话 ID：优先使用传入的 conversationId；为空时生成无连字符的 UUID
//  2. 构建初始计划（问题、会话 ID、聊天模式），并填充当前时间
//  3. 若命令中指定了文档 ID，则从可检索文档列表中查找并写入文档名与索引任务 ID；缺失则返回错误
func (c *ConversationLogicImpl) buildConversationContext(ctx context.Context, cmd *vo.ChatCommand, sink adapter.Sink) (*conversation.Context, error) {
	// 规范化会话 ID —— 空值时生成 UUID 作为新会话标识
	conversationId := utils.Trim(cmd.ConversationId)
	if conversationId == "" {
		conversationId = utils.GenerateUUIDWithoutHyphen()
	}

	selectionSnapshot, err := c.baseGateway.DetermineKnowledgeScope(ctx, cmd.ChatMode, cmd.KnowledgeBaseSelectionMode, cmd.SelectedKnowledgeBaseIds)
	if err != nil {
		return nil, err
	}
	// 构建启动计划，填充问题、会话 ID、聊天模式
	convCtx := &conversation.Context{
		ConversationId:                 conversationId,
		Question:                       cmd.Question,
		ChatMode:                       enum.ToChatQueryMode(cmd.ChatMode),
		KnowledgeBaseSelectionSnapshot: selectionSnapshot,
		Sink:                           sink,
	}
	if selectionSnapshot.SelectionModeName() == enum.KbSelectionModeNone {
		convCtx.ChatMode = enum.ChatQueryModeOpenChat
	}

	// 当指定文档 ID 时，验证该文档存在，并写入文档名与索引任务 ID
	if cmd.SelectedDocumentId != 0 {
		documents := utils.Ternary(cmd.KnowledgeBaseSelectionMode == enum.KbSelectionModeNone, nil, selectionSnapshot.AllowedDocuments)
		index := slices.IndexFunc(documents, func(doc *vo.DocumentMetadata) bool {
			return doc.DocumentId == cmd.SelectedDocumentId
		})
		// 指定的文档不存在或索引不可用，直接返回错误
		if index == -1 {
			return nil, errorx.ErrDocumentIndexUnavailable.Format(cmd.SelectedDocumentId)
		}
		convCtx.SelectedDocumentId = documents[index].DocumentId
		convCtx.SelectedDocumentName = documents[index].DocumentName
		convCtx.SelectedTaskId = documents[index].LastIndexTaskId
	}
	return convCtx, nil
}

// releaseConversationLock 释放会话运行锁
func (c *ConversationLogicImpl) releaseConversationLock(leaseKey string) {
	err := c.distributedLock.Unlock(leaseKey)
	if err != nil && !errors.Is(err, errorx.ErrDistributedLockNotFound) {
		logx.Warnf("会话分布式锁释放失败, leaseKey=%s, err=%v", leaseKey, err)
	}
}
