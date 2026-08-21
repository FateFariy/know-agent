package vo

import (
	"encoding/json"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

type KnowledgeBaseSelectionSnapshot struct {
	SelectionMode              enum.KnowledgeBaseSelectionMode `json:"selectionMode"`
	SelectedKnowledgeBaseIds   []int64                         `json:"selectedKnowledgeBaseIds"`
	SelectedKnowledgeBaseNames []string                        `json:"selectedKnowledgeBaseNames"`
	SelectedKnowledgeBases     []*KnowledgeBase                `json:"selectedKnowledgeBases"`
	AllowedDocuments           []*DocumentMetadata             `json:"allowedDocuments"`
	RagRuntimeOptions          *RagRuntimeOptions              `json:"ragRuntimeOptions,omitempty"`
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

// SelectionNames 返回所有选中的知识库名称
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

// SelectedDocumentIds 返回所有允许文档的ID列表
func (s *KnowledgeBaseSelectionSnapshot) SelectedDocumentIds() []int64 {
	if s == nil || len(s.AllowedDocuments) == 0 {
		return nil
	}
	return utils.Map(s.AllowedDocuments, func(doc *DocumentMetadata) int64 {
		return doc.DocumentId
	})
}

// SelectedTaskIds 返回所有允许文档的任务ID列表
func (s *KnowledgeBaseSelectionSnapshot) SelectedTaskIds() []int64 {
	if s == nil || len(s.AllowedDocuments) == 0 {
		return nil
	}
	return utils.Map(s.AllowedDocuments, func(doc *DocumentMetadata) int64 {
		return doc.LastIndexTaskId

	})
}

func (s *KnowledgeBaseSelectionSnapshot) RagRuntimeConfigSnapshot() string {
	if s == nil || s.RagRuntimeOptions == nil {
		return ""
	}
	snapshot, _ := json.Marshal(s.RagRuntimeOptions)
	return string(snapshot)
}

// ResolveAllowedExecutionScope 从知识库选择快照解析允许的执行范围
//
// 检查逻辑：
//  1. 遍历所有允许文档，校验文档描述符的一致性（documentId 与 taskId 必须为正、无冲突）
//  2. 将解析出的文档ID与任务ID列表，与声明列表比对
//  3. 若存在不一致或冲突，返回 inconsistent 范围
//  4. 若解析结果为空，返回 empty 范围
//  5. 一切正常时返回 ready 范围
func (s *KnowledgeBaseSelectionSnapshot) ResolveAllowedExecutionScope() *AllowedExecutionScope {
	if s == nil {
		return newInconsistentScope("知识库选择快照为空，无法解析执行范围")
	}

	// 遍历文档描述符，构建文档ID→任务ID映射，同时检测冲突
	tasksByDocument := make(map[int64]int64, len(s.AllowedDocuments))
	documentIds := make([]int64, 0, len(s.AllowedDocuments))
	taskIds := make([]int64, 0, len(s.AllowedDocuments))
	descriptorConflict := false
	for _, desc := range s.AllowedDocuments {
		if desc == nil || desc.DocumentId <= 0 || desc.LastIndexTaskId <= 0 {
			descriptorConflict = true
			continue
		}
		if prevTaskID, exists := tasksByDocument[desc.DocumentId]; exists && prevTaskID != desc.LastIndexTaskId {
			descriptorConflict = true
		} else {
			tasksByDocument[desc.DocumentId] = desc.LastIndexTaskId
			documentIds = append(documentIds, desc.DocumentId)
			taskIds = append(taskIds, desc.LastIndexTaskId)
		}
	}

	// 检查一致性：无冲突且解析结果与声明列表匹配
	if descriptorConflict {
		return newInconsistentScope("知识选择快照中文档/任务标识不一致")
	}

	if len(documentIds) == 0 {
		return newEmptyScope("所选知识范围没有已就绪的文档/任务对")
	}

	return &AllowedExecutionScope{
		documentIds: documentIds,
		taskIds:     taskIds,
		Consistent:  true,
	}
}

// AllowedExecutionScope 允许执行的知识范围
// 通过知识库选择快照解析得到，用于约束文档路由和检索范围
type AllowedExecutionScope struct {
	documentIds []int64 //  文档ID列表
	taskIds     []int64 //  任务ID列表
	Consistent  bool    //  是否一致
	Reason      string  //  原因
}

// newEmptyScope 创建空的允许范围（已就绪但无数据）
func newEmptyScope(reason string) *AllowedExecutionScope {
	return &AllowedExecutionScope{
		Consistent: true,
		Reason:     reason,
	}
}

// newInconsistentScope 创建不一致的允许范围
func newInconsistentScope(reason string) *AllowedExecutionScope {
	return &AllowedExecutionScope{
		Consistent: false,
		Reason:     reason,
	}
}

// Executable 判断范围是否可执行：一致且非空且文档与任务ID数量一致
func (s *AllowedExecutionScope) Executable() bool {
	return s != nil || s.Consistent && len(s.documentIds) > 0 && len(s.documentIds) == len(s.taskIds)
}

// Contains 判断指定的文档ID与任务ID是否在当前范围内
func (s *AllowedExecutionScope) Contains(documentId, taskId int64) bool {
	if s == nil || documentId <= 0 || taskId <= 0 {
		return false
	}
	for i, docID := range s.documentIds {
		if docID == documentId && i < len(s.taskIds) && s.taskIds[i] == taskId {
			return true
		}
	}
	return false
}

// DocumentIds 返回文档ID列表副本
func (s *AllowedExecutionScope) DocumentIds() []int64 {
	if s == nil {
		return nil
	}
	return utils.Copy(s.documentIds)
}

// TaskIds 返回任务ID列表副本
func (s *AllowedExecutionScope) TaskIds() []int64 {
	if s == nil {
		return nil
	}
	return utils.Copy(s.taskIds)
}

func (s *AllowedExecutionScope) FilterCandidates(candidates []*DocumentRouteCandidate) []*DocumentRouteCandidate {
	if s == nil || !s.Executable() || len(candidates) == 0 {
		return nil
	}

	filtered := make([]*DocumentRouteCandidate, 0, len(candidates))
	for _, c := range candidates {
		if c != nil && s.Contains(c.DocumentId, c.LastIndexTaskId) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
