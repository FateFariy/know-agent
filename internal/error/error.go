package errorx

import "github.com/swiftbit/know-agent/common"

var (
	ErrDocumentNotFound      = common.NewBizError(20001, "文档不存在")
	ErrUnsupportedFileType   = common.NewBizError(20002, "暂不支持文件类型: %s")
	ErrEmptyFileContent      = common.NewBizError(20003, "文件内容为空：%s")
	ErrDocumentStatusInvalid = common.NewBizError(20004, "文档状态无效：%s")
	ErrDocumentStorageFailed = common.NewBizError(20010, "文件存储失败：%s")
)
