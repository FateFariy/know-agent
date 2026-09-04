package consumer

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/svc"
)

// SemanticCacheWriteConsumer 语义缓存异步写入消费者：订阅写入主题，幂等双写（MySQL + Milvus）。
// 向量化在 store.Put 内部完成（消费端持有 Embedder），消息体不携带向量。
type SemanticCacheWriteConsumer struct {
	store conversation.SemanticCacheStore
	c     rocketmq.PushConsumer
	topic string
}

func NewSemanticCacheWriteConsumer(svcCtx *svc.ServiceContext, store conversation.SemanticCacheStore) *SemanticCacheWriteConsumer {
	c, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName("semantic-cache-write-group"),
		consumer.WithNameServer([]string{svcCtx.Config.MQ.Addr}),
	)
	if err != nil {
		panic(err)
	}
	return &SemanticCacheWriteConsumer{
		store: store,
		c:     c,
		topic: svcCtx.Config.Chat.SemanticCache.WriteTopic,
	}
}

func (c *SemanticCacheWriteConsumer) Start() {
	logx.Info("启动语义缓存写入消费者...")
	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		callback := func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			for i := range msgs {
				var entry entity.ChatCacheEntry
				if err := json.Unmarshal(msgs[i].Body, &entry); err != nil {
					logx.Errorf("语义缓存写入消息反序列化失败: %v", err)
					return consumer.ConsumeRetryLater, err
				}
				if err := c.store.Put(ctx, &entry); err != nil {
					logx.Errorf("语义缓存写入消费失败: cacheId=%d, error=%v", entry.ID, err)
					return consumer.ConsumeRetryLater, err
				}
			}
			return consumer.ConsumeSuccess, nil
		}

		if err := c.c.Subscribe(c.topic, consumer.MessageSelector{}, callback); err != nil {
			panic(err)
		}
		if err := c.c.Start(); err != nil {
			panic(err)
		}

		<-sig
		_ = c.c.Shutdown()
	}()
}
