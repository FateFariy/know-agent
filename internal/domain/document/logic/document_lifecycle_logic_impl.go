package logic

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

type DocumentLifecycleLogicImpl struct {
	port *adapter.DocumentPort
}

var _ DocumentLifecycleLogic = (*DocumentLifecycleLogicImpl)(nil)

func NewDocumentLifecycleLogicImpl(port *adapter.DocumentPort) DocumentLifecycleLogic {
	return &DocumentLifecycleLogicImpl{
		port: port,
	}
}

// Upload 上传文件
func (d *DocumentLifecycleLogicImpl) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, document entity.Document) error {
	// 通过读取前 512 字节检测 MIME 类型
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	mimeType := http.DetectContentType(buf[:n])
	if !isAllowedType(mimeType) {
		return common.NewBizErrorf(http.StatusUnsupportedMediaType, "不支持的文件类型: %s", mimeType)
	}
	_, _ = file.Seek(0, io.SeekStart)

	documentID := utils.GetSnowflakeNextID()

	_, err := d.port.UploadOriginalFile(ctx, documentID, header.Filename, buf, mimeType)

	// TODO implement me
	panic("implement me")
}
