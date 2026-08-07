package save

import "context"

type ProfileGeneratePhase struct {
}

func NewProfileGeneratePhase() *ProfileGeneratePhase {
	return &ProfileGeneratePhase{}
}

func (p *ProfileGeneratePhase) Name() string {
	return "文档属性阶段"
}

func (p *ProfileGeneratePhase) Execute(ctx context.Context, saveCtx *Context) error {
	return nil
}
