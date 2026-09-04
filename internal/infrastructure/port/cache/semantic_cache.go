package cache

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	milvuscol "github.com/milvus-io/milvus/client/v2/column"
	milvusentity "github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"gorm.io/gorm/clause"

	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// 向量集合字段定义（仅轻量索引，完整真值存 MySQL）。集合结构见 etc/semantic_cache_collection.json，
const (
	vectorField     = "vector"
	cacheIDField    = "cache_id"
	queryTextField  = "query_text"
	chatModeField   = "chat_mode"
	docIDsField     = "scope_document_ids"
	taskIDsField    = "scope_task_ids"
	kbIDsField      = "scope_knowledge_base_ids"
	searchTopK      = 3
	defaultCollName = "chat_cache"
)

// SemanticCache 语义缓存存储实现（基础设施层）。
//
// 存储分工：
//   - Milvus(chat_cache)：仅存一条轻量向量索引，含 cache_id(主键,int64) + vector + query_text + chat_mode +
//     三组 scope id 数组；向量化在 Put/Search 内部完成（持有 svcCtx.Emb）。
//   - MySQL(chat_cache_entry)：存完整语义缓存真值（answer_draft + execution_plan 等 JSON），是主数据源。
//
// 读写链路：
//   - Put：先写 MySQL 完整条目（主键回填），再向量化 QueryText 后写一条 Milvus 向量记录（先删后插，幂等）。
//   - Search：向量化 QueryText → Milvus ANN 检索（scope 子集匹配过滤）→ 候选 cache_id → 回查 MySQL。
//   - Invalidate：按 scope 软删除 MySQL + 删除 Milvus 对应向量记录。
//
// 集合由外部 etc/semantic_cache_collection.json 建表；缺失时仅告警并降级为未命中，不在此处建表。
type SemanticCache struct {
	milvusClient *milvusclient.Client
	emb          embedding.Embedder
	db           *gorm.DB
	collection   string
	dim          int
}

var _ conversation.SemanticCacheStore = (*SemanticCache)(nil)

func NewSemanticCache(svcCtx *svc.ServiceContext, emb embedding.Embedder) *SemanticCache {
	collection := svcCtx.Config.Chat.SemanticCache.Collection
	if collection == "" {
		collection = defaultCollName
	}
	sc := &SemanticCache{
		milvusClient: svcCtx.Milvus,
		emb:          emb,
		db:           svcCtx.Db,
		collection:   collection,
		dim:          svcCtx.Config.Embedding.Dimensions,
	}
	return sc
}

// embed 文本向量化
func (s *SemanticCache) embed(ctx context.Context, text string) (milvusentity.FloatVector, error) {
	vecs, err := s.emb.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil, errors.New("embedding 结果为空")
	}
	return utils.Map(vecs[0], func(t float64) float32 {
		return float32(t)
	}), nil
}

// Search 在 scope 内对查询文本做向量候选召回（向量化在内部完成），
// 返回相似度 ≥ Threshold 且按相似度降序的完整候选；无候选返回 (nil, nil)
func (s *SemanticCache) Search(ctx context.Context, input *conversation.SearchInput) ([]*conversation.CacheHit, error) {
	topK := max(input.TopK, searchTopK)
	entries, err := s.annSearch(ctx, input, topK)
	if err != nil {
		return nil, err
	}
	hits := make([]*conversation.CacheHit, 0, len(entries))
	for _, cand := range entries {
		entry, err := s.loadEntry(ctx, cand.ID)
		if err != nil {
			return nil, err
		}
		hits = append(hits, &conversation.CacheHit{Entry: entry, Similarity: cand.Similarity})
	}
	return hits, nil
}

// annSearch 执行 Milvus ANN 召回：按阈值过滤并按相似度降序，仅返回主键与相似度（真值回查见 SearchCandidates）
func (s *SemanticCache) annSearch(ctx context.Context, input *conversation.SearchInput, topK int) ([]*entity.ChatCacheEntry, error) {
	vec, err := s.embed(ctx, input.QueryText)
	if err != nil {
		return nil, fmt.Errorf("语义缓存: 向量化失败: %w", err)
	}
	annParam := index.NewCustomAnnParam()
	annParam.WithRadius(float64(input.Threshold))
	annParam.WithRangeFilter(1.0)

	opt := milvusclient.NewSearchOption(s.collection, topK, []milvusentity.Vector{vec}).
		WithFilter(s.buildFilter(input.Scope)).
		WithOutputFields(cacheIDField).
		WithConsistencyLevel(milvusentity.ClBounded).WithAnnParam(annParam)

	results, err := s.milvusClient.Search(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("语义缓存: 检索失败: %w", err)
	}

	entries := make([]*entity.ChatCacheEntry, 0, len(results))
	for _, rs := range results {
		col := rs.GetColumn(cacheIDField)
		if col == nil {
			continue
		}
		cv, ok := col.(*milvuscol.ColumnInt64)
		if !ok {
			continue
		}
		ids := cv.Data()
		for i := 0; i < len(ids) && i < len(rs.Scores); i++ {
			entries = append(entries, &entity.ChatCacheEntry{ID: ids[i], Similarity: rs.Scores[i]})
		}
	}
	slices.SortFunc(entries, func(a, b *entity.ChatCacheEntry) int { return cmp.Compare(b.Similarity, a.Similarity) })
	return entries, nil
}

