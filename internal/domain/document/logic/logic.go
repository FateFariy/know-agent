package logic

import (
	"context"
	"mime/multipart"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// DocumentLifecycleLogic 文档生命周期业务逻辑接口
type DocumentLifecycleLogic interface {
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, document entity.Document) error
}
