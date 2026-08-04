package signal

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// DetectorContext 检测器上下文，包含文档级别的共享信息
type DetectorContext struct {
	DocumentTitle string          // 文档标题，用于识别重复标题噪声
	LineFrequency map[string]int  // 行文本出现频率，用于检测重复噪声（如页眉页脚）
	LineContext   *vo.LineContext // 当前行的上下文信息（前后行、空白行等）
}

type DetectOption struct {
	maxPlainHeadingChars int
}

type DetectorOption func(*DetectOption)

// WithMaxPlainHeadingChars 设置最大普通标题字符数
func WithMaxPlainHeadingChars(maxPlainHeadingChars int) DetectorOption {
	if maxPlainHeadingChars <= 0 {
		maxPlainHeadingChars = 80
	}
	return func(opts *DetectOption) {
		opts.maxPlainHeadingChars = maxPlainHeadingChars
	}
}

// GetDetectorOptions 获取检测器选项
func GetDetectorOptions(base *DetectOption, opts ...DetectorOption) *DetectOption {
	if base == nil {
		base = new(DetectOption)
	}

	for i := range opts {
		opt := opts[i]
		if opt != nil {
			opt(base)
		}
	}

	return base
}

// Detector 文档结构信号检测器接口, 实现者按优先级顺序执行, 第一个匹配成功的检测器返回结果（责任链模式）
type Detector interface {
	// Detect 检测文本行，返回结构信号；不匹配时返回 nil
	Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal

	// Order 返回检测器优先级，数值越小越优先执行
	Order() int
}

// NewDetectorContext 创建检测器上下文
func NewDetectorContext(documentTitle string, lineFrequency map[string]int) *DetectorContext {
	return &DetectorContext{
		DocumentTitle: documentTitle,
		LineFrequency: lineFrequency,
	}
}
