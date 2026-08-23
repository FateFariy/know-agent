package enum

type MetricName = string

const (
	AnswerFaithfulness MetricName = "answer_faithfulness" // 答案忠实度
	AnswerRelevancy    MetricName = "answer_relevancy"    // 答案相关性
	ContextRecall      MetricName = "context_recall"      // 上下文召回率
	ContextPrecision   MetricName = "context_precision"   // 上下文精确度
)
