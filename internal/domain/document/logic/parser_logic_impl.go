package logic

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type ParserLogicImpl struct {
}

var _ ParserLogic = (*ParserLogicImpl)(nil)

func NewParserLogicImpl() *ParserLogicImpl {
	return &ParserLogicImpl{}
}

func (p *ParserLogicImpl) Parse(ctx context.Context, bytes []byte, originalFileName string, mimeType, fileType int) (*vo.DocumentAnalysisResult, error) {
	// TODO implement me
	panic("implement me")
}
