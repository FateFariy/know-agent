package save

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// ProfileGenerator 文档画像生成器
type ProfileGenerator interface {
	Generate(ctx context.Context, documentId int64, analysisResult *aggregate.AnalysisResult, structureNodes []*entity.StructureNode) (*entity.DocumentProfile, error)
}

// ProfileGeneratePhase 文档画像生成阶段
type ProfileGeneratePhase struct {
	repo adapter.DocumentRepository
	gen  ProfileGenerator
}

// NewProfileGeneratePhase 创建文档画像生成阶段
func NewProfileGeneratePhase(repo adapter.DocumentRepository, gen ProfileGenerator) *ProfileGeneratePhase {
	return &ProfileGeneratePhase{repo: repo}
}

func (p *ProfileGeneratePhase) Name() string {
	return "文档画像生成阶段"
}

func (p *ProfileGeneratePhase) Execute(ctx context.Context, saveCtx *Context) error {
	if saveCtx == nil || saveCtx.DocumentId == 0 {
		return nil
	}

	profile, err := p.gen.Generate(ctx, saveCtx.DocumentId, saveCtx.AnalysisResult, saveCtx.StructureNodes)
	if err != nil {
		return err
	}
	return p.repo.SaveProfile(ctx, profile)
}
