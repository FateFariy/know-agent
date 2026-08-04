package parser

import (
	"bytes"
	"context"
	"strings"

	"github.com/ledongthuc/pdf"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/parse"
)

const PDF = "pdf"

type PDFParser struct {
}

var _ parse.Parser = (*PDFParser)(nil)

func (p *PDFParser) Name() string {
	return PDF
}

func (p *PDFParser) Parse(_ context.Context, bytesData []byte) (string, error) {
	f, err := pdf.NewReader(bytes.NewReader(bytesData), int64(len(bytesData)))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for i := 1; i <= f.NumPage(); i++ {
		page := f.Page(i)
		if page.V.IsNull() || page.V.Key("Contents").Kind() == pdf.Null {
			continue
		}

		rows, _ := page.GetTextByRow()
		for _, row := range rows {
			for _, word := range row.Content {
				sb.WriteString(word.S)
			}
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
