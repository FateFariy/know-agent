package errorx

import "github.com/swiftbit/know-agent/common"

var (
	ErrUnsupportedFileType    = common.NewBizError(20002, "当前文件类型暂不支持")
	ErrDocumentUploadFailed   = common.NewBizError(20010, "文件上传失败")
	ErrDocumentDownloadFailed = common.NewBizError(20011, "文件下载失败")
	ErrDocumentDeleteFailed   = common.NewBizError(20012, "文件删除失败")
)
