package semantic

import "context"

// Similarity 文本相似度计算接口
type Similarity interface {
	// Calculate 计算两个文本的相似度
	Calculate(ctx context.Context, text1, text2 string) (float64, error)
}
