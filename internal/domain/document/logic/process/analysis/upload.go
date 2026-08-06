package analysis

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

// UploadPhase 上传阶段：上传解析后的纯文本到对象存储
type UploadPhase struct {
	repo adapter.DocumentRepository
	port *adapter.DocumentPort
}

func NewUploadPhase(repo adapter.DocumentRepository, port *adapter.DocumentPort) *UploadPhase {
	return &UploadPhase{repo: repo, port: port}
}

func (p *UploadPhase) Name() string {
	return "上传阶段"
}

func (p *UploadPhase) Execute(ctx context.Context, parseCtx *Context) error {
	parsedTextPath, err := p.port.UploadParsedText(ctx, parseCtx.DocumentID, parseCtx.AnalysisResult.ParsedText)
	if err != nil {
		return err
	}
	parseCtx.ParsedTextPath = parsedTextPath
	return nil
}

func (p *UploadPhase) saveParseArtifactsAndBlocks(ctx context.Context, documentID, taskID int64, analysisResult *vo.AnalysisResult) error {
	artifacts, err := p.buildParseArtifactEntities(ctx, documentID, taskID, analysisResult.ParseArtifacts)
	if err != nil {
		return err
	}
	blocks, err := p.buildDocumentBlockEntities(ctx, documentID, taskID, analysisResult.Blocks)
	if err != nil {
		return err
	}
	oldArtifacts, err := p.repo.SelectArtifactsByTask(ctx, documentID, taskID)
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
		if err = p.repo.DT(txCtx, artifacts); err != nil {
			return err
		}
		if err = p.repo.InsertDocumentBlockBatch(txCtx, blocks); err != nil {
			return err
		}
		if err = p.repo.DeleteArtifactsByTask(txCtx, documentID, taskID); err != nil {
			return err
		}
		if err := p.repo.DeleteBlocksByTask(txCtx, documentID, taskID); err != nil {
		}
		return err
	}

	return nil
}

func (p *UploadPhase) buildParseArtifactEntities(ctx context.Context, documentID, taskID int64,
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
		objectName, err := p.port.UploadParseArtifact(ctx, documentID, taskID, fileName, candidate.ContentType, content)
		if err != nil {
			return nil, fmt.Errorf("上传解析产物 %s 失败: %w", fileName, err)
		}

		candidate.ID = utils.GetSnowflakeNextID()
		candidate.DocumentID = documentID
		candidate.TaskID = taskID
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

func (p *UploadPhase) buildDocumentBlockEntities(ctx context.Context, documentID, taskID int64,
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
		candidate.DocumentID = documentID
		candidate.TaskID = taskID
		candidate.ParentBlockID = idsMap[candidate.ParentBlockNo]

		if candidate.ImageContentBase64 != "" {
			content, err := decodeBase64(candidate.ImageContentBase64, fmt.Sprintf("图片 %d", candidate.BlockNo))
			if err != nil {
				return nil, err
			}
			fileName := utils.BlankToDefault(candidate.ImageFileName, fmt.Sprintf("image_%d.png", candidate.BlockNo))
			objectName, err := p.port.UploadParseArtifact(ctx, documentID, taskID, fileName, "image/png", content)
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
