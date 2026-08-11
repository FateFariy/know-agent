package entity

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/shared"
)

// DocumentBlock 文档块实体
type DocumentBlock struct {
	ID                 int64          `gorm:"column:id"`                  // 主键ID
	DocumentId         int64          `gorm:"column:document_id"`         // 文档ID
	TaskId             int64          `gorm:"column:task_id"`             // 任务ID
	BlockNo            int            `gorm:"column:block_no"`            // 块序号
	BlockType          string         `gorm:"column:block_type"`          // 块类型
	ParentBlockId      int64          `gorm:"column:parent_block_id"`     // 父块ID
	SectionPath        string         `gorm:"column:section_path"`        // 章节路径
	CanonicalPath      string         `gorm:"column:canonical_path"`      // 规范路径
	PageNo             int            `gorm:"column:page_no"`             // 页码
	PageRange          string         `gorm:"column:page_range"`          // 页码范围
	BboxJson           string         `gorm:"column:bbox_json"`           // 边界框 JSON
	Text               string         `gorm:"column:text"`                // 文本内容
	ContentWithWeight  string         `gorm:"column:content_with_weight"` // 带权重的内容
	TableHTML          string         `gorm:"column:table_html"`          // 表格 HTML
	ImageObjectName    string         `gorm:"column:image_object_name"`   // 图片对象名
	ImageCaption       string         `gorm:"column:image_caption"`       // 图片说明
	MetadataJson       string         `gorm:"column:metadata_json"`       // 元数据 JSON
	Metadata           map[string]any `gorm:"-"`                          // 元数据
	ParentBlockNo      int            `gorm:"-"`                          // 父块编号
	BoundingBoxJSON    string         `gorm:"-"`                          // 边界框JSON
	TableRows          [][]string     `gorm:"-"`                          // 表格行数据
	ImageFileName      string         `gorm:"-"`                          // 图片文件名
	ImageContentBase64 string         `gorm:"-"`                          // 图片Base64内容
}

// HasBlockContent 检查块是否有内容
func (b *DocumentBlock) HasBlockContent() bool {
	return b != nil && strings.TrimSpace(b.RenderBlockContent()) != ""
}

func (b *DocumentBlock) RenderBlockContent() string {
	if b == nil {
		return ""
	}
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
	return b != nil && utils.EqualsIgnoreCase(b.BlockType, "TITLE")
}

func (b *DocumentBlock) ResolveTitle() string {
	if b == nil {
		return ""
	}
	blocks := DocumentBlocks{b}
	return blocks.ResolveTitle(b.SectionPath)
}

func (b *DocumentBlock) ResolveChunkType() string {
	if b == nil {
		return ""
	}
	blocks := DocumentBlocks{b}
	return blocks.ResolveChunkType()
}

func (b *DocumentBlock) Ids() string {
	if b == nil {
		return ""
	}
	blocks := DocumentBlocks{b}
	return blocks.Ids()
}

func (b *DocumentBlock) ExtractKeywords(tokenizer shared.Tokenizer) []string {
	if b == nil {
		return nil
	}
	seed := shared.NewKeywordSeed(b.ResolveTitle(), b.SectionPath, b.RenderBlockContent())
	return seed.Build(tokenizer)
}

func (b *DocumentBlock) ExtractQuestions(keywords []string) []string {
	if b == nil {
		return nil
	}
	seed := shared.NewQuestionSeed(b.ResolveTitle(), b.ResolveChunkType(), keywords)
	return seed.Build()
}

func (b *DocumentBlock) ExtractContentWithWeight(keywords []string, parserWeightedContent string) string {
	if b == nil {
		return ""
	}
	text := b.RenderBlockContent()
	title := b.ResolveTitle()
	chunkType := b.ResolveChunkType()
	questions := b.ExtractQuestions(keywords)
	seed := shared.NewRichContentSeed(text, b.SectionPath, title, chunkType, parserWeightedContent, keywords, questions)
	return seed.Build()
}

func (b *DocumentBlock) RenderBlockWeightedContent(keywords []string) string {
	if b == nil {
		return ""
	}
	contentWithWeight := strings.TrimSpace(b.ContentWithWeight)
	if contentWithWeight != "" {
		return contentWithWeight
	}
	return b.ExtractContentWithWeight(keywords, "")
}

func (b *DocumentBlock) CloneWithText(text string) *DocumentBlock {
	if b == nil {
		return nil
	}
	clone := *b
	clone.Text = text
	return &clone
}

func (b *DocumentBlock) BuildContentWithWeight() string {
	parts := make([]string, 0, 6)
	if b.SectionPath != "" {
		parts = append(parts, fmt.Sprintf("section: %s", b.SectionPath))
	}
	if b.BlockType != "" {
		parts = append(parts, fmt.Sprintf("type: %s", b.BlockType))
	}
	if b.Text != "" {
		parts = append(parts, b.Text)
	}
	if b.ImageCaption != "" && b.ImageCaption != b.Text {
		parts = append(parts, fmt.Sprintf("caption: %s", b.ImageCaption))
	}
	if len(b.TableRows) > 0 {
		parts = append(parts, tableText(b.TableRows))
	}
	return strings.Join(parts, "\n")
}

// tableText 将二维字符串数组（表格行）转换为纯文本表示
func tableText(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		// 收集非空单元格
		cells := make([]string, 0, len(row))
		for _, cell := range row {
			if cell != "" {
				cells = append(cells, cell)
			}
		}
		// 若该行有非空单元格，则构建行字符串
		if len(cells) > 0 {
			lines = append(lines, strings.Join(cells, " | "))
		}
	}
	return strings.Join(lines, "\n")
}

