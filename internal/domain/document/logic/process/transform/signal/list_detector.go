package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// 列表项检测正则表达式
var (
	checkboxPattern = regexp.MustCompile(`^\[[ xX]]\s+(.+)$`) // 复选框列表（如：[ ] 待办、[x] 已完成）
	bulletPattern   = regexp.MustCompile(`^([-*+•])\s+(.+)$`) // 无序列表（如：- 项目、* 项目、+ 项目、• 项目）
)

// ListItemDetector 列表项检测器, 检测无序列表项和复选框列表项
type ListItemDetector struct{}

// Detect 检测列表项, 优先匹配复选框列表，再匹配无序列表
func (d *ListItemDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	// 匹配复选框列表（如：[ ] 待办事项、[x] 已完成）
	if matches := checkboxPattern.FindStringSubmatch(text); len(matches) == 2 {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindListItem,
			Title:      strutil.Trim(matches[1]),
			Reasons:    []string{"checkbox-list"},
			Confidence: 0.92,
		}
	}

	// 匹配无序列表（如：- 项目、* 项目、+ 项目）
	if matches := bulletPattern.FindStringSubmatch(text); len(matches) == 3 {
		return &vo.DocumentStructureSignal{
			Kind:       vo.SignalKindListItem,
			Title:      strutil.Trim(matches[2]),
			Reasons:    []string{"bullet-list"},
			Confidence: 0.90,
		}
	}

	return nil
}

func (d *ListItemDetector) Order() int {
	return 90
}
