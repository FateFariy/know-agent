package graph

import (
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// AnswerRender 图谱回答渲染接口
type AnswerRender interface {
	RenderAnswer(mode vo.ExecutionMode, decision *vo.DocumentNavigationDecision, result *entity.GraphQueryResult) string
}
