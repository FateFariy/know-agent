package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
)

// ExtractJsonObject 从文本中抽取首个 { 到末个 } 之间的内容；找不到则返回原文
func ExtractJsonObject(raw string) string {
	return extractJson(raw, "{", "}")
}

// ExtractJsonArray 从文本中抽取首个 [ 到末个 ] 之间的内容；找不到则返回原文
func ExtractJsonArray(raw string) string {
	return extractJson(raw, "[", "]")
}

// Unmarshal 根据 dest 的类型动态选择抽取策略：结构体走 ExtractJsonObject，切片走 ExtractJsonArray
// 其他类型（map、基础类型等）走默认的 ExtractJsonObject
func Unmarshal[T any](raw string, dest *T) error {
	if dest == nil {
		return fmt.Errorf("dest 不能为 nil")
	}

	// 通过反射判断目标类型的 kind，选择合适的 JSON 抽取函数
	rt := reflect.TypeOf(*dest)
	extractor := ExtractJsonObject
	if rt != nil && rt.Kind() == reflect.Slice {
		extractor = ExtractJsonArray
	}

	// 抽取 JSON 片段，再交给标准库反序列化
	raw = extractor(raw)
	return json.Unmarshal([]byte(raw), dest)
}

// extractJson 从文本中抽取首个 str1 到末个 str2 之间的内容；找不到则返回原文
func extractJson(raw, str1, str2 string) string {
	trimmed := strutil.Trim(raw)
	if strutil.IsBlank(trimmed) {
		return trimmed
	}

	start := strings.Index(trimmed, str1)
	end := strings.LastIndex(trimmed, str2)
	if start == -1 || end == -1 || end < start {
		return trimmed
	}
	return trimmed[start : end+1]
}

// ToCompactJSON 将任意切片序列化为紧凑 JSON
func ToCompactJSON[T any](v []T) string {
	data, err := json.Marshal(v)
	if err != nil || len(data) == 0 || string(data) == "null" {
		return "[]"
	}
	return string(data)
}
