package enum

type AnswerShapeRequirement string

const (
	AnswerShapeRequirementCompare                  AnswerShapeRequirement = "COMPARE"
	AnswerShapeRequirementList                     AnswerShapeRequirement = "LIST"
	AnswerShapeRequirementSteps                    AnswerShapeRequirement = "STEPS"
	AnswerShapeRequirementNegativeBoundary         AnswerShapeRequirement = "NEGATIVE_BOUNDARY"
	AnswerShapeRequirementCategoryTableRowCoverage AnswerShapeRequirement = "CATEGORY_TABLE_ROW_COVERAGE"
)
