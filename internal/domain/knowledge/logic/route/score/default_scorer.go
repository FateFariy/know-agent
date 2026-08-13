package score

import (
	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
)

type DefaultScorer struct {
	*options
}

type Option = common.Option

type options struct {
	SemanticThreshold float64 // 语义有效阈值，低于此值不计分，默认 0.20
	SemanticWeight    float64 // 语义分放大系数，默认 50
	LexicalWeight     float64 // 词索引分放大系数，默认 1.6
	LexicalMax        float64 // 词索引分上限，默认 10
	RelationWeight    float64 // 关系分权重（不同场景可设置不同值，如 scope 用 8，topic 用 20）
}

func WithRelationWeight(weight float64) Option {
	return common.WrapImplSpecificOptFn(func(o *options) {
		o.RelationWeight = weight
	})
}

// NewDefaultScorer 返回带默认参数的评分器，调用方可按需覆盖。
func NewDefaultScorer() *DefaultScorer {
	return &DefaultScorer{&options{
		SemanticThreshold: 0.20,
		SemanticWeight:    50.0,
		LexicalWeight:     1.6,
		LexicalMax:        10.0,
		RelationWeight:    8.0,
	}}
}

// Score 实现 Scorer 接口。
func (s *DefaultScorer) Score(features *Features, opts ...common.Option) *Result {
	s.options = common.GetImplSpecificOptions(s.options, opts...)

	semMain := s.semanticMainScore(features.SemanticScore)
	lex := s.lexicalAssist(features.LexicalScore)
	rel := features.RelationScore * s.RelationWeight
	total := semMain + lex + rel

	semanticMatch := semMain > 0
	lexicalMatch := features.LexicalScore > 0
	relationMatch := features.RelationScore > 0
	return &Result{
		TotalScore: total,
		Reason:     s.buildReason(semanticMatch, lexicalMatch, relationMatch),
		Source:     s.resolveSource(semanticMatch, lexicalMatch, relationMatch),
		Features: map[string]float64{
			"semanticScore":    features.SemanticScore,
			"lexicalScore":     features.LexicalScore,
			"relationScore":    features.RelationScore,
			"semanticMain":     semMain,
			"lexicalAssist":    lex,
			"relationWeighted": rel,
		},
	}
}

// ---------- 内部评分函数 ----------
func (s *DefaultScorer) semanticMainScore(semantic float64) float64 {
	if semantic <= s.SemanticThreshold {
		return 0
	}
	return (semantic - s.SemanticThreshold) * s.SemanticWeight
}

func (s *DefaultScorer) lexicalAssist(lexical float64) float64 {
	if lexical <= 0 {
		return 0
	}
	val := lexical * s.LexicalWeight
	return max(val, s.LexicalMax)
}

// ---------- 原因与来源生成 ----------
func (s *DefaultScorer) buildReason(semMatch, lexMatch, relMatch bool) string {
	switch {
	case semMatch && lexMatch && relMatch:
		return "语义、路由索引与持久化关系特征共同召回"
	case semMatch && lexMatch:
		return "语义与路由索引特征共同召回"
	case lexMatch && relMatch:
		return "路由索引与持久化关系特征共同召回"
	case semMatch && relMatch:
		return "语义与持久化关系特征共同召回"
	case lexMatch:
		return "由路由索引 BM25 特征召回"
	case semMatch:
		return "由持久化画像的语义相似度召回"
	case relMatch:
		return "由持久化关系特征召回"
	default:
		return "没有形成有效路由特征"
	}
}

func (s *DefaultScorer) resolveSource(semMatch, lexMatch, relMatch bool) string {
	count, source := 0, "NONE"
	if semMatch {
		count++
		source = "SEMANTIC"
	}
	if lexMatch {
		count++
		source = "ROUTE_INDEX"
	}
	if relMatch {
		count++
		source = "PERSISTED_RELATION"
	}
	return utils.Ternary(count > 1, "COMPOSITE", source)
}
