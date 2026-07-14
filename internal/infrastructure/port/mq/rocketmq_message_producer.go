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

func (m *RocketMQMessageProducer) Send(ctx context.Context, topic, key string, message any) error {
	if err := m.p.Start(); err != nil {
		return err
	}
	messageJson, err := json.Marshal(message)
	if err != nil {
		return err
	}
	callback := func(ctx context.Context, result *primitive.SendResult, e error) {
		if e != nil {
			logx.Errorf("receive message error: %s\n", e)
		}
	}
	if err = m.p.SendAsync(ctx, callback, primitive.NewMessage(topic, messageJson)); err != nil {
		return err
	}

	if err = m.p.Shutdown(); err != nil {
		logx.Errorf("shutdown producer error: %s\n", err)
	}
	return nil
}
