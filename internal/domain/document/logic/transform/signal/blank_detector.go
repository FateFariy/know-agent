package signal

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// BlankDetector 空行检测器, 识别完全空白的文本行
type BlankDetector struct{}

// Detect 检测空行
func (d *BlankDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindBlank,
			Confidence: 1.0,
		}
	}
	return nil
}

func (d *BlankDetector) Order() int {
	return 0
}
