package svc

import (
	"context"
	"fmt"
	"strings"
	"time"

	arkemb "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
	einoschema "github.com/cloudwego/eino/schema"
	"github.com/go-playground/validator/v10"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
)

type ServiceContext struct {
	Config     *config.Config
	Validate   *validator.Validate
	Minio      *minio.Client
	Db         *gorm.DB
	Rdb        *redis.Client
	RedSync    *redsync.Redsync
	Emb        embedding.Embedder
	ChatModel  model.BaseModel[*einoschema.AgenticMessage]
	JudgeModel model.BaseModel[*einoschema.AgenticMessage]
	Milvus     *milvusclient.Client
}

func NewServiceContext(c *config.Config) *ServiceContext {
	redisClient := NewRedisClient(c)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return &ServiceContext{
		Config:     c,
		Validate:   validator.New(),
		Rdb:        redisClient,
		Db:         NewMySQLDB(c),
		Minio:      NewMinioClient(c),
		RedSync:    NewRedSync(redisClient),
		Emb:        NewArkEmbedding(ctx, c),
		ChatModel:  NewArkChatModel(ctx, c, "chat"),
		JudgeModel: NewArkChatModel(ctx, c, "judge"),
		Milvus:     NewMilvusClient(ctx, c),
	}
}

// NewMySQLDB 创建 MySQL 数据库连接
func NewMySQLDB(c *config.Config) *gorm.DB {
	m := c.Mysql
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", m.Username, m.Password, m.Endpoint, m.DbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		panic(err)
	}
	return db
}

// NewRedisClient 创建 Redis 客户端
func NewRedisClient(c *config.Config) *redis.Client {
	conf := c.Redis[0]
	client := redis.NewClient(&redis.Options{
		Addr: conf.Host,
		DB:   0,
	})
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		panic(fmt.Sprintf("redis connection error: %v", err))
	}
	return client
}

// NewMinioClient 创建 Minio 客户端
func NewMinioClient(c *config.Config) *minio.Client {
	endpoint := c.Minio.Addr
	accessKeyID := c.Minio.AccessKeyID
	accessKey := c.Minio.SecretAccessKey
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, accessKey, ""),
		Secure: c.Minio.UseSSL,
	})

	if err != nil {
		panic(err)
	}
	return minioClient
}

// NewRedSync 创建 Redis 同步客户端
func NewRedSync(client *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(client)
	return redsync.New(pool)
}

// NewArkEmbedding 创建 ark embedding 模型
func NewArkEmbedding(ctx context.Context, c *config.Config) embedding.Embedder {
	apiType := arkemb.APITypeText
	if strings.Contains(string(arkemb.APITypeMultiModal), c.Embedding.APIType) {
		apiType = arkemb.APITypeMultiModal
	}
	emb, err := arkemb.NewEmbedder(ctx, &arkemb.EmbeddingConfig{
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

func NewArkChatModel(ctx context.Context, c *config.Config, function string) *agenticark.Model {
	llmConf := c.ChatModel["Ark"]
	if function == "judge" {
		llmConf = c.JudgeModel["Ark"]
	}
	chatModel, err := agenticark.New(ctx, &agenticark.Config{
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

func NewMilvusClient(ctx context.Context, c *config.Config) *milvusclient.Client {
	client, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: c.Milvus.Addr,
	})
	if err != nil {
		panic(err)
	}
	return client
}
