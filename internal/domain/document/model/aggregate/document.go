package aggregate

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// Document 文档聚合根
// 包含文档、任务和任务日志，作为文档领域的聚合入口
type Document struct {
	Document *entity.Document
	Task     *entity.DocumentTask
	TaskLog  *entity.DocumentTaskLog
}
