package chat

import (
	"fmt"
	"strings"
)

func (r *ChatReq) Validate() error {
	if strings.TrimSpace(r.Question) == "" {
		return fmt.Errorf("question 不能为空")
	}
	return nil
}
