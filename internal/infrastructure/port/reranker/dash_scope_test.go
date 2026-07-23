package reranker

import (
	"context"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"

	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

var svcCtx *svc.ServiceContext

func init() {
	var c config.Config
	conf.MustLoad("E:\\gocode\\ragent-convert\\know-agent\\etc\\config-prod.yaml", &c)
	svcCtx = svc.NewServiceContext(&c)
}

func TestDashScope(t *testing.T) {
	dashScope := NewDashScope(svcCtx)
	chunks, err := dashScope.Process(context.Background(), "RAG文档分块有什么方法？", []*vo.DocumentChunk{
		{
			Content: "文档切块 > RAG 有哪些文本分块策略？",
		},
		{
			Content: "文档切块 > RAG 文本分块策略有哪些？",
		},
		{
			Content: "文档切块 > 段落太长 / 长文档怎么处理？",
		},
		{
			Content: "文档切块 > 什么是父子chunk？有什么作用？",
		},
	})
	if err != nil {
		t.Error(err)
	}
	for _, chunk := range chunks {
		t.Log(chunk)
	}

}
