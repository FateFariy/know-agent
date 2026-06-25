package errorx

import "github.com/swiftbit/know-agent/common"

var (
	ErrDocumentNotFound         = common.NewBizError(20001, "文档不存在")
	ErrUnsupportedFileType      = common.NewBizError(20002, "暂不支持文件类型: %s")
	ErrEmptyFileContent         = common.NewBizError(20003, "文件内容为空：%s")
	ErrDocumentStatusInvalid    = common.NewBizError(20004, "文档状态无效：%s")
	ErrStrategyPlanNotFound     = common.NewBizError(20005, "策略方案不存在：%s")
	ErrStrategyStepEmpty        = common.NewBizError(20006, "当前没有可执行的策略步骤")
	ErrIndexTaskRunning         = common.NewBizError(20007, "当前文档已有索引任务正在执行")
	ErrKafkaSendFailed          = common.NewBizError(20008, "异步任务投递失败：%s")
	ErrDocumentParseFailed      = common.NewBizError(20009, "文件解析失败：%s")
	ErrDocumentStorageFailed    = common.NewBizError(20010, "文件存储失败：%s")
	ErrDocumentVectorFailed     = common.NewBizError(20011, "向量化处理失败：%s")
	ErrDocumentIndexUnavailable = common.NewBizError(20012, "文档当前没有可用索引：%s")
	ErrDocumentRetrieveEmpty    = common.NewBizError(20013, "未检索到可用资料：%s")
	ErrTaskNotFound             = common.NewBizError(20014, "任务不存在：%s")
)
