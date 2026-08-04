package signal

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// QuoteDetector 引用检测器, 检测以 > 开头的引用行（Markdown 引用格式），优先级 Order=80
type QuoteDetector struct{}

// Detect 检测引用行, 判断条件：首字符为 >（Markdown 引用符号）
func (d *QuoteDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	if text[0] == '>' {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindQuote,
			Title:      text,
			Reasons:    []string{"quote"},
			Confidence: 0.88,
		}
	}

	return nil
}

func (d *QuoteDetector) Order() int {
	return 80
}
