package vo

import (
	"regexp"

	"github.com/swiftbit/know-agent/common/utils"
)

var (
	// yearPattern 匹配年份提示
	yearPattern = regexp.MustCompile(`\b(20\d{2})\b`)

	// decimalSectionPattern 匹配十进制/阿拉伯数字章节提示（如"3.2节"、"1.1"）
	decimalSectionPattern = regexp.MustCompile(`\b\d+(?:\.\d+){0,3}\s*(?:节|章节|部分|章|节)\b`)

	// namedSectionPattern 匹配具名章节提示（如"第三章"、"附录A"）
	namedSectionPattern = regexp.MustCompile(`第\s*[一二三四五六七八九十百千\d]+\s*[章节部分]|附录\s*[A-Za-z一二三四五六七八九十\d]+`)
)

// RetrievalMetadataFilters 检索元数据过滤条件
type RetrievalMetadataFilters struct {
	DocumentNameHints []string `json:"documentNameHints,omitempty"` // 文档名称提示
	SectionPathHints  []string `json:"sectionPathHints,omitempty"`  // 章节路径提示
	EntityHints       []string `json:"entityHints,omitempty"`       // 实体提示
	YearHints         []string `json:"yearHints,omitempty"`         // 年份提示
}

// Clone 深拷贝过滤器
func (f *RetrievalMetadataFilters) Clone() *RetrievalMetadataFilters {
	if f == nil {
		return nil
	}
	return &RetrievalMetadataFilters{
		DocumentNameHints: utils.Copy(f.DocumentNameHints),
		SectionPathHints:  utils.Copy(f.SectionPathHints),
		YearHints:         utils.Copy(f.YearHints),
		EntityHints:       utils.Copy(f.EntityHints),
	}
}

// NewMetadataFilters 构建检索元数据过滤条件
func NewMetadataFilters(normalizedQuery string, intentResult *IntentRecognitionResult) *RetrievalMetadataFilters {
	filters := &RetrievalMetadataFilters{}

	if utils.IsBlank(normalizedQuery) && intentResult == nil {
		return filters
	}
	fn := func(hints []string) []string {
		return utils.FilterUniqueLimit(hints, 8, func(hint string) (string, bool) {
			return hint, hint != ""
		})
	}

	// 提取年份
	filters.YearHints = fn(yearPattern.FindAllString(normalizedQuery, -1))

	// 提取章节提示（阿拉伯数字 + 中文命名）
	sectionHints := append([]string{}, decimalSectionPattern.FindAllString(normalizedQuery, -1)...)
	sectionHints = append(sectionHints, namedSectionPattern.FindAllString(normalizedQuery, -1)...)
	filters.SectionPathHints = fn(sectionHints)

	// 基于意图结果补充实体提示
	if intentResult != nil {
		filters.EntityHints = fn(intentResult.Entities)
	}

	return filters
}
