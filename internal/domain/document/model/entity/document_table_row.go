package entity

import "github.com/swiftbit/know-agent/internal/domain/document/model/shared"

// TableRow 文档表格行实体
type TableRow struct {
	ID           int64              `gorm:"column:id"`          // 主键ID
	DocumentId   int64              `gorm:"column:document_id"` // 文档ID
	TaskId       int64              `gorm:"column:task_id"`     // 任务ID
	TableId      int64              `gorm:"column:table_id"`    // 表格ID
	RowNo        int                `gorm:"column:row_no"`      // 行序号
	RowText      string             `gorm:"column:row_text"`    // 行文本
	RowIndex     int                // 行索引
	IsHeader     bool               // 是否为表头行
	SourceRowNo  int                // 源行号
	SyntaxNodeId string             // 语法节点ID
	SourceSpan   *shared.SourceSpan // 源文本位置范围
	Cells        []*TableCell       // 单元格列表
}
