package entity

import (
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

// DocumentBlock 文档块实体
type DocumentBlock struct {
	ID                 int64      `gorm:"column:id"`                  // 主键ID
	DocumentId         int64      `gorm:"column:document_id"`         // 文档ID
	TaskId             int64      `gorm:"column:task_id"`             // 任务ID
	BlockNo            int        `gorm:"column:block_no"`            // 块序号
	BlockType          string     `gorm:"column:block_type"`          // 块类型
	ParentBlockId      int64      `gorm:"column:parent_block_id"`     // 父块ID
	SectionPath        string     `gorm:"column:section_path"`        // 章节路径
	CanonicalPath      string     `gorm:"column:canonical_path"`      // 规范路径
	PageNo             int        `gorm:"column:page_no"`             // 页码
	PageRange          string     `gorm:"column:page_range"`          // 页码范围
	BboxJSON           string     `gorm:"column:bbox_json"`           // 边界框 JSON
	Text               string     `gorm:"column:text"`                // 文本内容
	ContentWithWeight  string     `gorm:"column:content_with_weight"` // 带权重的内容
	TableHTML          string     `gorm:"column:table_html"`          // 表格 HTML
	ImageObjectName    string     `gorm:"column:image_object_name"`   // 图片对象名
	ImageCaption       string     `gorm:"column:image_caption"`       // 图片说明
	MetadataJSON       string     `gorm:"column:metadata_json"`       // 元数据 JSON
	ParentBlockNo      int        `gorm:"-"`                          // 父块编号
	BoundingBoxJSON    string     `gorm:"-"`                          // 边界框JSON
	TableRows          [][]string `gorm:"-"`                          // 表格行数据
	ImageFileName      string     `gorm:"-"`                          // 图片文件名
	ImageContentBase64 string     `gorm:"-"`                          // 图片Base64内容
}

// HasBlockContent 检查块是否有内容
func (b *DocumentBlock) HasBlockContent() bool {
	return strings.TrimSpace(b.RenderBlockContent()) != ""
}

func (b *DocumentBlock) RenderBlockContent() string {
	text := utils.FirstNonBlank(b.Text, b.ImageCaption, b.TableHTML)
	if strutil.Trim(text) == "" {
		return ""
	}

	// 根据块类型处理内容
	switch strings.ToUpper(strutil.Trim(b.BlockType)) {
	case "TITLE":
		if !strings.HasPrefix(strutil.Trim(text), "#") {
			return "# " + text
		}
	case "TABLE":
		if strutil.Trim(b.TableHTML) != "" && !strings.Contains(text, b.TableHTML) {
			return "[TABLE]\n" + text + "\n\n" + b.TableHTML
		}
	case "IMAGE", "FIGURE":
		if strutil.Trim(b.ImageCaption) != "" {
			return "[IMAGE]\n" + b.ImageCaption
		}
	}

	return text
}
