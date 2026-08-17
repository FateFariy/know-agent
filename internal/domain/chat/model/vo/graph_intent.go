package vo

import "github.com/swiftbit/know-agent/common/utils"

// Clone 深拷贝图谱意图。
func (g *GraphIntent) Clone() *GraphIntent {
	if g == nil {
		return nil
	}
	s := *g
	s.Entities = utils.Copy(g.Entities)
	s.TargetEntities = utils.Copy(g.TargetEntities)
	return &s
}

// GraphIntent 图谱检索意图
type GraphIntent struct {
	Requested      bool     `json:"requested"`      // 是否请求
	Entities       []string `json:"entities"`       // 实体列表
	TargetEntities []string `json:"targetEntities"` // 目标实体列表
	MaxHops        int      `json:"maxHops"`        // 最大跳数
	Source         string   `json:"source"`         // 来源
}
