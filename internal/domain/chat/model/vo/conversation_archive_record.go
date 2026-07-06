package vo

import (
	"time"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
)

type ConversationArchiveRecord struct {
	ConversationId         string
	Running                bool
	ChatMode               int
	SelectedDocumentId     int64
	SelectedDocumentName   string
	CreatedTime            time.Time
	UpdatedTime            time.Time
	Exchanges              []*entity.ChatExchange
	CheckpointCount        int32                          `json:"checkpointCount"`        // 检查点数量
	MessageCount           int32                          `json:"messageCount"`           // 消息总数
	LatestUserMessage      string                         `json:"latestUserMessage"`      // 最新用户提问
	LatestAssistantMessage string                         `json:"latestAssistantMessage"` // 模型最新回复
	LatestExchangeId       int64                          `json:"latestExchangeId"`       // 最新一轮交互ID
	LatestTurnStatus       string                         `json:"latestTurnStatus"`       // 本轮交互状态
	LatestTurnErrorMessage string                         `json:"latestTurnErrorMessage"` // 本轮交互错误信息
	MemorySummary          *ConversationMemorySummaryResp `json:"memorySummary"`          // 会话记忆摘要
}
