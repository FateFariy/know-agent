package save

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// Context 保存阶段上下文
type Context struct {
	DocumentId     int64
	TaskId         int64
	AnalysisResult *vo.AnalysisResult
	StructureNodes []*entity.StructureNode
	ParsedTextPath string
}
