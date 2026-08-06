package parse

import (
	"context"
)

// Parser 文件解析器接口
type Parser interface {
	// Name 获取解析器名称
	Name() string
	// Parse 解析文件
	Parse(ctx context.Context, bytesData []byte) (string, error)
}
