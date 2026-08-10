package domain

import (
	"github.com/google/wire"

	"github.com/swiftbit/know-agent/internal/domain/chat"
	"github.com/swiftbit/know-agent/internal/domain/document"
	"github.com/swiftbit/know-agent/internal/domain/knowledge"
)

var ProviderSet = wire.NewSet(
	chat.ProviderSet,
	document.ProviderSet,
	knowledge.ProviderSet,
)
