package chat

import (
	"strings"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
)

const (
	OpenChat = "open_chat"
	AutoDoc  = "auto_document"
	Document = "document"
)

func (r *ChatReq) Validate() error {
	r.ConversationId = strings.TrimSpace(r.ConversationId)
	r.ConversationId = utils.Ternary(r.ConversationId != "", r.ConversationId, utils.GenerateUUIDWithoutHyphen())
	if strings.TrimSpace(r.Question) == "" {
		return common.ErrParm.Format("question 不能为空")
	}
	if r.ChatMode == OpenChat && r.SelectedDocumentId != 0 {
		return common.ErrParm.Format("open_chat 模式 selectedDocumentId 必须为空")
	}
	if r.ChatMode == AutoDoc && r.SelectedDocumentId != 0 {
		return common.ErrParm.Format("auto_document 模式 selectedDocumentId 必须为空")
	}
	if r.SelectedDocumentId == 0 {
		return common.ErrParm.Format("document 模式必须传 selectedDocumentId")
	}
	return nil
}
