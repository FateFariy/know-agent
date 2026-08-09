package vo

import "github.com/swiftbit/know-agent/internal/domain/document/model/enum"

type DocumentIndexBuild struct {
	DocumentId      int64  // 文档ID
	TaskId          int64  // 任务ID
	TaskType        int    // 任务类型
	TaskTypeName    string // 任务类型描述
	TaskStatus      int    // 任务状态
	TaskStatusName  string // 任务状态描述
	IndexStatus     int    // 索引状态
	IndexStatusName string // 索引状态描述
}

func (d *DocumentIndexBuild) FillEnumNames() {
	if d == nil {
		return
	}
	d.TaskTypeName = enum.TaskTypeName(d.TaskType)
	d.TaskStatusName = enum.TaskStatusName(d.TaskStatus)
	d.IndexStatusName = enum.IndexStatusName(d.IndexStatus)
}
