package parser

import (
	"bytes"
	"context"

	"github.com/ledongthuc/pdf"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/parse"
)

var PDF = "pdf"

type PDFParser struct {
}

var _ parse.Parser = (*PDFParser)(nil)

func (p *PDFParser) Name() string {
	return PDF
}

func (p *PDFParser) Parse(ctx context.Context, bytesData []byte) (string, error) {
	f, err := pdf.NewReader(bytes.NewReader(bytesData), int64(len(bytesData)))
	if err != nil {
		return "", err
	}

	var textBuffer bytes.Buffer
	pages := f.NumPage()
	for i := 1; i <= pages; i++ {
		page := f.Page(i)

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		textBuffer.WriteString(text)
		textBuffer.WriteString("\n")
	}

	return textBuffer.String(), nil
}
