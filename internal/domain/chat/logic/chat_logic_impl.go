package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/chat/support"
)

const (
	chatRunningLeasePrefix        = "chat:running:"
	chatRunningLeaseTTL           = 30 * time.Second
	chatRunningLeaseRenewInterval = 10 * time.Second
)

// ChatLogicImpl 聊天业务逻辑实现
type ChatLogicImpl struct {
	repo               adapter.ChatRepository
	streamEventBuilder *support.StreamEventBuilder
	runtimeRegistry    *ChatRuntimeRegistry
}

// NewChatLogic 创建聊天逻辑实例
func NewChatLogic(repo adapter.ChatRepository) *ChatLogicImpl {
	return &ChatLogicImpl{
		repo:               repo,
		streamEventBuilder: support.NewStreamEventBuilder(),
		runtimeRegistry:    NewChatRuntimeRegistry(),
	}
}

// ChatRuntimeRegistry 运行时会话注册表
type ChatRuntimeRegistry struct {
	tasks sync.Map
}

func NewChatRuntimeRegistry() *ChatRuntimeRegistry {
	return &ChatRuntimeRegistry{}
}

func (r *ChatRuntimeRegistry) Register(taskInfo *TaskInfo) bool {
	_, loaded := r.tasks.LoadOrStore(taskInfo.ConversationId, taskInfo)
	return !loaded
}

func (r *ChatRuntimeRegistry) Get(conversationId string) (*TaskInfo, bool) {
	task, ok := r.tasks.Load(conversationId)
	if !ok {
		return nil, false
	}
	return task.(*TaskInfo), true
}

func (r *ChatRuntimeRegistry) Remove(conversationId string, taskInfo *TaskInfo) {
	r.tasks.CompareAndDelete(conversationId, taskInfo)
}

// TaskInfo 任务信息
type TaskInfo struct {
	ConversationId      string
	ExchangeId          int64
	Question            string
	AnswerBuffer        strings.Builder
	FirstResponseTimeMs int64
	StartTime           int64
	Finalized           atomic.Bool
	EventMetadata       *support.StreamEventMetadata
	ThinkingSteps       []string
	References          []*support.SearchReference
	UsedTools           []string
}

func NewTaskInfo(conversationId string, exchangeId int64, question string) *TaskInfo {
	return &TaskInfo{
		ConversationId: conversationId,
		ExchangeId:     exchangeId,
		Question:       question,
		StartTime:      time.Now().UnixMilli(),
		EventMetadata: &support.StreamEventMetadata{
			ConversationId: conversationId,
			ExchangeId:     exchangeId,
		},
		ThinkingSteps: []string{},
		References:    []*support.SearchReference{},
		UsedTools:     []string{},
	}
}

// OpenConversationStream 打开会话流
func (c *ChatLogicImpl) OpenConversationStream(ctx context.Context, cmd *vo.ChatCommand) (<-chan string, error) {
	cmdJSON, _ := json.Marshal(cmd)
	logx.Infof("======request内容：%s", string(cmdJSON))

	conversationId := normalizeConversationId(cmd.ConversationId)

	stream := make(chan string)

	go func() {
		defer close(stream)

		taskInfo := NewTaskInfo(conversationId, cmd.ConversationId, question)

		if !c.runtimeRegistry.Register(taskInfo) {
			errMsg := c.streamEventBuilder.ErrorWithMetadata("该会话当前正在执行中，请稍后再试", taskInfo.EventMetadata)
			stream <- errMsg
			return
		}

		defer func() {
			c.runtimeRegistry.Remove(conversationId, taskInfo)
		}()

		c.executeConversation(taskInfo, stream)
	}()

	return stream, nil
}

func (c *ChatLogicImpl) executeConversation(taskInfo *TaskInfo, stream chan<- string) {
	thinkingMsg := c.streamEventBuilder.ThinkingWithMetadata("正在分析问题上下文。", taskInfo.EventMetadata)
	stream <- thinkingMsg

	thinkingMsg = c.streamEventBuilder.ThinkingWithMetadata("正在检索相关知识...", taskInfo.EventMetadata)
	stream <- thinkingMsg

	time.Sleep(500 * time.Millisecond)

	answer := "这是模拟的回答内容。您的问题是：" + taskInfo.Question
	for i := 0; i < len(answer); i += 5 {
		end := i + 5
		if end > len(answer) {
			end = len(answer)
		}
		textMsg := c.streamEventBuilder.TextWithMetadata(answer[i:end], taskInfo.EventMetadata)
		stream <- textMsg
		taskInfo.AnswerBuffer.WriteString(answer[i:end])
		time.Sleep(100 * time.Millisecond)
	}

	references := []*support.SearchReference{
		{
			ReferenceId:  "ref-001",
			SourceType:   "document",
			Title:        "相关文档标题",
			Snippet:      "这是文档中的相关片段内容...",
			DocumentId:   1,
			DocumentName: "示例文档.pdf",
			Score:        0.85,
		},
	}
	refMsg := c.streamEventBuilder.ReferencesWithMetadata(references, taskInfo.EventMetadata)
	stream <- refMsg

	recommendations := []string{
		"您想了解更多相关信息吗？",
		"是否需要深入探讨这个话题？",
	}
	recMsg := c.streamEventBuilder.RecommendationsWithMetadata(recommendations, taskInfo.EventMetadata)
	stream <- recMsg
}

