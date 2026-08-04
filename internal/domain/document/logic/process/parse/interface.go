package parse

import "context"

type Parser interface {
	// Name 获取解析器名称
	Name() string

	// Parse 解析文件
	Parse(ctx context.Context, bytesData []byte) (string, error)
}
