package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/evaluate"
)

// Sample 离线评测的单条样本
type Sample struct {
	// ID 样本标识（如数据集中的业务主键），仅用于结果溯源
	ID string `json:"id"`

	// Question 用户问题
	Question string `json:"question"`

	// Contexts 检索到的上下文片段。
	Contexts []string `json:"contexts"`

	// Answer RAG 系统的回答
	Answer string `json:"answer"`

	// GroundTruth 标准参考答案
	GroundTruth string `json:"ground_truth"`
}

// SetContexts 由检索器在评测前填充检索到的上下文片段。
func (s *Sample) SetContexts(contexts ...string) {
	s.Contexts = contexts
}

// toInput 转换为对话包定义的评估输入，供 evaluate 包评估器消费
func (s *Sample) toInput() *evaluate.EvaluationInput {
	return &evaluate.EvaluationInput{
		Question:    s.Question,
		Contexts:    s.Contexts,
		Answer:      s.Answer,
		GroundTruth: s.GroundTruth,
	}
}

// Dataset 离线评测数据集
type Dataset struct {
	// Name 数据集名称
	Name string `json:"name"`

	// Samples 评测样本集合
	Samples []*Sample `json:"samples"`
}

// DefaultDatasetDir ========== CRUD_QA 数据集加载（适配 eval/dataset/*.json） ==========
// 原始记录字段：id / questions / answers
//   - id         -> ID          样本标识（如数据集中的业务主键），仅用于结果溯源
//   - questions  -> Question    用户问题
//   - answers    -> GroundTruth 标准参考答案
//
// DefaultDatasetDir 同目录下的数据集目录
const DefaultDatasetDir = "dataset"

// LoadCRUDDataset 从单个 CRUD_QA 格式 JSON 文件加载离线评测数据集
func LoadCRUDDataset(path string) (*Dataset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取数据集失败: %w", err)
	}

	var records []map[string]any
	if err = json.Unmarshal(raw, &records); err != nil {
		return nil, fmt.Errorf("解析数据集JSON失败: %w", err)
	}
	if len(records) == 0 {
		return nil, ErrEmptyDataset
	}

	samples := make([]*Sample, 0, len(records))
	for _, rec := range records {
		sample := &Sample{
			ID:          strVal(rec["id"]),
			Question:    strVal(rec["questions"]),
			GroundTruth: strVal(rec["answers"]),
		}
		samples = append(samples, sample)
	}

	return &Dataset{
		Name:    baseName(path),
		Samples: samples,
	}, nil
}

// LoadCRUDDatasetDir 加载同目录下 dataset 目录中的全部 CRUD_QA JSON 数据集
func LoadCRUDDatasetDir(dir string) (*Dataset, error) {
	if dir == "" {
		dir = DefaultDatasetDir
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取数据集目录失败: %w", err)
	}

	var merged []*Sample
	seen := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		ds, err := LoadCRUDDataset(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		// 去重：同一 id 仅保留首次出现，避免不同 doc 数量数据集间的样本重复
		for _, s := range ds.Samples {
			if seen[s.ID] {
				continue
			}
			seen[s.ID] = true
			merged = append(merged, s)
		}
	}
	if len(merged) == 0 {
		return nil, ErrEmptyDataset
	}
	return &Dataset{Name: baseName(dir), Samples: merged}, nil
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func baseName(path string) string {
	if path == "" {
		return "dataset"
	}
	name := filepath.Base(path)
	for ext := len(name) - 1; ext >= 0; ext-- {
		if name[ext] == '.' {
			name = name[:ext]
			break
		}
	}
	return name
}
