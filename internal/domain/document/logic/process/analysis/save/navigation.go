package save

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

type NavigationUploadPhase struct {
	port *adapter.DocumentPort
}

func NewNavigationUploadPhase(port *adapter.DocumentPort) *NavigationUploadPhase {
	return &NavigationUploadPhase{
		port: port,
	}
}

func (n *NavigationUploadPhase) Name() string {
	return "导航上传阶段"
}

func (n *NavigationUploadPhase) Execute(ctx context.Context, saveCtx *Context) error {
	return nil
}
