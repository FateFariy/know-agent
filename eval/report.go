package eval

import (
	"encoding/json"
	"errors"
	"fmt"
)

// 专用错误，便于调用方区分离线评测的边界条件
var (
	ErrEmptyDataset = errors.New("eval: 数据集为空，无法执行离线评测")
	ErrNoEvaluator  = errors.New("eval: 未提供任何评估器")
)

// MetricScore 单条样本中某个指标的具体得分
type MetricScore struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
	Err   string  `json:"error,omitempty"`
}

// SampleResult 单条样本的评测结果（含所有指标明细）
type SampleResult struct {
	ID     string        `json:"id"`
	Scores []MetricScore `json:"scores"`
}

// MetricSummary 某个指标的聚合统计
type MetricSummary struct {
	Name        string  `json:"name"`
	Mean        float64 `json:"mean"`
	SampleCount int     `json:"sample_count"`
	ErrorCount  int     `json:"error_count"`
}

// Report 离线评测报告
type Report struct {
	DatasetName string          `json:"dataset_name"`
	SampleCount int             `json:"sample_count"`
	Metrics     []MetricSummary `json:"metrics"`
	Details     []SampleResult  `json:"details"`
}

// MeanScore 取得指定指标的平均分，指标不存在时返回 0。
func (r *Report) MeanScore(metricName string) float64 {
	for _, m := range r.Metrics {
		if m.Name == metricName {
			return m.Mean
		}
	}
	return 0
}

// String 以可读 JSON 形式输出报告，便于日志与离线归档。
func (r *Report) String() string {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf("Report{marshal error: %v}", err)
	}
	return string(b)
}
