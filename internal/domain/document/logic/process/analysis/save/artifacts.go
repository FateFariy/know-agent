package save

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

type ArtifactPersistPhase struct {
	repo      adapter.DocumentRepository
	tableRepo adapter.TableRepository
	port      *adapter.DocumentPort
}

func NewArtifactPersistPhase(repo adapter.DocumentRepository, tableRepo adapter.TableRepository, port *adapter.DocumentPort) *ArtifactPersistPhase {
	return &ArtifactPersistPhase{
		repo:      repo,
		tableRepo: tableRepo,
		port:      port,
	}
}

func (p *ArtifactPersistPhase) Name() string {
	return "解析产物保存阶段"
}

func (p *ArtifactPersistPhase) Execute(ctx context.Context, saveCtx *Context) error {
	if saveCtx == nil || saveCtx.AnalysisResult == nil {
		return nil
	}
	return p.saveBlocks(ctx, saveCtx)
}

func (p *ArtifactPersistPhase) saveBlocks(ctx context.Context, parseCtx *Context) error {
	documentId, taskId, analysisResult := parseCtx.DocumentId, parseCtx.TaskId, parseCtx.AnalysisResult
	blocks, err := p.buildDocumentBlockEntities(ctx, documentId, taskId, analysisResult.Blocks)
	if err != nil {
		return err
	}
	txFn := func(txCtx context.Context) error {
		if err = p.tableRepo.DeleteTablesByTask(txCtx, documentId, taskId); err != nil {
			return err
		}
		if err = p.repo.DeleteDocumentBlocksByTask(txCtx, documentId, taskId); err != nil {
			return err
		}
		if err = p.repo.InsertDocumentBlockBatch(txCtx, blocks); err != nil {
			return err
		}
		// todo 待完善，保存表格
		return nil
	}

	return p.repo.Do(ctx, txFn)
}

func (p *ArtifactPersistPhase) buildDocumentBlockEntities(ctx context.Context, documentId, taskId int64,
	candidates []*entity.DocumentBlock) ([]*entity.DocumentBlock, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}
	idsMap := make(map[int]int64, len(candidates))
	for _, candidate := range candidates {
		if candidate != nil && candidate.BlockNo > 0 {
			idsMap[candidate.BlockNo] = utils.GetSnowflakeNextID()
		}
	}
	blocks := make([]*entity.DocumentBlock, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.BlockNo <= 0 || candidate.BlockType == "" {
			continue
		}
		candidate.ID = idsMap[candidate.BlockNo]
		candidate.DocumentId = documentId
		candidate.TaskId = taskId
		candidate.ParentBlockId = idsMap[candidate.ParentBlockNo]

		if candidate.ImageContentBase64 != "" {
			content, err := decodeBase64(candidate.ImageContentBase64, fmt.Sprintf("图片 %d", candidate.BlockNo))
			if err != nil {
				return nil, err
			}
			fileName := utils.BlankToDefault(candidate.ImageFileName, fmt.Sprintf("image_%d.png", candidate.BlockNo))
			objectName, err := p.port.UploadParseArtifact(ctx, documentId, taskId, fileName, "image/png", content)
			if err != nil {
				return nil, err
			}
			candidate.ImageObjectName = objectName
		}
		blocks = append(blocks, candidate)
	}
	return blocks, nil
}

// decodeBase64 Base64 解码，带错误包装
func decodeBase64(encoded, context string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%s Base64 解码失败: %w", context, err)
	}
	return decoded, nil
}
