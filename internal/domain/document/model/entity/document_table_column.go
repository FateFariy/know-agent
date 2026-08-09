package entity

// TableColumn 文档表格列实体
type TableColumn struct {
	ID             int64  `gorm:"column:id"`              // 主键ID
	DocumentId     int64  `gorm:"column:document_id"`     // 文档ID
	TaskId         int64  `gorm:"column:task_id"`         // 任务ID
	TableId        int64  `gorm:"column:table_id"`        // 表格ID
	ColumnNo       int    `gorm:"column:column_no"`       // 列序号
	ColumnName     string `gorm:"column:column_name"`     // 列名
	NormalizedName string `gorm:"column:normalized_name"` // 归一化列名
	ValueType      string `gorm:"column:value_type"`      // 值类型
}
