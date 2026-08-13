package score

// Scorer 定义评分能力，方便扩展不同加权策略或机器学习模型。
type Scorer interface {
	Score(features *Features) *Result
}
