package mq

import (
	"context"
	"encoding/json"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/svc"
)

type RocketMQMessageProducer struct {
	p rocketmq.Producer
}

var _ adapter.MessageProducer = (*RocketMQMessageProducer)(nil)

func NewRocketMQMessageProducer(svcCtx *svc.ServiceContext) *RocketMQMessageProducer {
	p, err := rocketmq.NewProducer(
		producer.WithNameServer([]string{svcCtx.Config.MQ.Endpoint}),
		producer.WithRetry(svcCtx.Config.MQ.Retry))
	if err != nil {
		panic(err)
	}
	return &RocketMQMessageProducer{
		p: p,
	}
}

// Start 启动生产者
func (m *RocketMQMessageProducer) Start() {
	if err := m.p.Start(); err != nil {
		panic(err)
	}
}

// Send 发送消息
func (m *RocketMQMessageProducer) Send(ctx context.Context, topic, key string, message any) error {
	messageJson, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = m.p.SendSync(ctx, primitive.NewMessage(topic, messageJson))
	return err
}

// Close 关闭生产者
func (m *RocketMQMessageProducer) Close() {
	if err := m.p.Shutdown(); err != nil {
		logx.Errorf("rocketmq producer shutdown failed: %v", err)
	}
}
