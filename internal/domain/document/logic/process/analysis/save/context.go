package save

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// Context 保存阶段上下文
type Context struct {
	DocumentId     int64
	TaskId         int64
	AnalysisResult *aggregate.AnalysisResult
	StructureNodes []*entity.StructureNode
	ParsedTextPath string
}
