package knowledge

import (
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"
)

func (r *KnowledgeScopeSaveReq) Validate() (err error) {
	if strutil.IsBlank(r.ScopeCode) {
		return fmt.Errorf("scope_code 不能为空")
	}
	if strutil.IsBlank(r.ScopeName) {
		return fmt.Errorf("scope_name 不能为空")
	}
	return nil
}

func (r *KnowledgeTopicSaveReq) Validate() (err error) {
	if strutil.IsBlank(r.TopicCode) {
		return fmt.Errorf("topic_code 不能为空")
	}
	if strutil.IsBlank(r.TopicName) {
		return fmt.Errorf("topic_name 不能为空")
	}
	if strutil.IsBlank(r.ScopeCode) {
		return fmt.Errorf("scope_code 不能为空")
	}
	return nil
}

func (r *KnowledgeScopeDeleteReq) Validate() (err error) {
	if strutil.IsBlank(r.ScopeCode) {
		return fmt.Errorf("scope_code 不能为空")
	}
	return nil
}

func (r *KnowledgeTopicDeleteReq) Validate() (err error) {
	if strutil.IsBlank(r.TopicCode) {
		return fmt.Errorf("topic_code 不能为空")
	}
	return nil
}