// ListKnowledgeDocumentOptions 获取知识文档选项列表
func (c *ChatLogicImpl) ListKnowledgeDocumentOptions(ctx context.Context) ([]*chat.KnowledgeDocumentOptionResp, error) {
	return []*chat.KnowledgeDocumentOptionResp{
		{DocumentId: 1, DocumentName: "产品手册.pdf"},
		{DocumentId: 2, DocumentName: "技术文档.docx"},
		{DocumentId: 3, DocumentName: "用户指南.txt"},
	}, nil
}

// StopConversation 停止会话
func (c *ChatLogicImpl) StopConversation(ctx context.Context, conversationId string) (*chat.ConversationStopResp, error) {
	taskInfo, ok := c.runtimeRegistry.Get(conversationId)
	if !ok {
		return &chat.ConversationStopResp{Success: false}, fmt.Errorf("没有找到正在执行的会话")
	}

	return &chat.ConversationStopResp{Success: true}, nil
}

// GetSession 获取会话详情
func (c *ChatLogicImpl) GetSession(ctx context.Context, conversationId string) (*chat.ConversationSessionResp, error) {
	return &chat.ConversationSessionResp{
		ConversationId: conversationId,
		Title:          "测试会话",
		LatestMessage:  "你好，有什么可以帮助你的？",
		CreateTime:     time.Now().Format(time.RFC3339),
		UpdateTime:     time.Now().Format(time.RFC3339),
	}, nil
}

// GetExchangeDetail 获取对话详情
func (c *ChatLogicImpl) GetExchangeDetail(ctx context.Context, conversationId, exchangeId string) (*chat.ConversationExchangeDetailResp, error) {
	return &chat.ConversationExchangeDetailResp{
		ExchangeId:     exchangeId,
		ConversationId: conversationId,
		UserMessage:    "你好",
		AgentMessage:   "你好，有什么可以帮助你的？",
		CreateTime:     time.Now().Format(time.RFC3339),
	}, nil
}

// ListSessions 获取会话列表
func (c *ChatLogicImpl) ListSessions(ctx context.Context, req *chat.ConversationSessionListReq) (*chat.ConversationSessionListResp, error) {
	pageNo := req.PageNo
	if pageNo <= 0 {
		pageNo = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	return &chat.ConversationSessionListResp{
		PageNo:   pageNo,
		PageSize: pageSize,
		Total:    100,
		Records: []*chat.ConversationSessionResp{
			{
				ConversationId: uuid.New().String(),
				Title:          "会话1",
				LatestMessage:  "消息内容1",
				CreateTime:     time.Now().Format(time.RFC3339),
				UpdateTime:     time.Now().Format(time.RFC3339),
			},
			{
				ConversationId: uuid.New().String(),
				Title:          "会话2",
				LatestMessage:  "消息内容2",
				CreateTime:     time.Now().Format(time.RFC3339),
				UpdateTime:     time.Now().Format(time.RFC3339),
			},
		},
	}, nil
}

// ResetConversation 重置会话
func (c *ChatLogicImpl) ResetConversation(ctx context.Context, conversationId string) (*chat.ConversationResetResp, error) {
	return &chat.ConversationResetResp{Success: true}, nil
}

// RebuildConversationSummary 重建会话摘要
func (c *ChatLogicImpl) RebuildConversationSummary(ctx context.Context, conversationId string) (*chat.ConversationMemorySummaryResp, error) {
	return &chat.ConversationMemorySummaryResp{
		ConversationId: conversationId,
		Summary:        "这是会话的摘要内容",
	}, nil
}

// GetRetrievalResults 获取检索结果
func (c *ChatLogicImpl) GetRetrievalResults(ctx context.Context, conversationId string, exchangeId int64) ([]*chat.RetrievalResultResp, error) {
	return []*chat.RetrievalResultResp{}, nil
}

// GetChannelExecutions 获取渠道执行结果
func (c *ChatLogicImpl) GetChannelExecutions(ctx context.Context, conversationId string, exchangeId int64) ([]*chat.ChannelExecutionResp, error) {
	return []*chat.ChannelExecutionResp{}, nil
}

// GetStageBenchmarks 获取阶段基准
func (c *ChatLogicImpl) GetStageBenchmarks(ctx context.Context) ([]*chat.StageBenchmarkResp, error) {
	return []*chat.StageBenchmarkResp{}, nil
}

func normalizeConversationId(conversationId string) string {
	conversationId = strings.TrimSpace(conversationId)
	if conversationId != "" {
		return conversationId
	}
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
