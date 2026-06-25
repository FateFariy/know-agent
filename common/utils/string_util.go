package utils

import "strings"

func EqualsIgnoreCase(a, b string) bool {
	return strings.ToLower(a) == strings.ToLower(b)
}

func BlankToDefault(s string, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}
