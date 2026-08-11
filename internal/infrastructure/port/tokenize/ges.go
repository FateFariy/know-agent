package tokenize

import "github.com/go-ego/gse"

// GseTokenizer 是 domain.Tokenizer 的 GSE 实现
type GseTokenizer struct {
	seg    gse.Segmenter
	config GseConfig
}

// GseConfig 用于配置 GSE 行为（停用词、词典、HMM 等）
type GseConfig struct {
	DictPath string // 词典文件路径
	StopPath string // 停用词文件路径
	UseHMM   bool   // 是否默认启用 HMM
	AlphaNum bool   // 是否保留字母数字组合
}

// NewGseTokenizer 创建基于 GSE 的分词器
func NewGseTokenizer(cfg GseConfig) (*GseTokenizer, error) {
	var seg gse.Segmenter
	seg.AlphaNum = cfg.AlphaNum

	// 加载词典
	if cfg.DictPath != "" {
		if err := seg.LoadDict(cfg.DictPath); err != nil {
			return nil, err
		}
	} else {
		_ = seg.LoadDict() // 使用内置默认词典
	}

	// 加载停用词
	if cfg.StopPath != "" {
		if err := seg.LoadStop(cfg.StopPath); err != nil {
			return nil, err
		}
	}

	return &GseTokenizer{
		seg:    seg,
		config: cfg,
	}, nil
}

// SegmentWords 分词并返回结果（暂时仅支持精确模式）
func (t *GseTokenizer) SegmentWords(text string) []string {
	return t.seg.Cut(text, t.config.UseHMM)
}
