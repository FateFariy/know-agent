package trigger

import (
	"github.com/google/wire"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/api/document"
	"github.com/swiftbit/know-agent/internal/trigger/handler"
)

var ProviderSet = wire.NewSet(
	handler.NewDocumentService,
	wire.Bind(new(document.HTTPServer), new(*handler.DocumentService)),
	handler.NewChatService,
	wire.Bind(new(chat.HTTPServer), new(*handler.ChatService)),
)
