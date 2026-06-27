package utils

import "strings"

func EqualsIgnoreCase(a, b string) bool {
	return strings.EqualFold(a, b)
}

func BlankToDefault(s string, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}
