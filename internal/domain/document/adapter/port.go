package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type Storage interface {
	// Name 返回存储类型名称
	Name() int

	// UploadOriginalFile 上传原始文件
	UploadOriginalFile(ctx context.Context, documentId int64, fileName string, bytes []byte, contentType string) (*vo.StoredObjectInfo, error)

	// UploadParsedText 上传解析后的文本内容
	UploadParsedText(ctx context.Context, documentId int64, parsedText string) (string, error)

	// UploadParseArtifact 上传解析产物
	UploadParseArtifact(ctx context.Context, documentId, taskId int64, name, contentType string, content []byte) (string, error)

	// DownloadObject 下载二进制对象文件
	DownloadObject(ctx context.Context, objectName string) ([]byte, error)

	// DownloadText 下载文本内容
	DownloadText(ctx context.Context, objectName string) (string, error)

	// DeleteObjects 批量删除存储对象
	DeleteObjects(ctx context.Context, objectNameList []string) error
}

type MessageProducer interface {
	Send(ctx context.Context, topic, key string, message any) error
}

type KeywordIndexer interface {
	// BuildIndexes 构建索引
	BuildIndexes(ctx context.Context, chunks []*entity.DocumentChunk) error

	// DeleteByDocumentId 根据文档ID删除
	DeleteByDocumentId(ctx context.Context, documentId int64) error
}
type VectorIndexer interface {
	// BuildVectors 构建索引
	BuildVectors(ctx context.Context, chunks []*entity.DocumentChunk) error

	// DeleteByDocumentId 根据文档ID删除
	DeleteByDocumentId(ctx context.Context, documentId int64) error
}

type KnowledgeGateway interface {
	RequireEnabled(ctx context.Context, knowledgeBaseId int64) (*entity.KnowledgeBase, error)
}

// PromptRenderer 负责将 sourceText 渲染为大模型提示词
type PromptRenderer interface {
	// Render 渲染提示词
	Render(templateName string, variables map[string]any) (string, error)
}
