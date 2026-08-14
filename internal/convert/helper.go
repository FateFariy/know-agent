package convert

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/enum"
)

func TimeToString(t time.Time) string {
	return t.Format(time.DateTime)
}

func TimeToStringMs(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

func ToChatQueryModeName(code int) string {
	return enum.ChatQueryModeName(code)
}

func ToChatDebugTrace(debugTraceJson string) *chat.ChatDebugTrace {
	var debugTrace cvo.ChatDebugTrace
	if err := json.Unmarshal([]byte(debugTraceJson), &debugTrace); err != nil {
		return nil
	}
	return &chat.ChatDebugTrace{
		ExecutionMode:                 debugTrace.ExecutionMode,
		ChatMode:                      enum.ChatQueryModeName(debugTrace.ChatMode),
		OriginalQuestion:              debugTrace.OriginalQuestion,
		RewriteQuestion:               debugTrace.RewriteQuestion,
		RewriteSubQuestions:           debugTrace.RewriteSubQuestions,
		RetrievalQuestion:             debugTrace.RetrievalQuestion,
		AgentQuestion:                 debugTrace.AgentQuestion,
		NavigationDecision:            ToChatDocumentNavigationDecision(debugTrace.NavigationDecision),
		HistorySummary:                debugTrace.HistorySummary,
		LongTermSummary:               debugTrace.LongTermSummary,
		RecentHistoryTranscript:       debugTrace.RecentHistoryTranscript,
		AnswerRecentTranscript:        debugTrace.RecentQuestionTranscript,
		AnswerHistoryContext:          debugTrace.QuestionHistoryContext,
		AnswerHistoryFollowUpQuestion: debugTrace.QuestionHistoryFollowUpQuestion,
		HistoryCompressionApplied:     debugTrace.HistoryCompressionApplied,
		HistoryCoveredExchangeId:      debugTrace.HistoryCoveredExchangeId,
		HistoryCoveredExchangeCount:   debugTrace.HistoryCoveredExchangeCount,
		HistoryCompressionCount:       debugTrace.HistoryCompressionCount,
		CurrentDateText:               debugTrace.CurrentDateText,
		RequiresFreshSearch:           debugTrace.RequiresRealTimeSearch,
		RequiresCurrentDateAnchoring:  debugTrace.RequiresCurrentDateAnchoring,
		SubQuestions:                  debugTrace.SubQuestions,
		SelectedDocumentId:            debugTrace.SelectedDocumentId,
		SelectedTaskId:                debugTrace.SelectedTaskId,
		RetrievalNotes:                debugTrace.RetrievalNotes,
		UsedChannels:                  debugTrace.UsedChannels,
		ToolTraces:                    ToChatToolTraces(debugTrace.ToolTraces),
		ModelUsageTraces:              ToChatModelUsageTraces(debugTrace.ModelUsageTraces),
		LimitStats:                    ToChatLimitStats(debugTrace.LimitStats),
		RagSystemPrompt:               debugTrace.RagSystemPrompt,
		RagUserPrompt:                 debugTrace.RagUserPrompt,
		NoEvidenceReply:               debugTrace.NoEvidenceReply,
	}
}

func ToChatDocumentNavigationDecision(src *cvo.DocumentNavigationDecision) *chat.DocumentNavigationDecision {
	if src == nil {
		return nil
	}

	mode := src.ExecutionModeName
	if mode == "" && src.ExecutionMode != nil {
		mode = src.ExecutionMode.Name()
	}
	return &chat.DocumentNavigationDecision{
		NavigationAction:  src.NavigationAction,
		ExecutionMode:     mode,
		StructureAnchor:   ToChatStructureAnchor(src.StructureAnchor),
		ItemAnchor:        ToChatItemAnchor(src.ItemAnchor),
		RetrievalPlan:     ToChatRetrievalQuestionPlan(src.RetrievalPlan),
		SummaryText:       src.SummaryText,
		QueryContextHints: append([]string(nil), src.QueryContextHints...),
		SoftSectionHints:  append([]string(nil), src.SoftSectionHints...),
	}
}

func ToChatStructureAnchor(src *cvo.ConversationStructureAnchor) *chat.ConversationStructureAnchor {
	if src == nil {
		return nil
	}
	return &chat.ConversationStructureAnchor{
		RootSectionCode:   src.RootSectionCode,
		RootSectionTitle:  src.RootSectionTitle,
		TargetSectionHint: src.TargetSectionHint,
		StructureNodeId:   src.StructureNodeId,
		CanonicalPath:     src.CanonicalPath,
		ScopeMode:         src.ScopeMode,
	}
}

func ToChatItemAnchor(src *cvo.ConversationItemAnchor) *chat.ConversationItemAnchor {
	if src == nil {
		return nil
	}
	return &chat.ConversationItemAnchor{
		ItemIndex:       src.ItemIndex,
		ItemText:        src.ItemText,
		StructureNodeId: src.StructureNodeId,
		CanonicalPath:   src.CanonicalPath,
	}
}

func ToChatRetrievalQuestionPlan(src *cvo.RetrievalQuestionPlan) *chat.RetrievalQuestionPlan {
	if src == nil {
		return nil
	}
	return &chat.RetrievalQuestionPlan{
		RetrievalQuestion: src.RetrievalQuestion,
		SubQuestions:      append([]string(nil), src.SubQuestions...),
	}
}

func ToChatToolTraces(src []*cvo.ChatToolTrace) []*chat.ChatToolTrace {
	if src == nil {
		return nil
	}
	result := make([]*chat.ChatToolTrace, len(src))
	for i, t := range src {
		if t == nil {
			result[i] = nil
			continue
		}
		result[i] = &chat.ChatToolTrace{
			ToolName:       t.ToolName,
			Status:         t.Status,
			InputSummary:   t.InputSummary,
			EffectiveInput: t.EffectiveInput,
			OutputSummary:  t.OutputSummary,
			ErrorMessage:   t.ErrorMessage,
			ReferenceCount: t.ReferenceCount,
			Topic:          t.Topic,
			DurationMs:     t.DurationMs,
		}
	}
	return result
}

func ToChatModelUsageTraces(src []*cvo.ChatModelUsageTrace) []*chat.ChatModelUsageTrace {
	if src == nil {
		return nil
	}
	result := make([]*chat.ChatModelUsageTrace, len(src))
	for i, t := range src {
		if t == nil {
			result[i] = nil
			continue
		}
		result[i] = &chat.ChatModelUsageTrace{
			StageName:        t.StageName,
			Provider:         t.Provider,
			Model:            t.Model,
			PromptTokens:     t.PromptTokens,
			CompletionTokens: t.CompletionTokens,
			TotalTokens:      t.TotalTokens,
			EstimatedCost:    t.EstimatedCost,
			DurationMs:       t.DurationMs,
			Status:           t.Status,
		}
	}
	return result
}

func ToChatLimitStats(src *cvo.ChatLimitStats) *chat.ChatLimitStats {
	if src == nil {
		return nil
	}
	return &chat.ChatLimitStats{
		ModelCallsUsed:        src.ModelCallsUsed,
		ModelCallsRunLimit:    src.ModelCallsRunLimit,
		ModelCallsThreadLimit: src.ModelCallsThreadLimit,
		ToolCallsUsed:         src.ToolCallsUsed,
		ToolCallsRunLimit:     src.ToolCallsRunLimit,
		ToolCallsThreadLimit:  src.ToolCallsThreadLimit,
		LimitTriggered:        src.LimitTriggered,
		LimitReason:           src.LimitReason,
	}
}

func JsonArrayToStringSlice(src common.JSONArray) []string {
	return common.JSONArrayTo(src, func(item any) string {
		return item.(string)
	})
}

func StringToStringSlice(src string) []string {
	return strings.Split(src, ",")
}

func JsonArrayToSearchReferences(src common.JSONArray) []*chat.SearchReference {
	return common.JSONArrayTo(src, func(item any) *chat.SearchReference {
		refMap := item.(map[string]any)
		return &chat.SearchReference{
			ReferenceId:        refMap["referenceId"].(string),
			SourceType:         refMap["sourceType"].(string),
			Title:              refMap["title"].(string),
			Url:                refMap["url"].(string),
			Snippet:            refMap["snippet"].(string),
			DocumentId:         convertor.ToString(refMap["documentId"]),
			DocumentName:       refMap["documentName"].(string),
			ChunkId:            convertor.ToString(refMap["chunkId"]),
			ParentBlockId:      convertor.ToString(refMap["parentBlockId"]),
			ParentBlockNo:      int(refMap["parentBlockNo"].(float64)),
			ChunkNo:            int(refMap["chunkNo"].(float64)),
			SectionPath:        refMap["sectionPath"].(string),
			StructureNodeId:    convertor.ToString(refMap["structureNodeId"]),
			StructureNodeType:  int(refMap["structureNodeType"].(float64)),
			CanonicalPath:      refMap["canonicalPath"].(string),
			ItemIndex:          int(refMap["itemIndex"].(float64)),
			Score:              refMap["score"].(float64),
			SubQuestionIndex:   int(refMap["subQuestionIndex"].(float64)),
			SubQuestion:        refMap["subQuestion"].(string),
			Channel:            refMap["channel"].(string),
			ToolName:           refMap["toolName"].(string),
			KnowledgeScopeCode: refMap["knowledgeScopeCode"].(string),
			KnowledgeScopeName: refMap["knowledgeScopeName"].(string),
		}
	})
}

func ToRouteStatus(code int) string {
	return klvo.RouteStatusName(code)
}

func NormalizeString(s string) string {
	return strutil.Trim(s)
}

func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

func StringToInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}
