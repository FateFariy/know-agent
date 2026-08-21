package tokenize

import (
	"github.com/go-ego/gse"

	"github.com/swiftbit/know-agent/internal/svc"
)

// GseTokenizer 是 domain.Tokenizer 的 GSE 实现
type GseTokenizer struct {
	seg      gse.Segmenter
	DictPath string // 词典文件路径
	StopPath string // 停用词文件路径
	UseHMM   bool   // 是否默认启用 HMM
	AlphaNum bool   // 是否保留字母数字组合
}

// NewGseTokenizer 创建基于 GSE 的分词器
func NewGseTokenizer(svcCtx *svc.ServiceContext) *GseTokenizer {
	cfg := svcCtx.Config.Gse
	var seg gse.Segmenter
	seg.AlphaNum = cfg.AlphaNum

	// 加载词典
	if cfg.DictPath != "" {
		if err := seg.LoadDict(cfg.DictPath); err != nil {
			panic(err)
		}
	} else {
		_ = seg.LoadDict() // 使用内置默认词典
	}

	// 加载停用词
	if cfg.StopPath != "" {
		if err := seg.LoadStop(cfg.StopPath); err != nil {
			panic(err)
		}
	}

	return &GseTokenizer{
		seg:      seg,
		DictPath: cfg.DictPath,
		StopPath: cfg.StopPath,
		UseHMM:   cfg.UseHMM,
		AlphaNum: cfg.AlphaNum,
	}
}

// SegmentWords 分词并返回结果（暂时仅支持精确模式）
func (t *GseTokenizer) SegmentWords(text string) []string {
	return t.seg.Cut(text, t.UseHMM)
}
