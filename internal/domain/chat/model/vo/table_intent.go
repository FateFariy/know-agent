package vo

import (
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

// TableIntent 表格检索意图
type TableIntent struct {
	Requested bool     `json:"requested"` // 是否请求
	TableOps  []string `json:"tableOps"`  // 表格操作列表
	Source    string   `json:"source"`    // 来源
}

// BuildTableIntent 构建表格检索意图
func BuildTableIntent(intentResult *IntentRecognitionResult) *TableIntent {
	return &TableIntent{
		Requested: channelRequested(intentResult, "TABLE_QUERY", "TABLE"),
		TableOps:  normalizeStringsLimit(intentResult.TableOps, 8),
		Source:    intentSource(intentResult),
	}
}

// channelRequested 检查通道是否被请求
func channelRequested(intentResult *IntentRecognitionResult, queryType, intent string) bool {
	if intentResult == nil {
		return false
	}
	if intentResult.QueryType == queryType {
		return true
	}
	for _, ch := range intentResult.Channels {
		if ch == intent {
			return true
		}
	}
	return false
}

// intentSource 获取意图来源
func intentSource(intentResult *IntentRecognitionResult) string {
	if intentResult == nil {
		return "NONE"
	}
	return utils.BlankToDefault(intentResult.Source, "query-understanding")
}

// normalizeStringsLimit 标准化字符串列表并限制数量
func normalizeStringsLimit(items []string, limit int) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	result := make([]string, 0, limit)
	for _, item := range items {
		cleaned := strings.TrimSpace(strings.ToLower(item))
		if cleaned != "" {
			if _, exists := seen[cleaned]; !exists {
				seen[cleaned] = struct{}{}
				result = append(result, cleaned)
				if len(result) >= limit {
					break
				}
			}
		}
	}
	return result
}
