package utils

func PointerOrDefault[T any](value *T, defaultValue T) T {
	if value == nil {
		return defaultValue
	}
	return *value
}

func Pointer[T any](value T) *T {
	return &value
}
