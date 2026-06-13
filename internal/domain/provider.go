package domain

import (
	"github.com/google/wire"

	chatLogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic"
)

var ProviderSet = wire.NewSet(
	logic.NewDocumentLifecycleLogicImpl,
	wire.Bind(new(logic.DocumentLifecycleLogic), new(*logic.LifecycleLogicImpl)),
	adapter.NewDocumentPort,
	chatLogic.NewChatLogic,
	wire.Bind(new(chatLogic.ChatLogic), new(*chatLogic.ChatLogicImpl)),
)
