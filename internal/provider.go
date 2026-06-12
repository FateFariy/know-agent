package internal

import (
	"github.com/google/wire"

	"github.com/swiftbit/know-agent/internal/domain"
	"github.com/swiftbit/know-agent/internal/infrastructure"
	"github.com/swiftbit/know-agent/internal/server"
	"github.com/swiftbit/know-agent/internal/svc"
	"github.com/swiftbit/know-agent/internal/trigger"
)

var ProviderSet = wire.NewSet(
	domain.ProviderSet,
	infrastructure.ProviderSet,
	trigger.ProviderSet,
	server.ProviderSet,
	svc.ProviderSet,
)
