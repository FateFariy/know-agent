package logic

import (
	"context"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// ProfileLogicImpl 文档画像逻辑实现
type ProfileLogicImpl struct {
	repo    adapter.DocumentRepository
	storage adapter.Storage
	gen     process.ProfileGenerator
}

var _ ProfileLogic = (*ProfileLogicImpl)(nil)

// NewProfileLogicImpl 构造函数
func NewProfileLogicImpl(repo adapter.DocumentRepository, storage adapter.Storage, gen process.ProfileGenerator) *ProfileLogicImpl {
	return &ProfileLogicImpl{
		repo:    repo,
		storage: storage,
		gen:     gen,
	}
}

// GetAllProfiles 根据文档ID获取画像
func (p *ProfileLogicImpl) GetAllProfiles(ctx context.Context) ([]*entity.DocumentProfile, error) {
	return p.repo.SelectDocumentProfiles(ctx)
}

// GetProfileByDocumentId 根据文档ID获取画像
func (p *ProfileLogicImpl) GetProfileByDocumentId(ctx context.Context, documentId int64) (*entity.DocumentProfile, error) {
	return p.repo.SelectProfileByDocumentId(ctx, documentId)
}

// RegenerateProfile 重新生成文档画像
func (p *ProfileLogicImpl) RegenerateProfile(ctx context.Context, documentId int64) (*entity.DocumentProfile, error) {
	document, err := p.repo.SelectDocumentById(ctx, documentId)
	if err != nil {
		return nil, err
	}
	var parsedText string
	if utils.IsNotBlank(document.ParseTextPath) {
		parsedText, err = p.storage.DownloadText(ctx, document.ParseTextPath)
		if err != nil {
			return nil, err
		}
	}
	structureNodes, err := p.repo.SelectStructureNodeListByDocumentId(ctx, documentId)
	if err != nil {
		return nil, err
	}
	analysisResult := &aggregate.AnalysisResult{ParsedText: parsedText}
	return p.gen.Generate(ctx, documentId, analysisResult, structureNodes)
}

// BatchRegenerateProfiles 批量重新生成文档画像
func (p *ProfileLogicImpl) BatchRegenerateProfiles(ctx context.Context, documentIds []int64) ([]*entity.DocumentProfile, error) {
	if len(documentIds) == 0 {
		return []*entity.DocumentProfile{}, nil
	}
	result := make([]*entity.DocumentProfile, 0, len(documentIds))
	for _, id := range documentIds {
		profile, err := p.RegenerateProfile(ctx, id)
		if err != nil {
			return result, err
		}
		result = append(result, profile)
	}
	return result, nil
}
