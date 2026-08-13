package aggregate

import (
	"encoding/json"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

type KnowledgeBaseSelectionSnapshot struct {
	SelectionMode              enum.KnowledgeBaseSelectionMode `json:"selectionMode"`
	SelectedKnowledgeBaseIds   []int64                         `json:"selectedKnowledgeBaseIds"`
	SelectedKnowledgeBaseNames []string                        `json:"selectedKnowledgeBaseNames"`
	SelectedKnowledgeBases     []*entity.KnowledgeBase         `json:"selectedKnowledgeBases"`
	AllowedDocuments           []*vo.DocumentMetadata          `json:"allowedDocuments"`
	RagRuntimeOptions          *vo.RagRuntimeOptions           `json:"ragRuntimeOptions,omitempty"`
}

// SelectionModeName 返回选择模式的名称，若快照或模式为空则返回 enum.KbSelectionModeNone
func (s *KnowledgeBaseSelectionSnapshot) SelectionModeName() string {
	if s == nil || s.SelectionMode == "" {
		return enum.KbSelectionModeNone
	}
	return s.SelectionMode
}

func (s *KnowledgeBaseSelectionSnapshot) SelectionIDs() string {
	if s == nil || len(s.SelectedKnowledgeBaseIds) == 0 {
		return ""
	}
	ids, _ := json.Marshal(s.SelectedKnowledgeBaseIds)
	return string(ids)
}

func (s *KnowledgeBaseSelectionSnapshot) SelectionNames() string {
	if s == nil || len(s.SelectedKnowledgeBaseNames) == 0 {
		return ""
	}
	names := make([]string, 0, len(s.SelectedKnowledgeBaseNames))
	for _, name := range s.SelectedKnowledgeBaseNames {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}
	marshal, _ := json.Marshal(names)
	return string(marshal)
}
