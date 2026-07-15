package errorx

import "github.com/swiftbit/know-agent/common"

var (
	ErrDocumentNotFound         = common.NewBizError(20001, "文档不存在：%d")
	ErrUnsupportedFileType      = common.NewBizError(20002, "暂不支持文件类型: %s")
	ErrEmptyFileContent         = common.NewBizError(20003, "文件内容为空：%s")
	ErrDocumentStatusInvalid    = common.NewBizError(20004, "文档状态无效：%s")
	ErrDocumentProfileNotFound  = common.NewBizError(20005, "文档属性不存在")
	ErrStrategyPlanNotFound     = common.NewBizError(20006, "策略方案不存在：%s")
	ErrStrategyStepEmpty        = common.NewBizError(20007, "当前没有可执行的策略步骤")
	ErrIndexTaskRunning         = common.NewBizError(20008, "当前文档已有索引任务正在执行")
	ErrKafkaSendFailed          = common.NewBizError(20009, "异步任务投递失败：%s")
	ErrDocumentParseFailed      = common.NewBizError(20010, "文件解析失败：%s")
	ErrDocumentStorageFailed    = common.NewBizError(20011, "文件存储失败：%s")
	ErrDocumentVectorFailed     = common.NewBizError(20012, "向量化处理失败：%s")
	ErrDocumentIndexUnavailable = common.NewBizError(20013, "文档当前没有可用索引：%s")
	ErrDocumentRetrieveEmpty    = common.NewBizError(20014, "未检索到可用资料：%s")
	ErrTaskNotFound             = common.NewBizError(20015, "任务不存在：%s")
	ErrParentBlockMissing       = common.NewBizError(20016, "当前方案缺少父块流水线，无法生成 Parent-Child 结构")
	ErrChildBlockMissing        = common.NewBizError(20017, "当前方案缺少子块流水线，无法生成 Parent-Child 结构")
	ErrDistributedLockNotFound  = common.NewBizError(20018, "分布式锁[%s]不存在")
	ErrSessionNotFound          = common.NewBizError(20019, "会话不存在: %s")
	ErrExchangeNotFound         = common.NewBizError(20020, "对话记录不存在: %s")
)
