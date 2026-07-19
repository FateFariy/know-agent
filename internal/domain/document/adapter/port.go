package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type DocumentPort struct {
	Storage
	MessageProducer
	VectorIndexer
	KeywordIndexer
}

func NewDocumentPort(storage Storage, messageProducer MessageProducer, vectorDB VectorIndexer, keywordSearch KeywordIndexer) *DocumentPort {
	return &DocumentPort{
		Storage:         storage,
		MessageProducer: messageProducer,
		VectorIndexer:   vectorDB,
		KeywordIndexer:  keywordSearch,
	}
}

type Storage interface {
	// UploadOriginalFile 上传原始文件
	UploadOriginalFile(ctx context.Context, documentID int64, fileName string, bytes []byte, contentType string) (*vo.StoredObjectInfo, error)

	// UploadParsedText 上传解析后的文本内容
	UploadParsedText(ctx context.Context, documentID int64, parsedText string) (string, error)

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

type Indexer interface {
	// BuildIndex 构建索引
	BuildIndex(ctx context.Context, chunks []*entity.DocumentChunk) error

	// DeleteByDocumentId 根据文档ID删除
	DeleteByDocumentId(ctx context.Context, documentId int64) error
}
