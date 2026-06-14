package vo

import "time"

type ConversationArchiveRecord struct {
	ConversationId       string
	Running              bool
	ChatMode             ChatQueryMode
	SelectedDocumentId   int64
	SelectedDocumentName string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Exchanges            []*ConversationExchangeView
}
