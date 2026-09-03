package enum

import (
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

// StructureNavigationOperation 结构导航操作枚举
type StructureNavigationOperation = string

const (
	CurrentSection      StructureNavigationOperation = "CURRENT_SECTION"       // 当前章节
	ParentSection       StructureNavigationOperation = "PARENT_SECTION"        // 父章节
	PreviousSibling     StructureNavigationOperation = "PREVIOUS_SIBLING"      // 上一个兄弟章节
	NextSibling         StructureNavigationOperation = "NEXT_SIBLING"          // 下一个兄弟章节
	DirectChildren      StructureNavigationOperation = "DIRECT_CHILDREN"       // 直接子章节（仅一级）
	SectionWithSiblings StructureNavigationOperation = "SECTION_WITH_SIBLINGS" // 当前章节及其兄弟章节
	SectionWithChildren StructureNavigationOperation = "SECTION_WITH_CHILDREN" // 当前章节及其所有子章节
)

// ParseStructureOperations 解析字符串列表为 StructureNavigationOperation 列表，去重并忽略无效值
func ParseStructureOperations(raws []string) []StructureNavigationOperation {
	if len(raws) == 0 {
		return nil
	}
	seen := make(map[StructureNavigationOperation]bool)
	var result []StructureNavigationOperation

	for _, raw := range raws {
		op := strings.ToUpper(utils.Trim(raw))
		if op == "" {
			continue
		}
		switch op {
		case CurrentSection, ParentSection, PreviousSibling, NextSibling,
			DirectChildren, SectionWithSiblings, SectionWithChildren:
			if !seen[op] {
				seen[op] = true
				result = append(result, op)
			}
		}
	}
	return result
}
