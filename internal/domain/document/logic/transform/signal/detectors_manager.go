package signal

import (
	"sort"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// DetectorsManager 检测器管理器
type DetectorsManager struct {
	detectors []Detector // 检测器列表，按优先级排序
}

// NewDefaultDetectorsManager 创建默认检测器管理器，注册所有内置检测器并按优先级排序
func NewDefaultDetectorsManager() *DetectorsManager {
	mgr := &DetectorsManager{
		detectors: make([]Detector, 0),
	}
	mgr.registerDefaultDetectors()
	return mgr
}

// registerDefaultDetectors 注册默认检测器，按优先级顺序添加（Order 值越小越优先），注册后会自动排序
func (m *DetectorsManager) registerDefaultDetectors() {
	m.detectors = append(m.detectors, &BlankDetector{})            // Order=0  - 空行检测
	m.detectors = append(m.detectors, &NoiseDetector{})            // Order=10 - 噪声检测（页码、版权等）
	m.detectors = append(m.detectors, &MarkdownHeadingDetector{})  // Order=20 - Markdown 标题检测
	m.detectors = append(m.detectors, &ExplicitStepDetector{})     // Order=30 - 步骤编号检测（第X步）
	m.detectors = append(m.detectors, &ChapterHeadingDetector{})   // Order=40 - 章节标题检测（第X章）
	m.detectors = append(m.detectors, &AppendixHeadingDetector{})  // Order=50 - 附录标题检测
	m.detectors = append(m.detectors, &DecimalHeadingDetector{})   // Order=60 - 数字编号标题检测（1.1.1）
	m.detectors = append(m.detectors, &TableRowDetector{})         // Order=70 - 表格行检测
	m.detectors = append(m.detectors, &QuoteDetector{})            // Order=80 - 引用检测（>）
	m.detectors = append(m.detectors, &ListItemDetector{})         // Order=90 - 列表项检测（- * +）
	m.detectors = append(m.detectors, &SingleLevelDigitDetector{}) // Order=100 - 单层数字编号检测（1. 2.）
	m.detectors = append(m.detectors, &ChineseOutlineDetector{})   // Order=110 - 中文大纲检测（一、二、）

	// 按优先级排序，确保检测顺序正确
	sort.Slice(m.detectors, func(i, j int) bool {
		return m.detectors[i].Order() < m.detectors[j].Order()
	})
}

// Register 注册自定义检测器，按优先级排序
func (m *DetectorsManager) Register(detector Detector) {
	if detector == nil {
		return
	}
	m.detectors = append(m.detectors, detector)
	sort.Slice(m.detectors, func(i, j int) bool {
		return m.detectors[i].Order() < m.detectors[j].Order()
	})
}

// Detect 检测文本行, 返回结构信号, 按优先级顺序执行所有检测器，第一个匹配成功的返回结果
func (m *DetectorsManager) Detect(detCtx *DetectorContext, text string, opts ...DetectorOption) *vo.DocumentStructureSignal {
	for _, detector := range m.detectors {
		result := detector.Detect(detCtx, text, opts...)
		if result != nil {
			return result
		}
	}
	return nil
}
