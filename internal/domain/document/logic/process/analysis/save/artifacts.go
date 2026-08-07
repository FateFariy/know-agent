package save

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/slice"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type ArtifactPersistPhase struct {
	repo      adapter.DocumentRepository
	tableRepo adapter.TableRepository
	port      *adapter.DocumentPort
}

func (p *ArtifactPersistPhase) Name() string {
	return "解析产物保存阶段"
}

func (p *ArtifactPersistPhase) Execute(ctx context.Context, saveCtx *Context) error {
	//TODO implement me
	panic("implement me")
}

func NewArtifactPersistPhase(repo adapter.DocumentRepository, tableRepo adapter.TableRepository, port *adapter.DocumentPort) *ArtifactPersistPhase {
	return &ArtifactPersistPhase{
		repo:      repo,
		tableRepo: tableRepo,
		port:      port,
	}
}

func (p *ArtifactPersistPhase) saveParseArtifactsAndBlocks(ctx context.Context, documentId, taskId int64, analysisResult *vo.AnalysisResult) error {
	artifacts, err := p.buildParseArtifactEntities(ctx, documentId, taskId, analysisResult.ParseArtifacts)
	if err != nil {
		return err
	}
	blocks, err := p.buildDocumentBlockEntities(ctx, documentId, taskId, analysisResult.Blocks)
	if err != nil {
		return err
	}
	oldArtifacts, err := p.repo.SelectArtifactsByTask(ctx, documentId, taskId)
	if err != nil {
		return err
	}
	objects := slice.Map(oldArtifacts, func(_ int, artifact *entity.ParseArtifact) string {
		return artifact.ObjectName
	})

	txFn := func(txCtx context.Context) error {
		if err = p.port.DeleteObjects(txCtx, objects); err != nil {
			return err
		}
		if err = p.tableRepo.DeleteTablesByTask(txCtx, documentId, taskId); err != nil {
			return err
		}
		if err = p.repo.DeleteArtifactsByTask(txCtx, documentId, taskId); err != nil {
			return err
		}
		if err = p.repo.DeleteDocumentBlocksByTask(txCtx, documentId, taskId); err != nil {
			return err
		}
		if err = p.repo.InsertParsedArtifactBatch(txCtx, artifacts); err != nil {
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

func (p *ArtifactPersistPhase) buildParseArtifactEntities(ctx context.Context, documentId, taskId int64,
	candidates []*entity.ParseArtifact) ([]*entity.ParseArtifact, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}

	artifacts := make([]*entity.ParseArtifact, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.ContentBase64 == "" || candidate.ContentType == "" {
			continue
		}

		// Base64 解码
		content, err := decodeBase64(candidate.ContentBase64, fmt.Sprintf("解析产物 %s", candidate.FileName))
		if err != nil {
			continue
		}

		fileName := utils.BlankToDefault(candidate.FileName, strings.ToLower(candidate.ArtifactType)+".bin")

		// 上传到 MinIO
		objectName, err := p.port.UploadParseArtifact(ctx, documentId, taskId, fileName, candidate.ContentType, content)
		if err != nil {
			return nil, fmt.Errorf("上传解析产物 %s 失败: %w", fileName, err)
		}

		candidate.ID = utils.GetSnowflakeNextID()
		candidate.DocumentId = documentId
		candidate.TaskId = taskId
		candidate.ObjectName = objectName
		if candidate.ContentHash == "" {
			hasher := sha256.New()
			hasher.Write(content)
			candidate.ContentHash = hex.EncodeToString(hasher.Sum(nil))
		}
		artifacts = append(artifacts, candidate)
	}

	return artifacts, nil
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

//func (p *ArtifactPersistPhase) buildTableEntities(ctx context.Context, documentId, taskId int64,
//	blocks []*entity.DocumentBlock, candidates []*vo.TableCandidate) ([]*entity.DocumentTable, error) {
//	if len(candidates) == 0 {
//		return nil, nil
//	}
//	blocksByNo := utils.SliceToMapBy(blocks, func(item *entity.DocumentBlock) (int, *entity.DocumentBlock) {
//		return item.BlockNo, item
//	})
//	tableNo := 1
//	for _, candidate := range candidates {
//		if candidate == nil || candidate.SourceBlockNo <= 0 {
//			return nil, errors.New("table candidate 缺少 sourceBlockNo")
//		}
//		block := blocksByNo[candidate.SourceBlockNo]
//		if block == nil || block.DocumentId != documentId || block.TaskId != taskId {
//			return nil, fmt.Errorf("table candidate %d 对应的 block 不存在", candidate.SourceBlockNo)
//		}
//
//	}
//}

// decodeBase64 Base64 解码，带错误包装
func decodeBase64(encoded, context string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%s Base64 解码失败: %w", context, err)
	}
	return decoded, nil
}
