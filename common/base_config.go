package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/syncx"
	"github.com/zeromicro/go-zero/rest"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type Config interface {
	GetBaseConfig() *BaseConfig
}

type BaseConfig struct {
	Http rest.RestConf
	Auth struct {
		AccessSecret string
		AccessExpire time.Duration
	}
	Mysql MysqlConf
	Redis cache.CacheConf
}

type MysqlConf struct {
	Host     string `json:",default=127.0.0.1"`
	Port     int    `json:",default=3306"`
	Username string `json:",omitempty"`
	Password string `json:",omitempty"`
	DbName   string `json:",omitempty"`
}

var ProviderSet = wire.NewSet(NewDb, NewRedisCache, NewRedisClient, NewValidator)

func NewDb(c Config) *gorm.DB {
	m := c.GetBaseConfig().Mysql
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", m.Username, m.Password, m.Host, m.Port, m.DbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		panic(err)
	}
	return db
}

func NewRedisCache(c Config) cache.Cache {
	singleFlights := syncx.NewSingleFlight()
	stats := cache.NewStat("redis")
	err := errors.New("redis 缓存未命中")
	cacheConf := c.GetBaseConfig().Redis
	return cache.New(cacheConf, singleFlights, stats, err)
}

func NewRedisClient(c Config) *redis.Client {
	conf := c.GetBaseConfig().Redis[0]
	client := redis.NewClient(&redis.Options{
		Addr: conf.Host,
		DB:   0,
	})
	if _, err := client.Ping(context.Background()).Result(); err != nil {
		panic(fmt.Sprintf("redis connection error: %v", err))
	}
	return client
}

func NewValidator() *validator.Validate {
	return validator.New()
}
