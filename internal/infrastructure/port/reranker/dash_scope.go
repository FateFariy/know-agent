package reranker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpc"

	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

type rerankFullResp struct {
	RequestID string        `json:"request_id"`
	Output    *rerankOutput `json:"output,omitempty"`
	Results   []rerankItem  `json:"results,omitempty"`
	Usage     rerankUsage   `json:"usage"`
	Code      string        `json:"code,omitempty"`
	Message   string        `json:"message,omitempty"`
}

type rerankOutput struct {
	Results []rerankItem `json:"results"`
}

// 单条排序结果
type rerankItem struct {
	Document *rerankDoc `json:"document,omitempty"`
	Index    int        `json:"index"`
	Score    float64    `json:"relevance_score"`
}

// 文档对象
type rerankDoc struct {
	Text string `json:"text"`
}

// token用量
type rerankUsage struct {
	TotalTokens int `json:"total_tokens"`
}

type DashScope struct {
	URL    string
	ApiKey string
	*adapter.RerankOption
}

var _ adapter.Reranker = (*DashScope)(nil)

func NewDashScope(svcCtx *svc.ServiceContext) *DashScope {
	return &DashScope{
		URL:    svcCtx.Config.Chat.Rag.Rerank.URL,
		ApiKey: svcCtx.Config.Chat.Rag.Rerank.ApiKey,
		RerankOption: &adapter.RerankOption{
			Model: svcCtx.Config.Chat.Rag.Rerank.Model,
			TopN:  svcCtx.Config.Chat.Rag.Rerank.TopN,
		},
	}
}

// Process 调用 DashScope Rerank API 对传入的文档块进行重排序
func (d *DashScope) Process(ctx context.Context, question string, chunks []*vo.DocumentChunk, opts ...adapter.Option) ([]*vo.DocumentChunk, error) {
	// 合并外部选项
	d.RerankOption = adapter.GetCommonOptions(d.RerankOption, opts...)

	// 调用 DashScope Rerank HTTP API
	start := time.Now()
	resp, err := d.doRerankRequest(ctx, question, chunks)
	if err != nil {
		return nil, err
	}
	durationMs := time.Since(start).Milliseconds()
	logx.Infof("rerank duration: %dms", durationMs)

	// 遍历 API 返回结果，回填每个 chunk 的 RerankScore / RerankOriginalIndex / RerankQuery / RerankModel
	results := make([]*vo.DocumentChunk, 0, len(chunks))
	for _, result := range resp.Results {
		chunk := chunks[result.Index]
		chunk.RerankScore = result.Score
		chunk.RerankOriginalIndex = result.Index
		chunk.RerankQuery = chunk.Content
		chunk.RerankModel = d.Model
		results = append(results, chunk)
	}
	// 先按 RerankScore 降序排序整个 results 切片
	sort.Slice(results, func(i, j int) bool {
		return results[i].RerankScore > results[j].RerankScore
	})

	return results, nil
}

type rerankReq struct {
	Authorization string   `header:"Authorization"`
	ContentType   string   `header:"Content-Type"`
	Model         string   `json:"model"`
	Query         string   `json:"query"`
	Documents     []string `json:"documents"`
	TopN          int      `json:"top_n"`
}

// doRerankRequest 向 DashScope 发起 Rerank HTTP 请求并解析返回结果
func (d *DashScope) doRerankRequest(ctx context.Context, question string, chunks []*vo.DocumentChunk) (*rerankFullResp, error) {
	req := &rerankReq{
		Model:         d.Model,
		Query:         question,
		Documents:     slice.Map(chunks, func(_ int, chunk *vo.DocumentChunk) string { return chunk.Content }),
		TopN:          d.TopN,
		Authorization: "Bearer " + d.ApiKey,
		ContentType:   "application/json",
	}
	// 发送 HTTP 请求
	resp, err := httpc.Do(ctx, http.MethodPost, d.URL, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// 读取响应体并反序列化为 rerankFullResp
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var respBody rerankFullResp
	if err = json.Unmarshal(bodyBytes, &respBody); err != nil {
		return nil, err
	}

	// 检查 DashScope 错误码与消息
	if respBody.Code != "" || respBody.Message != "" {
		return nil, fmt.Errorf("dash scope rerank failed, msg: %s", respBody.Message)
	}

	return &respBody, nil
}
