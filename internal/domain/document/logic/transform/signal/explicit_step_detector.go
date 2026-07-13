package signal

import (
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// explicitStepPattern 步骤编号模式
//
// 支持格式：第1步、第一步、步骤1、步骤一（含中英文数字）
// 捕获组1："第X步"中的数字，捕获组2："步骤X"中的数字，捕获组3：步骤内容
var explicitStepPattern = regexp.MustCompile(`^(?:第\s*([0-9一二三四五六七八九十百]+)\s*步|步骤\s*([0-9一二三四五六七八九十百]+))\s*[:：、.]?\s*(.+)$`)

// ExplicitStepDetector 显式步骤检测器, 检测明确的步骤编号（第X步、步骤X）
type ExplicitStepDetector struct {
	BaseDetector
}

// Detect 检测步骤编号, 支持中英文数字：第1步、第一步、步骤1、步骤一
func (d *ExplicitStepDetector) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	if text == "" {
		return nil
	}

	matches := explicitStepPattern.FindStringSubmatch(text)
	if len(matches) < 4 {
		return nil // 需要至少4个捕获组（完整匹配+3个捕获组）
	}

	// 合并两个数字捕获组（取非空的那个）
	itemIndex := utils.ParseChineseNumber(utils.BlankToDefault(strutil.Trim(matches[1]), strutil.Trim(matches[2])))

	return &vo.DocumentStructureSignal{
		Kind:       vo.SignalKindStepItem,
		Title:      strutil.Trim(matches[3]), // 步骤内容
		ItemIndex:  itemIndex,                // 步骤序号
		Reasons:    []string{"explicit-step"},
		Confidence: 0.96,
	}
}

func (d *ExplicitStepDetector) Order() int {
	return 30
}
