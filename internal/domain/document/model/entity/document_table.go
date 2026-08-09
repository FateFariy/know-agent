package entity

import (
	"fmt"

	"github.com/swiftbit/know-agent/internal/domain/document/model/shared"
)

// DocumentTable 文档表格实体
type DocumentTable struct {
	ID              int64              `gorm:"column:id"`            // 主键ID
	DocumentId      int64              `gorm:"column:document_id"`   // 文档ID
	TaskId          int64              `gorm:"column:task_id"`       // 任务ID
	BlockId         int64              `gorm:"column:block_id"`      // 块ID
	TableNo         int                `gorm:"column:table_no"`      // 表格序号
	SectionPath     string             `gorm:"column:section_path"`  // 章节路径
	PageNo          int                `gorm:"column:page_no"`       // 页码
	PageRange       string             `gorm:"column:page_range"`    // 页码范围
	BboxJson        string             `gorm:"column:bbox_json"`     // 边界框 JSON
	Title           string             `gorm:"column:title"`         // 表格标题
	RowCount        int                `gorm:"column:row_count"`     // 行数
	ColumnCount     int                `gorm:"column:column_count"`  // 列数
	TableHTML       string             `gorm:"column:table_html"`    // 表格 HTML
	MetadataJSON    string             `gorm:"column:metadata_json"` // 元数据 JSON
	SourceBlockNo   int                `gorm:"-"`                    // 源块编号
	BoundingBoxJSON string             `gorm:"-"`                    // 边界框JSON
	TitleHint       string             `gorm:"-"`                    // 标题提示
	ProjectionOwner string             `gorm:"-"`                    // 投影所有者
	SchemaVersion   string             `gorm:"-"`                    // 模式版本
	SourceOrigin    string             `gorm:"-"`                    // 源文本来源
	SourceHash      string             `gorm:"-"`                    // 源文本哈希
	SyntaxNodeId    string             `gorm:"-"`                    // 语法节点ID
	SourceSpan      *shared.SourceSpan `gorm:"-"`                    // 源文本位置范围
	Rows            []*TableRow        `gorm:"-"`                    // 表格行列表
	SourceMetadata  map[string]any     `gorm:"-"`                    // 源元数据
}

func (t *DocumentTable) ValidateCandidateRows() ([]*TableRow, error) {
	if len(t.Rows) == 0 {
		return nil, fmt.Errorf("table candidate 缺少 rows")
	}

	rows := t.Rows
	columnCount := -1

	for rowIndex, row := range rows {
		// 验证行基础属性
		if row == nil ||
			row.RowIndex != rowIndex ||
			row.Header != (rowIndex == 0) ||
			row.Cells == nil ||
			len(row.Cells) == 0 {
			return nil, fmt.Errorf("table candidate row 顺序/header 非法: rowIndex=%d", rowIndex)
		}

		// 验证列数一致性
		if columnCount < 0 {
			columnCount = len(row.Cells)
		} else if columnCount != len(row.Cells) {
			return nil, fmt.Errorf("table candidate columnCount 不守恒")
		}

		// 验证单元格属性
		for colIndex, cell := range row.Cells {
			if cell == nil ||
				cell.RowIndex != rowIndex ||
				cell.ColumnIndex != colIndex ||
				cell.Header != row.Header {
				return nil, fmt.Errorf("table candidate cell 顺序/header 非法: row=%d, column=%d",
					rowIndex, colIndex)
			}
		}
	}

	return rows, nil
}
