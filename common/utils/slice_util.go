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
