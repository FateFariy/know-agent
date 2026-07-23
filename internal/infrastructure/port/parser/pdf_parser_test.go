package parser

import (
	"context"
	"os"
	"testing"
)

func TestPdfParse(t *testing.T) {
	data, err := os.ReadFile("document.pdf")
	if err != nil {
		t.Error(err)
	}

	parser := &PDFParser{}
	text, err := parser.Parse(context.TODO(), data)
	if err != nil {
		t.Error(err)
	}
	t.Log(text)
}
