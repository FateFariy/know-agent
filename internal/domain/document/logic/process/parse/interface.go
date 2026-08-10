package parse

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// Parser 文件解析器接口
type Parser interface {
	// Name 获取解析器名称
	Name() string
	// Parse 解析文件
	Parse(ctx context.Context, sourceText []byte) (entity.DocumentBlocks, error)
}
