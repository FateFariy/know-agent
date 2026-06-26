package parser

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/parse"
)

var TEXT = "text"

type TextParser struct {
}

var _ parse.Parser = (*TextParser)(nil)

func (p *TextParser) Name() string {
	return TEXT
}

func (p *TextParser) Parse(ctx context.Context, bytesData []byte) (string, error) {
	return string(bytesData), nil
}
