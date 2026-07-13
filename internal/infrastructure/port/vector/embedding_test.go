package vector

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/conf"

	"github.com/swiftbit/know-agent/internal/config"
	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

var svcCtx *svc.ServiceContext
var chatModel *chatlogic.ChatModelImpl[*schema.AgenticMessage]

func init() {
	configFile := "E:/gocode/ragent-convert/know-agent/etc/config-dev.yaml"
	var c config.Config
	conf.MustLoad(configFile, &c)
	svcCtx = svc.NewServiceContext(&c)
	chatModel = chatlogic.NewChatModelImpl(svcCtx)
}

func TestEmbedding(t *testing.T) {
	retriever := NewMilvusVector(svcCtx)
	ctx := context.Background()
	// 构造单条检索请求测试数据
	retrieveReq := &vo.DocumentRetrieve{
		Question:       "RAG系统如何优化文档分块提升检索精度",
		RetrievalQuery: "大模型RAG分层分块策略、向量检索降噪方案",
		DocumentId:     10001,
		TaskId:         5001,
		DocumentIds:    []int64{10001, 10003, 10005},
		TaskIds:        []int64{5001, 5002},
		TopK:           8,
		Filters: &vo.DocumentRetrieveFilters{
			DocumentNameHints:     []string{"知识库分块规范文档", "向量检索优化白皮书"},
			BusinessCategoryHints: []string{"大模型应用", "智能检索"},
			DocumentTagHints:      []string{"RAG", "向量数据库", "分块策略"},
			SectionPathHints:      []string{"第一章/第一节", "第一章/第一节/数据表"},
			CanonicalPathHints:    []string{"/doc/10001/chapter1/section1", "/doc/10001/chapter1/section1/table1"},
			StructureNodeIdHints:  []int64{8001, 8008},
			ItemIndexHints:        []int{0, 1, 2},
			YearHints:             []string{"2025", "2026"},
		},
		QueryContextHints: []string{"当前场景：企业内部知识库问答", "禁止返回无关政策文档"},
	}
	documents, err := retriever.Search(ctx, retrieveReq)
	if err != nil {
		log.Fatalf("Failed to retrieve: %v", err)
		return
	}

	// 打印文档
	for i, doc := range documents {
		fmt.Printf("Document %d:\n", i)
		fmt.Printf("  ID: %s\n", doc.ID)
		fmt.Printf("  Content: %s\n", doc.Content)
		fmt.Printf("  Score: %v\n", doc.Score)
	}
}
