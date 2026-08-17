package vo

import "github.com/swiftbit/know-agent/common/utils"

// Clone 深拷贝表格意图。
func (t *TableIntent) Clone() *TableIntent {
	if t == nil {
		return nil
	}
	s := *t
	s.TableOps = utils.Copy(t.TableOps)
	return &s
}

// TableIntent 表格检索意图
type TableIntent struct {
	Requested bool     `json:"requested"` // 是否请求
	TableOps  []string `json:"tableOps"`  // 表格操作列表
	Source    string   `json:"source"`    // 来源
}