// loadEntry 按主键从 MySQL 回查完整缓存真值
func (s *SemanticCache) loadEntry(ctx context.Context, id int64) (*entity.ChatCacheEntry, error) {
	entry := &entity.ChatCacheEntry{}
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(entry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("语义缓存: 读取 MySQL 失败: %w", err)
	}
	return entry, nil
}

// Put 写入/更新一条缓存：写 MySQL 真值（主键回填） + 向量化后写一条 Milvus 向量记录（先删后插，幂等）
func (s *SemanticCache) Put(ctx context.Context, entry *entity.ChatCacheEntry) error {
	if entry.ID == 0 {
		entry.ID = utils.GetSnowflakeNextID()
	}
	assignments := clause.AssignmentColumns([]string{"answer_draft"})
	assignments = append(assignments, clause.Assignment{Column: clause.Column{Name: "update_time"}, Value: time.Now()})
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: assignments,
	}).Create(convert.ToChatCacheEntryModel(entry)).Error; err != nil {
		return fmt.Errorf("语义缓存: 写入 MySQL 失败: %w", err)
	}

	vec, err := s.embed(ctx, entry.QueryText)
	if err != nil {
		return fmt.Errorf("语义缓存: 向量化失败: %w", err)
	}
	if err = s.deleteVector(ctx, entry.ID); err != nil {
		logx.Warnf("语义缓存: 删除旧向量失败(忽略) id=%d, error=%v", entry.ID, err)
	}
	if err = s.insertVector(ctx, entry, vec); err != nil {
		return fmt.Errorf("语义缓存: 写入向量失败: %w", err)
	}
	return nil
}

// insertVector 写入一条 Milvus 向量记录
func (s *SemanticCache) insertVector(ctx context.Context, entry *entity.ChatCacheEntry, vec milvusentity.FloatVector) error {
	docIDs, _ := json.Marshal(entry.AllowedDocumentIds)
	taskIDs, _ := json.Marshal(entry.AllowedTaskIds)
	kbIDs, _ := json.Marshal(entry.KnowledgeBaseIds)
	_, err := s.milvusClient.Insert(ctx, milvusclient.NewColumnBasedInsertOption(s.collection,
		milvuscol.NewColumnInt64(cacheIDField, []int64{entry.ID}),
		milvuscol.NewColumnFloatVector(vectorField, s.dim, [][]float32{vec}),
		milvuscol.NewColumnVarChar(queryTextField, []string{entry.QueryText}),
		milvuscol.NewColumnVarChar(chatModeField, []string{strconv.Itoa(entry.ChatMode)}),
		milvuscol.NewColumnJSONBytes(docIDsField, [][]byte{docIDs}),
		milvuscol.NewColumnJSONBytes(taskIDsField, [][]byte{taskIDs}),
		milvuscol.NewColumnJSONBytes(kbIDsField, [][]byte{kbIDs}),
	))
	if err != nil {
		return err
	}
	_, err = s.milvusClient.Flush(ctx, milvusclient.NewFlushOption(s.collection))
	return err
}

// deleteVector 按 cache_id 删除 Milvus 向量记录（Put 幂等 / Invalidate 复用）
func (s *SemanticCache) deleteVector(ctx context.Context, cacheID int64) error {
	expr := fmt.Sprintf("%s == %d", cacheIDField, cacheID)
	_, err := s.milvusClient.Delete(ctx, milvusclient.NewDeleteOption(s.collection).WithExpr(expr))
	return err
}

// buildFilter 构造 Milvus 标量过滤表达式（scope 子集匹配）
// 语义：查询的 allowed id 集合 ⊆ 缓存的 allowed id 集合（json_contains 逐个校验）
func (s *SemanticCache) buildFilter(scope *vo.CacheScope) string {
	if scope == nil {
		return ""
	}
	var parts []string
	if scope.ChatMode != 0 {
		parts = append(parts, fmt.Sprintf("%s == \"%d\"", chatModeField, scope.ChatMode))
	}
	for _, id := range scope.AllowedDocumentIds {
		parts = append(parts, fmt.Sprintf("json_contains(%s, %d)", docIDsField, id))
	}
	for _, id := range scope.AllowedTaskIds {
		parts = append(parts, fmt.Sprintf("json_contains(%s, %d)", taskIDsField, id))
	}
	for _, id := range scope.KnowledgeBaseIds {
		parts = append(parts, fmt.Sprintf("json_contains(%s, %d)", kbIDsField, id))
	}
	return strings.Join(parts, " and ")
}
