package entity

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
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
	BboxJson           string     `gorm:"column:bbox_json"`           // 边界框 JSON
	Text               string     `gorm:"column:text"`                // 文本内容
	ContentWithWeight  string     `gorm:"column:content_with_weight"` // 带权重的内容
	TableHTML          string     `gorm:"column:table_html"`          // 表格 HTML
	ImageObjectName    string     `gorm:"column:image_object_name"`   // 图片对象名
	ImageCaption       string     `gorm:"column:image_caption"`       // 图片说明
	MetadataJson       string     `gorm:"column:metadata_json"`       // 元数据 JSON
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

func (b *DocumentBlock) IsTitleBlock() bool {
	return utils.EqualsIgnoreCase(b.BlockType, "TITLE")
}

func (b *DocumentBlock) ResolveTitle(sectionPath string) string {
	blocks := DocumentBlocks{b}
	return blocks.ResolveTitle(sectionPath)
}

func (b *DocumentBlock) ResolveChunkType() string {
	blocks := DocumentBlocks{b}
	return blocks.ResolveChunkType()
}

func (b *DocumentBlock) Ids() string {
	blocks := DocumentBlocks{b}
	return blocks.Ids()
}

type DocumentBlocks []*DocumentBlock

func (b DocumentBlocks) FirstPageNo() int {
	for _, block := range b {
		pageNo := block.PageNo
		if pageNo > 0 {
			return pageNo
		}
	}
	return 0
}

func (b DocumentBlocks) PageRange() string {
	// 提取有效页码并去重
	pageSet := make(map[int]bool, len(b))
	pages := make([]int, 0, len(b))
	for _, block := range b {
		if pn := block.PageNo; pn > 0 && !pageSet[pn] {
			pageSet[pn] = true
			pages = append(pages, pn)
		}
	}

	// 若存在有效页码，则排序并计算范围
	if len(pages) > 0 {
		slices.Sort(pages)

		first, last := pages[0], pages[len(pages)-1]
		if first == last {
			return strconv.Itoa(first)
		}
		return fmt.Sprintf("%d-%d", first, last)
	}

	// 回退逻辑：使用块自带的 pageRange
	for _, block := range b {
		if r := strings.TrimSpace(block.PageRange); r != "" {
			return r
		}
	}
	return ""
}

func (b DocumentBlocks) Ids() string {
	seen := make(map[int64]struct{})
	var builder strings.Builder
	count := 0
	builder.WriteString("[")
	for _, block := range b {
		id := block.ID
		_, exists := seen[id]
		if id > 0 && !exists {
			seen[id] = struct{}{}
			if count > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(strconv.FormatInt(id, 10))
			count++
		}
	}
	builder.WriteString("]")
	return builder.String()
}

func (b DocumentBlocks) CanonicalPaths() []string {
	result := make([]string, 0, len(b))
	for _, block := range b {
		result = append(result, block.CanonicalPath)
	}
	return result
}

func (b DocumentBlocks) ResolveTitle(sectionPath string) string {
	normalizeTitle := func(title string) string {
		normalized := strings.TrimSpace(title)
		strings.TrimLeft(normalized, "#")
		return normalized
	}
	for _, block := range b {
		if block.IsTitleBlock() && strutil.IsNotBlank(block.Text) {
			return normalizeTitle(block.Text)
		}
	}

	sectionPath = strings.TrimSpace(sectionPath)
	if sectionPath == "" {
		return ""
	}

	re := regexp.MustCompile(`[>/|]`)
	parts := re.Split(sectionPath, -1)
	for i := len(parts) - 1; i >= 0; i-- {
		segment := strings.TrimSpace(parts[i])
		if segment != "" {
			return normalizeTitle(segment)
		}
	}
	return normalizeTitle(sectionPath)
}

func (b DocumentBlocks) JoinBlockTexts() string {
	var builder strings.Builder
	first := true
	for _, block := range b {
		content := strings.TrimSpace(block.RenderBlockContent())
		if content == "" {
			continue
		}

		if !first {
			builder.WriteString("\n\n")
		}
		builder.WriteString(content)
		first = false
	}

	return builder.String()
}

func (b DocumentBlocks) CleanupAndSort() DocumentBlocks {
	if len(b) == 0 {
		return nil
	}
	predicate := func(item *DocumentBlock) bool {
		return item != nil && item.HasBlockContent()
	}
	result := make(DocumentBlocks, 0, len(b))
	for _, block := range b {
		if predicate(block) {
			result = append(result, block)
		}
	}
	less := func(a, b *DocumentBlock) int {
		if a.BlockNo == 0 {
			return 1
		} else if a.BlockNo != b.BlockNo {
			return a.BlockNo - b.BlockNo
		}
		return int(a.ID - b.ID)
	}
	slices.SortFunc(result, less)
	return result
}

func (b DocumentBlocks) ResolveChunkType() string {
	seen := make(map[string]struct{})
	var blockTypes []string

	for _, block := range b {
		blockType := strutil.Trim(block.BlockType)
		if blockType == "" {
			continue
		}
		upperType := strings.ToUpper(blockType)
		if _, exists := seen[upperType]; !exists {
			seen[upperType] = struct{}{}
			blockTypes = append(blockTypes, upperType)
		}
	}

	if len(blockTypes) == 0 {
		return "TEXT"
	}
	if len(blockTypes) == 1 {
		return blockTypes[0]
	}
	return "MIXED"
}

func (b DocumentBlocks) CommonSectionPath() string {
	for _, block := range b {
		trimmed := strings.TrimSpace(block.SectionPath)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
