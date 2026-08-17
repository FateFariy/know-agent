package vo

import (
	"time"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// RetrievalChannelPlan 检索通道计划
type RetrievalChannelPlan struct {
	Name               enum.RetrievalChannel `json:"name"`               // 通道名称
	Enabled            bool                  `json:"enabled"`            // 是否启用
	TopK               int                   `json:"topK"`               // 返回数量
	Timeout            time.Duration         `json:"timeout"`            // 超时时间
	Budget             int                   `json:"budget"`             // 预算
	Weight             float64               `json:"weight"`             // 权重
	MinimumScore       float64               `json:"minimumScore"`       // 最小分数
	RelativeScoreFloor float64               `json:"relativeScoreFloor"` // 相对分数下限
}

// Clone 深拷贝通道计划
func (p *RetrievalChannelPlan) Clone() *RetrievalChannelPlan {
	if p == nil {
		return nil
	}
	s := *p
	return &s
}

// NewVectorChannelPlan 创建向量检索通道计划
func NewVectorChannelPlan(enabled bool, topK int, timeout time.Duration, weight, minScore float64) *RetrievalChannelPlan {
	return newChannelPlan(enum.RetrievalChannelVector, enabled, topK, timeout, weight, minScore, 0)
}

// NewKeywordChannelPlan 创建关键词检索通道计划
func NewKeywordChannelPlan(enabled bool, topK int, timeout time.Duration, weight, relativeFloor float64) *RetrievalChannelPlan {
	return newChannelPlan(enum.RetrievalChannelKeyword, enabled, topK, timeout, weight, 0, relativeFloor)
}

// NewTableChannelPlan 创建表格检索通道计划
func NewTableChannelPlan(enabled bool, topK int, timeout time.Duration, weight float64) *RetrievalChannelPlan {
	return newChannelPlan(enum.RetrievalChannelTable, enabled, topK, timeout, weight, 0, 0)
}

// NewGraphRAGChannelPlan 创建图RAG检索通道计划
func NewGraphRAGChannelPlan(enabled bool, topK int, timeout time.Duration, weight float64) *RetrievalChannelPlan {
	return newChannelPlan(enum.RetrievalChannelGraphRAG, enabled, topK, timeout, weight, 0, 0)
}

// NewRaptorChannelPlan 创建Raptor检索通道计划
func NewRaptorChannelPlan(enabled bool, topK int, timeout time.Duration, weight float64) *RetrievalChannelPlan {
	return newChannelPlan(enum.RetrievalChannelRaptor, enabled, topK, timeout, weight, 0, 0)
}

func newChannelPlan(channel enum.RetrievalChannel, enabled bool, topK int, timeout time.Duration, weight, minScore, relativeFloor float64) *RetrievalChannelPlan {
	return &RetrievalChannelPlan{
		Name:               channel,
		Enabled:            enabled,
		TopK:               topK,
		Timeout:            timeout,
		Budget:             topK,
		Weight:             weight,
		MinimumScore:       minScore,
		RelativeScoreFloor: relativeFloor,
	}
}
