package rerank

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

// ==================== 测试1：同义句组 Rerank ====================
func TestDashScope_Rerank_SynonymGroups(t *testing.T) {
	dashScope := NewDashScope(svcCtx)

	// 以"公司8月份的营业数据"为查询，chunks 中混入同义句和无关句
	query := "公司8月份的营业数据"
	chunks := []*vo.DocumentChunk{
		{Content: "公司8月份的成本支出"},         // 混淆对（相关但不同义）
		{Content: "公司8月份的营业数据情况如何"},     // ✅ 同义句
		{Content: "我想了解一下公司8月份的营收情况。"},  // ✅ 同义句
		{Content: "帮我订一张去北京的机票"},        // 完全无关
		{Content: "公司上个月（8月）的经营业绩怎么样？"}, // ✅ 同义句
	}

	result, err := dashScope.Process(context.Background(), query, chunks)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	t.Log("========== 同义句组 Rerank 结果 ==========")
	for i, chunk := range result {
		t.Logf("Rank #%d | Score: %.4f | Content: %s", i+1, chunk.RerankScore, chunk.Content)
	}

	// 断言：排在第一位的应该是同义句
	if len(result) > 0 {
		topContent := result[0].Content
		if topContent != "公司8月份的营业数据情况如何" &&
			topContent != "我想了解一下公司8月份的营收情况。" &&
			topContent != "公司上个月（8月）的经营业绩怎么样？" {
			t.Errorf("❌ 排在第一位的不是同义句，而是: %s", topContent)
		} else {
			t.Logf("✅ 排在第一位的是同义句: %s", topContent)
		}
	}
}

// ==================== 测试2：易混淆对 Rerank ====================
func TestDashScope_Rerank_ConfusingPairs(t *testing.T) {
	dashScope := NewDashScope(svcCtx)

	// 以"介绍Java的内存管理"为查询，chunks 中混入同义句和易混淆句
	query := "介绍Java的内存管理"
	chunks := []*vo.DocumentChunk{
		{Content: "Java的垃圾回收机制"},           // 混淆对（GC ≠ 内存管理）
		{Content: "Java的内存管理是什么"},          // ✅ 同义句
		{Content: "能给我讲讲Java是怎么进行内存管理的吗？"}, // ✅ 同义句
		{Content: "Python的内存管理是如何工作的？"},    // 混淆（不同语言）
		{Content: "怎么用Java写Web后端？"},        // 混淆（不同领域）
	}

	result, err := dashScope.Process(context.Background(), query, chunks)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	t.Log("========== 易混淆对 Rerank 结果 ==========")
	for i, chunk := range result {
		t.Logf("Rank #%d | Score: %.4f | Content: %s", i+1, chunk.RerankScore, chunk.Content)
	}

	// 断言：同义句应该排在混淆句前面
	if len(result) >= 2 {
		// 找到同义句和混淆句的最高排名
		synonymRank, confusingRank := len(result), len(result)
		for i, chunk := range result {
			switch chunk.Content {
			case "Java的内存管理是什么", "能给我讲讲Java是怎么进行内存管理的吗？":
				if i < synonymRank {
					synonymRank = i
				}
			case "Java的垃圾回收机制", "Python的内存管理是如何工作的？", "怎么用Java写Web后端？":
				if i < confusingRank {
					confusingRank = i
				}
			}
		}
		if synonymRank < confusingRank {
			t.Logf("✅ 同义句（Rank #%d）排在混淆句（Rank #%d）前面", synonymRank+1, confusingRank+1)
		} else {
			t.Errorf("❌ 混淆句（Rank #%d）排在同义句（Rank #%d）前面", confusingRank+1, synonymRank+1)
		}
	}
}

// ==================== 测试3：综合场景（多组混合） ====================
func TestDashScope_Rerank_Comprehensive(t *testing.T) {
	dashScope := NewDashScope(svcCtx)

	// 以"今天天气怎么样？"为查询，混合多组文本
	query := "今天天气怎么样？"
	chunks := []*vo.DocumentChunk{
		// 天气组（同义）
		{Content: "今天天气如何？"},
		{Content: "外面天气好吗？"},
		{Content: "今天的天气预报是什么？"},
		// 混淆组
		{Content: "明天早上8点的会议"},
		{Content: "帮我找附近的川菜馆"},
		{Content: "怎么用Python写爬虫？"},
	}

	result, err := dashScope.Process(context.Background(), query, chunks)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	t.Log("========== 综合场景 Rerank 结果 ==========")
	for i, chunk := range result {
		t.Logf("Rank #%d | Score: %.4f | Content: %s", i+1, chunk.RerankScore, chunk.Content)
	}

	// 断言：前3名应该都是天气相关的同义句
	weatherCount := 0
	for i := 0; i < 3 && i < len(result); i++ {
		content := result[i].Content
		if content == "今天天气如何？" || content == "外面天气好吗？" || content == "今天的天气预报是什么？" {
			weatherCount++
		}
	}
	if weatherCount == 3 {
		t.Logf("✅ 前3名全部是天气相关同义句")
	} else {
		t.Errorf("❌ 前3名中只有 %d 个是天气相关同义句", weatherCount)
	}
}
