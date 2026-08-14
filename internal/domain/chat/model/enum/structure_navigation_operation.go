package enum

import (
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

// StructureNavigationOperation 结构导航操作枚举
type StructureNavigationOperation = string

const (
	CurrentSection      StructureNavigationOperation = "CURRENT_SECTION"
	ParentSection       StructureNavigationOperation = "PARENT_SECTION"
	PreviousSibling     StructureNavigationOperation = "PREVIOUS_SIBLING"
	NextSibling         StructureNavigationOperation = "NEXT_SIBLING"
	DirectChildren      StructureNavigationOperation = "DIRECT_CHILDREN"
	SectionWithSiblings StructureNavigationOperation = "SECTION_WITH_SIBLINGS"
	SectionWithChildren StructureNavigationOperation = "SECTION_WITH_CHILDREN"
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
