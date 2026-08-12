package utils

import "slices"

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

// FilterMapUniqueLimit 过滤并映射切片元素，按 Key 去重后返回前 n 个结果。
// 流程：Filter (bool) -> Map (K, V) -> Unique (K) -> Limit (n)。
// 当 n <= 0 时返回全部去重后的结果。
func FilterMapUniqueLimit[T, V any, K comparable](items []T, n int, keyOf func(T) (K, V, bool)) []V {
	if len(items) == 0 {
		return nil
	}
	if n <= 0 {
		n = len(items)
	}
	seen := make(map[K]struct{}, n)
	result := make([]V, 0, n)
	for _, v := range items {
		key, value, keep := keyOf(v)
		if !keep {
			continue
		}
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, value)
		}
		if len(result) >= n {
			break
		}
	}
	return result
}

// FilterUniqueLimit 过滤、去重并截断切片。
// 按 keyOf 返回的键去重，保留前 n 个；n<=0 返回全部。
// keyOf 的 bool 用于过滤，false 则跳过。
func FilterUniqueLimit[T any, K comparable](items []T, n int, keyOf func(T) (K, bool)) []T {
	return FilterMapUniqueLimit(items, n, func(item T) (K, T, bool) {
		key, keep := keyOf(item)
		return key, item, keep
	})
}

// FilterLimit 过滤并截断切片。
// filter 函数用于过滤，true 则保留，false 则跳过。
// n <= 0 时返回全部结果。
func FilterLimit[T any](items []T, n int, filter func(T) bool) []T {
	if len(items) == 0 {
		return nil
	}
	if n <= 0 {
		n = len(items)
	}
	result := make([]T, 0, n)
	for _, item := range items {
		if filter(item) {
			result = append(result, item)
		}
		if len(result) >= n {
			break
		}
	}
	return result
}

func ContainsAny[T comparable](slice []T, elements ...T) bool {
	for _, element := range elements {
		if slices.Index(slice, element) != -1 {
			return true
		}
	}
	return false
}
