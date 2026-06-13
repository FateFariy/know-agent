package vo

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
	d.TaskTypeName = TaskTypeName(d.TaskType)
	d.TaskStatusName = TaskStatusName(d.TaskStatus)
	d.IndexStatusName = IndexStatusName(d.IndexStatus)
}
