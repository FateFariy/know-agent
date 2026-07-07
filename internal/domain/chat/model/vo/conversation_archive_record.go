package vo

import (
	"time"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
)

type ConversationArchiveRecord struct {
	ConversationId         string                    // 会话ID
	Running                bool                      // 是否正在运行
	ChatMode               int                       // 聊天模式
	SelectedDocumentId     int64                     // 选中的文档ID
	SelectedDocumentName   string                    // 选中的文档名称
	CreatedTime            time.Time                 // 创建时间
	UpdatedTime            time.Time                 // 更新时间
	Exchanges              []*entity.ChatExchange    // 交互记录列表
	CheckpointCount        int                       // 检查点数量
	MessageCount           int                       // 消息总数
	LatestUserMessage      string                    // 最新用户提问
	LatestAssistantMessage string                    // 模型最新回复
	LatestExchangeId       int64                     // 最新一轮交互ID
	LatestTurnStatus       string                    // 本轮交互状态
	LatestTurnErrorMessage string                    // 本轮交互错误信息
	MemorySummary          *entity.ChatMemorySummary // 会话记忆摘要
}

// FillSummaryFields 根据 Exchanges 列表计算并填充会话归档记录的摘要字段
func (c *ConversationArchiveRecord) FillSummaryFields() {
	for _, exchange := range c.Exchanges {
		if strutil.IsNotBlank(exchange.Question) {
			c.MessageCount++
		}
		if strutil.IsNotBlank(exchange.Answer) {
			c.MessageCount++
		}
		if strutil.IsBlank(c.LatestUserMessage) {
			c.LatestUserMessage = exchange.Question
		}
		if strutil.IsBlank(c.LatestAssistantMessage) {
			c.LatestAssistantMessage = exchange.Answer
		}
	}

	// 步骤 2：取最后一条 exchange 作为最新轮次，填充其 ID、状态与错误信息
	lastExchange := c.Exchanges[len(c.Exchanges)-1]
	c.LatestExchangeId = lastExchange.ID
	c.LatestTurnStatus = ChatTurnStatusName(lastExchange.TurnStatus)
	c.LatestTurnErrorMessage = lastExchange.ErrorMessage
}