var nonAllowedCharsRegex = regexp.MustCompile(`[^0-9a-zA-Z\u4e00-\u9fff]+`)

// BuildCanonicalPath 生成规范路径，格式：章节路径-块编号
func (b *DocumentBlock) BuildCanonicalPath() string {
	section := b.SectionPath
	// 正则：匹配非字母、数字、汉字的字符（连续多个替换为一个 "-"）
	section = nonAllowedCharsRegex.ReplaceAllString(section, "-")
	section = strings.Trim(section, "-")
	if section == "" {
		section = "root"
	}
	return fmt.Sprintf("/%s/%d", section, b.BlockNo)
}

type DocumentBlocks []*DocumentBlock

func (b DocumentBlocks) FirstPageNo() int {
	for _, block := range b {
		if block == nil {
			continue
		}
		pageNo := block.PageNo
		if pageNo > 0 {
			return pageNo
		}
	}
	return 0
}

func (b DocumentBlocks) ExtractKeywords(tokenizer shared.Tokenizer) []string {
	if len(b) == 0 {
		return nil
	}
	sectionPath := b.FirstBlankSectionPath()
	text := b.JoinBlockTexts()
	title := b.ResolveTitle(sectionPath)
	seed := shared.NewKeywordSeed(title, sectionPath, text)
	return seed.Build(tokenizer)
}

func (b DocumentBlocks) ExtractQuestions(keywords []string) []string {
	if len(b) == 0 {
		return nil
	}
	title := b.ResolveTitle(b.FirstBlankSectionPath())
	seed := shared.NewQuestionSeed(title, b.ResolveChunkType(), keywords)
	return seed.Build()
}

func (b DocumentBlocks) ExtractContentWithWeight(keywords []string, parserWeightedContent string) string {
	if len(b) == 0 {
		return ""
	}
	text := b.JoinBlockTexts()
	title := b.ResolveTitle(b.FirstBlankSectionPath())
	chunkType := b.ResolveChunkType()
	questions := b.ExtractQuestions(keywords)
	seed := shared.NewRichContentSeed(text, b.FirstBlankSectionPath(), title, chunkType, parserWeightedContent, keywords, questions)
	return seed.Build()
}

func (b DocumentBlocks) JoinBlockWeightedContents(tokenizer shared.Tokenizer) string {
	var contents []string
	for _, block := range b {
		keywords := block.ExtractKeywords(tokenizer)
		content := block.RenderBlockWeightedContent(keywords)
		if content != "" {
			contents = append(contents, content)
		}
	}

	return strings.Join(contents, "\n\n")
}

func (b DocumentBlocks) PageRange() string {
	// 提取有效页码并去重
	pageSet := make(map[int]bool, len(b))
	pages := make([]int, 0, len(b))
	for _, block := range b {
		if block == nil {
			continue
		}
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
		if block == nil {
			continue
		}
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
		if block == nil {
			continue
		}
		id := block.ID
		if _, exists := seen[id]; id > 0 && !exists {
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

func (b DocumentBlocks) FirstBlankCanonicalPath() string {
	for _, block := range b {
		if block == nil {
			continue
		}
		space := strings.TrimSpace(block.CanonicalPath)
		if space != "" {
			return space
		}
	}
	return ""
}

func (b DocumentBlocks) ResolveTitle(sectionPath string) string {
	normalizeTitle := func(title string) string {
		normalized := strings.TrimSpace(title)
		return strings.TrimLeft(normalized, "#")
	}
	for _, block := range b {
		if block != nil && block.IsTitleBlock() && strutil.IsNotBlank(block.Text) {
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
		if block == nil {
			continue
		}
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

func (b DocumentBlocks) ToMap() map[int64]*DocumentBlock {
	result := make(map[int64]*DocumentBlock)
	for _, block := range b {
		if block != nil {
			result[block.ID] = block
		}
	}
	return result
}

func (b DocumentBlocks) ResolveChunkType() string {
	seen := make(map[string]struct{})
	var blockTypes []string

	for _, block := range b {
		if block == nil {
			continue
		}
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

func (b DocumentBlocks) FirstBlankSectionPath() string {
	for _, block := range b {
		if block == nil {
			continue
		}
		trimmed := strings.TrimSpace(block.SectionPath)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// Normalize 对块列表进行规范化处理：
//   - 过滤掉无文本、无表格 HTML、无图片说明的空块；
//   - 重新连续编号 block_no；
//   - 清理文本（使用 cleanupText）；
//   - 根据标题更新当前章节路径；
//   - 为每个块补全 section_path、canonical_path 和 content_with_weight
func (b DocumentBlocks) Normalize() DocumentBlocks {
	normalized := make(DocumentBlocks, 0, len(b))
	currentSection := ""

	for _, block := range b {
		text := utils.CleanupSpace(block.Text)
		if text == "" && block.TableHTML == "" && block.ImageCaption == "" {
			continue
		}

		// 重新编号（从1开始）
		block.BlockNo = len(normalized) + 1
		block.Text = text

		// 若当前块是标题，更新当前章节信息
		if block.BlockType == "TITLE" {
			currentSection = text
		}

		// 补全 section_path（若为空则使用当前章节路径）
		if block.SectionPath == "" {
			block.SectionPath = currentSection
		}

		// 补全 canonical_path（若为空则自动生成）
		if block.CanonicalPath == "" {
			block.CanonicalPath = block.BuildCanonicalPath()
		}

		// 补全 content_with_weight（若为空则自动生成）
		if block.ContentWithWeight == "" {
			block.ContentWithWeight = block.BuildContentWithWeight()
		}

		normalized = append(normalized, block)
	}
	return normalized
}
