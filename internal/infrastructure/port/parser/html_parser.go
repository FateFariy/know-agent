package parser

import (
	"bytes"
	"context"

	"github.com/PuerkitoBio/goquery"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/parse"
)

var HTML = "html"

type HTMLParser struct {
}

var _ parse.Parser = (*HTMLParser)(nil)

func (p *HTMLParser) Name() string {
	return HTML
}

func (p *HTMLParser) Parse(ctx context.Context, bytesData []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(bytesData))
	if err != nil {
		return string(bytesData), nil
	}

	return doc.Text(), nil
}
