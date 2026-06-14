//go:build wireinject

package main

import (
	"github.com/google/wire"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal"

	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/server"
)

//go:generate wire gen ./wire.go
func WireApp(c config.Config) *server.Server {
	panic(wire.Build(
		provideCommonConfig,
		internal.ProviderSet,
	))
}

func provideCommonConfig(c config.Config) common.Config {
	return c
}
