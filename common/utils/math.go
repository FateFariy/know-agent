package utils

import "math"

// CosineSimilarity 计算两个等长向量的余弦相似度
func CosineSimilarity(left, right []float64) float64 {
	if len(left) == 0 || len(right) == 0 || len(left) != len(right) {
		return 0
	}
	var dot, lNorm, rNorm float64
	for i := 0; i < len(left); i++ {
		dot += left[i] * right[i]
		lNorm += left[i] * left[i]
		rNorm += right[i] * right[i]
	}
	if lNorm <= 0 || rNorm <= 0 {
		return 0
	}
	return dot / (math.Sqrt(lNorm) * math.Sqrt(rNorm))
}
