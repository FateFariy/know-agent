package utils

func SliceToMapBy[T any, K comparable, V any](slice []T, keyFunc func(T) (K, V)) map[K]V {
	if slice == nil {
		return nil
	}

	if len(slice) == 0 {
		return map[K]V{}
	}

	maps := make(map[K]V, len(slice))
	for _, item := range slice {
		key, value := keyFunc(item)
		maps[key] = value
	}
	return maps
}

func LimitSlice[T any](slice []T, limit int) []T {
	if slice == nil {
		return nil
	}

	if len(slice) <= limit {
		return slice
	}

	return slice[:limit]
}

func Distinct[T any, V comparable](slice []T, keyFunc func(T) V) []T {
	if len(slice) == 0 {
		return nil
	}
	seen := make(map[V]struct{})
	uniqueSlice := make([]T, 0, len(slice))
	for _, ref := range slice {
		key := keyFunc(ref)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			uniqueSlice = append(uniqueSlice, ref)
		}
	}

	return uniqueSlice
}

// DistinctFilterLimit 先按 keyOf 提取键对切片去重，并保留前 n 个结果；
// keyOf 返回的 bool 值用于元素级的选择过滤，false 时跳过该元素。
//   - items: 源切片；为空时直接返回 nil
//   - n: 保留的最大数量（≤0 时返回空切片）
//   - keyOf: (键, 是否保留) —— 键用于去重，bool 为 false 时跳过该元素
func DistinctFilterLimit[T any, V comparable](items []T, n int, keyOf func(T) (V, bool)) []T {
	if len(items) == 0 || n <= 0 {
		return nil
	}
	seen := make(map[V]struct{}, n)
	result := make([]T, 0, n)
	for _, v := range items {
		key, keep := keyOf(v)
		if !keep {
			continue
		}
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, v)
		}
		if len(result) >= n {
			break
		}
	}
	return result
}
