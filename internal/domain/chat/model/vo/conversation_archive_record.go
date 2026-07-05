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
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Exchanges            []*entity.ChatExchange
}
