package enum

import (
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

type AnswerShapeRequirement = string

const (
	AnswerShapeRequirementCompare                  AnswerShapeRequirement = "COMPARE"                     // 比较
	AnswerShapeRequirementList                     AnswerShapeRequirement = "LIST"                        // 列表
	AnswerShapeRequirementSteps                    AnswerShapeRequirement = "STEPS"                       // 步骤
	AnswerShapeRequirementNegativeBoundary         AnswerShapeRequirement = "NEGATIVE_BOUNDARY"           // 负向边界
	AnswerShapeRequirementCategoryTableRowCoverage AnswerShapeRequirement = "CATEGORY_TABLE_ROW_COVERAGE" // 分类表格行覆盖
)

// ParseAnswerShapes 解析字符串列表为 AnswerShapeRequirement 列表，若遇到未知值则返回 nil（表示无效）
func ParseAnswerShapes(raws []string) []AnswerShapeRequirement {
	if len(raws) == 0 {
		return nil
	}
	seen := make(map[AnswerShapeRequirement]bool)
	var result []AnswerShapeRequirement

	for _, raw := range raws {
		req := strings.ToUpper(utils.Trim(raw))
		if req == "" {
			continue
		}
		switch req {
		case AnswerShapeRequirementCompare, AnswerShapeRequirementList,
			AnswerShapeRequirementSteps, AnswerShapeRequirementNegativeBoundary,
			AnswerShapeRequirementCategoryTableRowCoverage:
			if !seen[req] {
				seen[req] = true
				result = append(result, req)
			}
		default:
			return nil
		}
	}
	return result
}
