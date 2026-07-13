package persistence

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/conf"

	"github.com/swiftbit/know-agent/internal/config"
	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

var svcCtx *svc.ServiceContext
var chatModel *chatlogic.ChatModelImpl[*schema.AgenticMessage]

func init() {
	configFile := "E:/gocode/ragent-convert/know-agent/etc/config-dev.yaml"
	var c config.Config
	conf.MustLoad(configFile, &c)
	svcCtx = svc.NewServiceContext(&c)
	chatModel = chatlogic.NewChatModelImpl(svcCtx)
}

func TestRepository(t *testing.T) {
	trace := vo.NewConversationTrace("1", 1, "1")
	withTrace, err := chatModel.StreamWithTrace(context.Background(), "system", "", "你是谁？", trace)
	if err != nil {
		return
	}
	for {
		select {
		case text, ok := <-withTrace:
			if !ok {
				fmt.Printf("%+v", trace.SnapshotModelUsageTraces()[0])
				return
			}
			fmt.Println(text)
		}
	}

}
