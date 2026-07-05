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
