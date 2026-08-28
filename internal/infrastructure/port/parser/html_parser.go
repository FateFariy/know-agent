package parser

import (
	"context"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/parse"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

const Html = "html"

type HTMLParser struct {
}

var _ parse.Parser = (*HTMLParser)(nil)

func (p *HTMLParser) Name() string {
	return Html
}

func (p *HTMLParser) Parse(_ context.Context, sourceText []byte) (entity.DocumentBlocks, error) {
	decoded := decodeText(sourceText)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(decoded))
	if err != nil {
		return nil, err
	}

	var blocks entity.DocumentBlocks
	blockNo := 1

	doc.Find("body").Children().Each(func(i int, s *goquery.Selection) {
		text := utils.CleanupSpace(s.Text())
		if text == "" {
			return
		}

		blockType := "TEXT"
		tag := goquery.NodeName(s)
		switch tag {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			blockType = "TITLE"
		default:
			blockType = classifyTextBlock(text)
		}

		blocks = append(blocks, &entity.DocumentBlock{
			BlockNo:   blockNo,
			BlockType: blockType,
			Text:      text,
			Metadata:  map[string]any{"parser": Html},
		})
		blockNo++
	})

	return blocks, nil
}
