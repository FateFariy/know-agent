package save

import (
	"context"
)

// NavigationUploadPhase 导航产物同步阶段
// todo graph、table、raptor 相关代码暂时跳过
type NavigationUploadPhase struct{}

func NewNavigationUploadPhase() *NavigationUploadPhase {
	return &NavigationUploadPhase{}
}

func (n *NavigationUploadPhase) Name() string {
	return "导航产物同步阶段"
}

func (n *NavigationUploadPhase) Execute(ctx context.Context, saveCtx *Context) error {
	// TODO: 待实现 - 导航ES索引同步和结构图投影同步
	// 当前graph、table、raptor相关代码暂时跳过
	return nil
}
