package vo

import "time"

type StreamLaunchPlan struct {
	Question             string
	ConversationId       string
	ChatMode             ChatQueryMode
	SelectedDocumentId   int64
	SelectedDocumentName string
	SelectedTaskId       int64
	LeaseKey             string
	LeaseOwnerToken      string
	CurrentDate          time.Time
	CurrentDateText      string
}
