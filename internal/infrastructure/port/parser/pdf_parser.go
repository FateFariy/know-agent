package parser

import (
	"bytes"
	"context"
	"strings"

	"github.com/ledongthuc/pdf"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/parse"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

const PDF = "pdf"

type PDFParser struct {
	textParser *TextParser
}

var _ parse.Parser = (*PDFParser)(nil)

func (p *PDFParser) Name() string {
	return PDF
}

func (p *PDFParser) Parse(_ context.Context, sourceText []byte) (entity.DocumentBlocks, error) {
	f, err := pdf.NewReader(bytes.NewReader(sourceText), int64(len(sourceText)))
	if err != nil {
		return nil, err
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

	blocks := make(entity.DocumentBlocks, 0)
	blocks = append(blocks, &entity.DocumentBlock{
		BlockNo:   1,
		BlockType: "TEXT",
		Text:      sb.String(),
		Metadata:  map[string]any{"parser": PDF},
	})
	return blocks, nil
}
