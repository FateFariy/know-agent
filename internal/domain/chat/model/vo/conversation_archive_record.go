package vo

import (
	"time"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
)

type ConversationArchiveRecord struct {
	ConversationId       string
	Running              bool
	ChatMode             int
	SelectedDocumentId   int64
	SelectedDocumentName string
	CreatedTime          time.Time
	UpdatedTime          time.Time
	Exchanges            []*entity.ChatExchange
}
