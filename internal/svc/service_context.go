package svc

import (
	"context"
	"strings"

	arkemb "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/go-playground/validator/v10"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/google/wire"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
)

var ProviderSet = wire.NewSet(NewServiceContext)

type ServiceContext struct {
	Config    *config.Config
	Validate  *validator.Validate
	Minio     *minio.Client
	Db        *gorm.DB
	Rdb       *redis.Client
	RedSync   *redsync.Redsync
	Emb       embedding.Embedder
	ChatModel model.BaseModel[*schema.AgenticMessage]
}

func NewServiceContext(c *config.Config) *ServiceContext {
	redisClient := common.NewRedisClient(c)
	return &ServiceContext{
		Config:    c,
		Validate:  common.NewValidator(),
		Rdb:       redisClient,
		Db:        common.NewDb(c),
		Minio:     NewMinioClient(c),
		RedSync:   NewRedSync(redisClient),
		Emb:       NewArkEmbedding(c),
		ChatModel: NewArkChatModel(c),
	}
}

// NewMinioClient 创建 Minio 客户端
func NewMinioClient(c *config.Config) *minio.Client {
	endpoint := c.Minio.Endpoint
	accessKeyID := c.Minio.AccessKeyID
	accessKey := c.Minio.SecretAccessKey
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, accessKey, ""),
		Secure: c.Minio.UseSSL,
	})
	if err != nil {
		return nil
	}
	return minioClient
}

// NewRedSync 创建 Redis 同步客户端
func NewRedSync(client *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(client)
	return redsync.New(pool)
}

// NewArkEmbedding 创建 ark embedding 模型
func NewArkEmbedding(c *config.Config) embedding.Embedder {
	apiType := arkemb.APITypeText
	if strings.Contains(c.Embedding.APIType, string(arkemb.APITypeMultiModal)) {
		apiType = arkemb.APITypeMultiModal
	}
	emb, err := arkemb.NewEmbedder(context.TODO(), &arkemb.EmbeddingConfig{
		APIKey:     c.Embedding.APIKey,
		Model:      c.Embedding.Model,
		APIType:    utils.Pointer(apiType),
		Dimensions: &c.Embedding.Dimensions,
	})
	if err != nil {
		panic(err)
	}
	return emb
}

func NewArkChatModel(c *config.Config) *agenticark.Model {
	llmConf := c.ChatModel["ark"]
	chatModel, err := agenticark.New(context.TODO(), &agenticark.Config{
		APIKey:      llmConf.ApiKey,
		Model:       llmConf.Model,
		MaxTokens:   utils.Pointer(llmConf.MaxTokens),
		Temperature: utils.Pointer(llmConf.Temperature),
		TopP:        utils.Pointer(llmConf.TopP),
	})
	if err != nil {
		panic(err)
	}
	return chatModel
}
