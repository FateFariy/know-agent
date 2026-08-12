package chat

import (
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
)

const (
	OpenChat = "open_chat"
	AutoDoc  = "auto_document"
	Document = "document"
)

func (r *ChatReq) Validate() (err error) {
	r.ConversationId = strutil.Trim(r.ConversationId)
	r.ConversationId = utils.BlankToDefault(r.ConversationId, utils.GenerateUUIDWithoutHyphen())
	r.Question = strutil.Trim(r.Question)
	defer func() {
		if err != nil {
			logx.Errorf("会话启动失败, conversationId=%s, question=%s, err=%s", r.ConversationId, r.Question, err.Error())
		}
	}()
	if strutil.IsBlank(r.Question) {
		return fmt.Errorf("question 不能为空")
	}
	switch r.ChatMode {
	case OpenChat:
		if r.KnowledgeBaseSelectionMode != "none" {
			return fmt.Errorf("open_chat 模式 KnowledgeBaseSelectionMode 必须为 'none'")
		}
		if r.SelectedDocumentId != "" {
			return fmt.Errorf("open_chat 模式 SelectedDocumentId 必须为空")
		}
	case AutoDoc:
		if r.KnowledgeBaseSelectionMode != "all" {
			return fmt.Errorf("auto_document 模式 KnowledgeBaseSelectionMode 必须为 'all'")
		}
		if r.SelectedDocumentId != "" {
			return fmt.Errorf("auto_document 模式 SelectedDocumentId 必须为空")
		}
	case Document:
		if r.KnowledgeBaseSelectionMode == "none" {
			return fmt.Errorf("document 模式必须选择知识库或使用全部知识库")
		}
		if r.SelectedDocumentId == "" {
			return fmt.Errorf("document 模式必须传 SelectedDocumentId")
		}
	default:
		return fmt.Errorf("未知的 ChatMode: %s", r.ChatMode)
	}
	return nil
}
