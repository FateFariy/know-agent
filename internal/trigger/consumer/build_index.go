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

	"github.com/swiftbit/know-agent/internal/domain/document/logic"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

type BuildIndexConsumer struct {
	l     logic.AsyncProcessingLogic
	c     rocketmq.PushConsumer
	topic string
}

func NewBuildIndexConsumer(svcCtx *svc.ServiceContext, l logic.AsyncProcessingLogic) *BuildIndexConsumer {
	c, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName("build-index-group"),
		consumer.WithNameServer([]string{svcCtx.Config.MQ.Endpoint}),
	)
	if err != nil {
		panic(err)
	}
	return &BuildIndexConsumer{
		l:     l,
		c:     c,
		topic: svcCtx.Config.MQ.IndexTopic,
	}
}

func (b *BuildIndexConsumer) Start() {
	logx.Infof("启动索引构建消费者...")
	go func() {
		sig := make(chan os.Signal)

		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		callback := func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			var msg vo.DocumentIndexBuildMessage
			for i := range msgs {
				_ = json.Unmarshal(msgs[i].Body, &msg)
				if err := b.l.HandleIndexBuild(ctx, msg.DocumentId, msg.TaskId, msg.PlanId); err != nil {
					logx.Errorf("构建索引失败: %s\n", err)
					return consumer.ConsumeRetryLater, err
				}
			}

			// 返回消费成功状态
			return consumer.ConsumeSuccess, nil
		}
		err := b.c.Subscribe(b.topic, consumer.MessageSelector{}, callback)

		if err != nil {
			panic(err)
		}

		// 启动消费者，注意必须在订阅之后调用
		if err = b.c.Start(); err != nil {
			panic(err)
		}

		// 阻塞等待信号，保持程序运行
		<-sig

		// 关闭消费者
		_ = b.c.Shutdown()
	}()
}
