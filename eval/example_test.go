package eval_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"

	"github.com/swiftbit/know-agent/eval"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/evaluate"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/retrieval/fuse"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/gateway"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/keyword"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/llm"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/prompt"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/rerank"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/storage"
	"github.com/swiftbit/know-agent/internal/infrastructure/port/vector"
	"github.com/swiftbit/know-agent/internal/svc"
)

// ExampleRetrievalAndEvaluate 演示完整的离线评测流水线：
//  1. 初始化基础设施（ServiceContext、向量库、关键词、LLM、重排序等）
//  2. 创建 RAG 检索引擎
//  3. 加载 CRUD_QA 数据集
//  4. 对每条样本执行检索，获取上下文
//  5. 用 ContextPrecision / ContextRecall 评估器计算指标
//  6. 产出聚合报告
//
// 注意：依赖 MySQL、Redis、Milvus、Minio 等基础设施，需提前就绪。
func TestRetrievalAndEvaluate(t *testing.T) {
	// ========== 0. 加载配置 ==========
	var c config.Config
	conf.MustLoad("etc/config-prod.yaml", &c)

	// ========== 1. 初始化基础设施 ==========
	serviceContext := svc.NewServiceContext(&c)
	chatModel := llm.NewChatModelImpl(serviceContext)
	renderer := prompt.NewRendererImpl()
	milvusVector := vector.NewMilvusVector(serviceContext)
	milvusKeyword := keyword.NewMilvusKeyword(serviceContext)
	minioStorage := storage.NewMinioStorage(serviceContext)
	dashScope := rerank.NewDashScope(serviceContext)
	documentRepo := persistence.NewDocumentRepository(serviceContext, minioStorage, milvusVector)
	chatRepo := persistence.NewChatRepository(serviceContext)
	documentForChat := gateway.NewDocumentAdapterForChat(documentRepo)

	// ========== 2. 创建 RAG 检索引擎 ==========
	channels := []retrieval.Retrieval{
		channel.NewVectorRetrievalChannel(serviceContext, milvusVector),
		channel.NewKeywordRetrievalChannel(serviceContext, milvusKeyword),
	}
	rrfFusion := fuse.NewRRFFusion()
	retrievalEngine := retrieval.NewRetrievalEngine(serviceContext, chatRepo, dashScope, channels, documentForChat, rrfFusion)

	// ========== 3. 创建评估器（仅 ContextPrecision + ContextRecall） ==========
	evaluators := []evaluate.Evaluator{
		evaluate.NewContextPrecisionEvaluator(chatModel, renderer),
		evaluate.NewContextRecallEvaluator(chatModel, renderer),
	}
	runner := eval.NewRunner(evaluators...)

	// ========== 4. 加载数据集 ==========
	dataset, err := eval.LoadCRUDDatasetDir("dataset")
	if err != nil {
		panic(fmt.Sprintf("加载数据集失败: %v", err))
	}
	fmt.Printf("数据集加载完成: name=%s, samples=%d\n", dataset.Name, len(dataset.Samples))

	// ========== 5. 对每条样本执行检索，填入上下文 ==========
	for _, sample := range dataset.Samples {
		plan := buildRetrievalPlan(sample.Question, &c)
		result, err := retrievalEngine.Retrieve(context.Background(), plan)
		if err != nil {
			fmt.Printf("检索失败: id=%s, question='%s', error=%v\n", sample.ID, sample.Question, err)
			continue
		}

		contexts := extractContexts(result)
		sample.SetContexts(contexts...)
		fmt.Printf("检索完成: id=%s, contexts=%d\n", sample.ID, len(contexts))
	}

	// ========== 6. 执行离线评测 ==========
	report, err := runner.Run(context.Background(), dataset)
	if err != nil {
		panic(fmt.Sprintf("评测失败: %v", err))
	}

	// 输出报告
	fmt.Println(report.String())
}

// buildRetrievalPlan 为单条问题构建最小化检索计划。
// 使用"全部知识库"模式，同时启用向量和关键词双通道。
func buildRetrievalPlan(question string, c *config.Config) *vo.RetrievalPlan {
	channelTimeout := c.Chat.Rag.ChannelTimeout
	subQuestionTimeout := c.Chat.Rag.SubQuestionTimeout
	candidateTopK := c.Chat.Rag.CandidateTopK
	finalTopK := 6
	if c.Chat.Rag.FinalTopK > 0 {
		finalTopK = c.Chat.Rag.FinalTopK
	}

	return &vo.RetrievalPlan{
		QuestionPlan: &vo.RetrievalQuestionPlan{
			CurrentQuestion:   question,
			RewrittenQuestion: question,
			RetrievalQuestion: question,
			ExecutionQueries: []*vo.RetrievalExecutionQuery{
				{
					Index:       1,
					SubQuestion: question,
				},
			},
			SubQuestions: []string{question},
		},
		ChatMode:             enum.ChatQueryModeName(enum.ChatQueryModeAutoDocument),
		PrimaryIntent:        "knowledge_retrieval",
		SuggestedIntents:     []string{"knowledge_retrieval"},
		ScopeMode:            enum.KbSelectionModeAll,
		KnowledgeBaseIds:     []int64{},
		AllowedDocumentScope: []int64{},
		DocumentScope:        []int64{0},
		TaskScope:            []int64{0},
		MetadataFilters:      vo.NewMetadataFilters(question, nil),
		Channels: []*vo.RetrievalChannelPlan{
			vo.NewVectorChannelPlan(true, candidateTopK, channelTimeout, 1.0, 0.6),
			vo.NewKeywordChannelPlan(true, candidateTopK, channelTimeout, 1.0, 0.35),
		},
		TableIntent:        &vo.TableIntent{},
		GraphIntent:        &vo.GraphIntent{},
		RaptorIntent:       &vo.RaptorIntent{},
		CandidateTopK:      candidateTopK,
		FinalTopK:          finalTopK,
		RerankEnabled:      true,
		RerankTopK:         candidateTopK,
		SubQuestionTimeout: subQuestionTimeout,
	}
}

// extractContexts 从检索结果中提取所有上下文片段文本。
func extractContexts(result *vo.RetrievalResult) []string {
	if result == nil {
		return nil
	}

	// 使用 map 去重，避免重复的上下文
	seen := make(map[string]struct{})
	var contexts []string
	for _, evidence := range result.SubQuestionEvidenceList {
		for _, doc := range evidence.SourceDocuments {
			if doc == nil || doc.OriginalSnippet == "" {
				continue
			}
			if _, ok := seen[doc.OriginalSnippet]; ok {
				continue
			}
			seen[doc.OriginalSnippet] = struct{}{}
			contexts = append(contexts, doc.OriginalSnippet)
		}
	}
	return contexts
}
