package svc

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	einoschema "github.com/cloudwego/eino/schema"
	"github.com/go-playground/validator/v10"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/config"
)

type ServiceContext struct {
	Config    *config.Config
	Validate  *validator.Validate
	Minio     *minio.Client
	Db        *gorm.DB
	Rdb       *redis.Client
	RedSync   *redsync.Redsync
	ChatModel model.BaseModel[*einoschema.Message]
	Milvus    *milvusclient.Client
	Neo4j     neo4j.DriverWithContext
}

func NewServiceContext(c *config.Config) *ServiceContext {
	redisClient := NewRedisClient(c)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return &ServiceContext{
		Config:    c,
		Validate:  validator.New(),
		Rdb:       redisClient,
		Db:        NewMySQLDB(c),
		Minio:     NewMinioClient(c),
		RedSync:   NewRedSync(redisClient),
		ChatModel: NewArkChatModel(ctx, c.ChatModel["Ark"]),
		Milvus:    NewMilvusClient(ctx, c),
		Neo4j:     NewNeo4jClient(c.Neo4j),
	}
}

// NewMySQLDB 创建 MySQL 数据库连接
func NewMySQLDB(c *config.Config) *gorm.DB {
	m := c.Mysql
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", m.Username, m.Password, m.Endpoint, m.DbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		//Logger:         logger.Default.LogMode(logger.Info),
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

func NewArkChatModel(ctx context.Context, c *config.LLMConf) *ark.ChatModel {
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:      c.ApiKey,
		Model:       c.Model,
		MaxTokens:   utils.Pointer(c.MaxTokens),
		Temperature: utils.Pointer(c.Temperature),
		TopP:        utils.Pointer(c.TopP),
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

// NewNeo4jClient 创建 Neo4j 驱动；未启用或连接失败均返回 nil（不阻断主流程）
func NewNeo4jClient(c config.Neo4jConf) neo4j.DriverWithContext {
	if !c.Enabled {
		return nil
	}
	driver, err := neo4j.NewDriverWithContext(c.Uri, neo4j.BasicAuth(c.Username, c.Password, ""))
	if err != nil {
		logx.Errorf("neo4j 驱动初始化失败: %v", err)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = driver.VerifyConnectivity(ctx); err != nil {
		logx.Errorf("neo4j 连接不可用，已跳过图谱能力: %v", err)
		_ = driver.Close(ctx)
		return nil
	}
	return driver
}
