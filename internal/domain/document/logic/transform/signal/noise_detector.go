package signal

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// 噪声检测正则表达式
var (
	pageNoisePattern      = regexp.MustCompile(`^(?:第\s*\d+\s*页|Page\s*\d+|\d+\s*/\s*\d+)$`)              // 页码模式（如：第1页、Page 5、1/10）
	copyrightNoisePattern = regexp.MustCompile(`.*(?:版权所有|未经授权|内部使用|copyright|all rights reserved|保密).*`) // 版权声明
	versionFooterPattern  = regexp.MustCompile(`.*(?:\bV\d+(?:\.\d+)*\b|版本|修订|Rev\.?\s*\d+).*`)           // 版本信息（如：V1.0、版本2.0）
)

// NoiseDetector 噪声检测器, 检测文档中的噪声内容（页码、版权声明、重复标题、页脚等）
type NoiseDetector struct {
	BaseDetector
}

// Detect 检测噪声内容, 通过行频率和模式匹配识别噪声，优先级顺序：重复标题 > 版权声明 > 重复页脚 > 页码
func (d *NoiseDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil // 空行由 BlankDetector 处理
	}

	// 获取当前行的出现频率（用于识别重复的页眉页脚）
	frequency := utils.Ternary(detCtx.LineFrequency == nil, 0, detCtx.LineFrequency[text])

	// 频率 >= 2 的行可能是噪声
	if frequency >= 2 {
		noise := &vo.DocumentStructureSignal{Kind: vo.SignalKindNoise, Confidence: 0.99}

		// 重复的文档标题（出现在每一页的页眉）
		if d.sameDocumentTitle(detCtx.DocumentTitle, text) {
			noise.Reasons = []string{"duplicate-document-title"}
			return noise
		}

		// 版权声明类噪声
		if copyrightNoisePattern.MatchString(text) {
			noise.Reasons = []string{"copyright-noise"}
			return noise
		}

		// 重复出现的版本信息或分隔符行（频率 >= 3 且长度 <= 120）
		if frequency >= 3 && utf8.RuneCountInString(text) <= 120 && (versionFooterPattern.MatchString(text) || strings.Contains(text, "|")) {
			noise.Reasons = []string{"version-footer-noise"}
			return noise
		}
	}

	// 页码噪声（单独处理，不受频率限制）
	if pageNoisePattern.MatchString(text) {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindNoise,
			Reasons:    []string{"page-noise"},
			Confidence: 0.98,
		}
	}

	return nil
}

func (d *NoiseDetector) Order() int {
	return 10
}
