package common

import (
	"database/sql/driver"
	"encoding/json"
)

// JSONArray 自定义JSON数组类型，用于处理json数组字段
type JSONArray []any

// Value 实现 driver.Valuer 接口
func (j *JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

func JSONArrayTo[T any](src JSONArray, parseFunc func(any) T) []T {
	if src == nil {
		return nil
	}

	var result = make([]T, len(src))
	for _, item := range src {
		result = append(result, parseFunc(item))
	}
	return result
}

func JSONArrayToIntSlice(src JSONArray) []int {
	return JSONArrayTo(src, func(item any) int {
		if val, ok := item.(int); ok {
			return val
		}
		return 0
	})
}

func ToJSONArray[T any](src []T) JSONArray {
	if src == nil {
		return nil
	}
	var result = make(JSONArray, len(src))
	for i, item := range src {
		result[i] = item
	}
	return result
}
