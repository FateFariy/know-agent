// 对话类型
/** 聊天请求 */
export interface ChatReq {
  question: string;
  conversationId?: string;
  /** 聊天模式 */
  chatMode: 'document' | 'open_chat' | 'auto_document';
  selectedDocumentId?: string;
}

/** 会话标识请求 */
export interface ConversationIdentityReq {
  conversationId: string;
}

/** 对话详情查询请求 */
export interface ConversationExchangeDetailQueryReq {
  conversationId: string;
  exchangeId: string;
}

/** 会话列表查询请求 */
export interface ConversationSessionListReq {
  pageNo?: number;
  pageSize?: number;
  keyword?: string;
  chatMode?: string;
  turnStatus?: string;
}

/** 检索观察查询请求 */
export interface RetrievalObserveReq {
  conversationId: string;
  exchangeId: string;
}

// ==================== 子响应类型（嵌套基础结构） ====================
/** 检索引用 */
export interface SearchReference {
  referenceId: string;
  sourceType: string;
  title: string;
  url: string;
  snippet: string;
  documentId: string;
  documentName: string;
  chunkId: string;
  parentBlockId: string;
  parentBlockNo: number;
  chunkNo: number;
  sectionPath: string;
  structureNodeId: string;
  structureNodeType: number;
  canonicalPath: string;
  itemIndex: number;
  score: number;
  subQuestionIndex: number;
  subQuestion: string;
  channel: string;
  toolName: string;
  knowledgeScopeCode: string;
  knowledgeScopeName: string;
}

/** 工具调用轨迹 */
export interface ChatToolTrace {
  toolName: string;
  status: string;
  inputSummary: string;
  effectiveInput: string;
  outputSummary: string;
  errorMessage: string;
  referenceCount: number;
  topic: string;
  durationMs: number;
}

/** 模型使用轨迹 */
export interface ChatModelUsageTrace {
  stageName: string;
  provider: string;
  model: string;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  estimatedCost: number;
  durationMs: number;
  status: string;
}

/** 限制统计 */
export interface ChatLimitStats {
  modelCallsUsed: number;
  modelCallsRunLimit: number;
  modelCallsThreadLimit: number;
  toolCallsUsed: number;
  toolCallsRunLimit: number;
  toolCallsThreadLimit: number;
  limitTriggered: boolean;
  limitReason: string;
}

/** 会话结构锚点 */
export interface ConversationStructureAnchor {
  rootSectionCode: string;
  rootSectionTitle: string;
  targetSectionHint: string;
  structureNodeId: number;
  canonicalPath: string;
  scopeMode: string;
}

/** 会话项目锚点 */
export interface ConversationItemAnchor {
  itemIndex: number;
  itemText: string;
  structureNodeId: number;
  canonicalPath: string;
}

/** 检索问题计划 */
export interface RetrievalQuestionPlan {
  retrievalQuestion: string;
  subQuestions: string[];
}

/** 文档导航决策 */
export interface DocumentNavigationDecision {
  navigationAction: string;
  executionMode: string;
  structureAnchor: ConversationStructureAnchor | null;
  itemAnchor: ConversationItemAnchor | null;
  retrievalPlan: RetrievalQuestionPlan | null;
  summaryText: string;
  queryContextHints: string[];
  softSectionHints: string[];
}

/** 对话摘要载荷 */
export interface SummaryPayload {
  summary: string;
  conversationGoal: string;
  stableFacts: string[];
  userPreferences: string[];
  resolvedPoints: string[];
  pendingQuestions: string[];
  retrievalHints: string[];
}

/** 调试轨迹 */
export interface ChatDebugTrace {
  executionMode: string;
  chatMode: string;
  originalQuestion: string;
  rewriteQuestion: string;
  rewriteSubQuestions: string[];
  retrievalQuestion: string;
  agentQuestion: string;
  navigationDecision: DocumentNavigationDecision | null;
  historySummary: string;
  longTermSummary: string;
  recentHistoryTranscript: string;
  answerRecentTranscript: string;
  answerHistoryContext: string;
  answerHistoryFollowUpQuestion: boolean;
  historyCompressionApplied: boolean;
  historyCoveredExchangeId: number;
  historyCoveredExchangeCount: number;
  historyCompressionCount: number;
  currentDateText: string;
  requiresFreshSearch: boolean;
  requiresCurrentDateAnchoring: boolean;
  subQuestions: string[];
  selectedDocumentId: number;
  selectedTaskId: number;
  retrievalNotes: string[];
  usedChannels: string[];
  toolTraces: ChatToolTrace[];
  modelUsageTraces: ChatModelUsageTrace[];
  limitStats: ChatLimitStats | null;
  ragSystemPrompt: string;
  ragUserPrompt: string;
  noEvidenceReply: string;
}

/** 对话交换单轮交互 */
export interface ConversationExchange {
  exchangeId: string;
  question: string;
  answer: string;
  thinkingSteps: string[];
  references: SearchReference[];
  recommendations: string[];
  usedTools: string[];
  debugTrace: ChatDebugTrace | null;
  turnStatus: number;
  errorMessage: string;
  firstResponseTimeMs: number;
  totalResponseTimeMs: number;
  createTime: string;
  updateTime: string;
}

/** 会话记忆摘要 */
export interface ConversationMemorySummaryResp {
  conversationId: string;
  isCompressed: boolean;
  coveredExchangeId: string;
  coveredExchangeCount: number;
  compressionCount: number;
  summaryVersion: number;
  summaryText: string;
  summaryPayload: SummaryPayload | null;
  lastSourceUpdateTime: string;
  updateTime: string;
}

/** 阶段追踪 */
export interface ConversationTraceStage {
  id: string;
  traceId: string;
  stageCode: string;
  stageName: string;
  stageOrder: number;
  stageLevel: number;
  parentStageId: string;
  executionMode: string;
  stageState: number;
  startTime: string;
  endTime: string;
  durationMs: number;
  summaryText: string;
  errorMessage: string;
  snapshotJson: string;
}

// ==================== 顶层响应结构体 ====================
/** 会话停止响应 */
export interface ConversationStopResp {
  conversationId: string;
  stopped: boolean;
  message: string;
}

/** 会话详情单条记录响应 */
export interface ConversationSessionResp {
  conversationId: string;
  running: boolean;
  checkpointCount: number;
  messageCount: number;
  latestUserMessage: string;
  latestAssistantMessage: string;
  latestExchangeId: string;
  latestTurnStatus: number;
  latestTurnErrorMessage: string;
  chatMode: string;
  selectedDocumentId: string;
  selectedDocumentName: string;
  createdTime: string;
  updatedTime: string;
  exchanges: ConversationExchange[];
  memorySummary: ConversationMemorySummaryResp | null;
}

/** 对话详情响应 */
export interface ConversationExchangeDetailResp {
  conversationId: string;
  exchange: ConversationExchange | null;
  stageTraces: ConversationTraceStage[];
}

/** 会话列表分页响应 */
export interface ConversationSessionListResp {
  pageNo: number;
  pageSize: number;
  total: number;
  totalPages: number;
  records: ConversationSessionResp[];
}

/** 会话重置响应 */
export interface ConversationResetResp {
  conversationId: string;
  stopped: boolean;
  removedDialogueCount: number;
  removedExchangeCount: number;
  removedCheckpointCount: number;
  message: string;
}

/** 检索结果明细 */
export interface RetrievalResultResp {
  id: string;
  conversationId: string;
  exchangeId: string;
  traceId: string;
  subQuestionIndex: number;
  subQuestion: string;
  channelType: string;
  channelRank: number;
  rrfRank: number;
  finalRank: number;
  originalScore: number;
  rrfScore: number;
  rerankScore: number;
  gatePassed: number;
  isElevated: number;
  isSelected: number;
  selectionReason: string;
  chunkId: string;
  chunkNo: number;
  parentBlockId: string;
  parentBlockNo: number;
  sectionPath: string;
  chunkTextPreview: string;
  chunkCharCount: number;
  createTime: string;
  documentId: string;
  documentName: string;
}

/** 渠道执行明细 */
export interface ChannelExecutionResp {
  id: string;
  conversationId: string;
  exchangeId: string;
  traceId: string;
  subQuestionIndex: number;
  subQuestion: string;
  channelType: string;
  executionState: number;
  startTime: string;
  endTime: string;
  durationMs: number;
  recalledCount: number;
  acceptedCount: number;
  finalSelectedCount: number;
  avgScore: number;
  maxScore: number;
  minScore: number;
  errorMessage: string;
  createTime: string;
}

export interface Snapshot {
  recommendationCount?: number;
  recommendations?: string[];
  finalStatus?: string;
  referenceCount?: number;
  answerLength?: number;
  errorMessage?: string;
  clarificationReply?: string;
  clarificationReason?: string;
  clarificationOptions?: string[];
  targetSection?: string;
  parentSection?: string;
  childCount?: number;
  previousSibling?: string;
  nextSibling?: string;
  answer?: string;
  targetItemIndex?: string,
  notes?: string[],
  matchedItemCount?: number;
  retrievalQuestion?: string;
  usedChannels?: string[];
  retrievalNotes?: string[];
  subQuestionCount?: number;
  subQuestions?: (SubQuestion | string)[];
  references?: Reference[];
  totalBudget?: number;
  perSubQuestionBudget?: number;
  renderedReferenceCount?: number;
  omittedReferenceCount?: number;
  renderedReferenceDetails?: string[];
  omittedReferenceDetails?: string[];
  systemPrompt?: string;
  userPrompt?: string;
  firstResponseTimeMs?: number;
  chatMode?: string;
  executionMode?: string;
  requiresRealTimeSearch?: boolean;
  requiresCurrentDateAnchoring?: boolean;
  confidence?: number;
  routeStatus?: string;
  candidateDocumentCount?: number;
  confidentTopDocument?: boolean;
  topDocumentId?: string;
  topDocumentName?: string;
  targetSectionHint?: string;
  navigationSummary?: string;
  compressionApplied?: boolean;
  coveredExchangeId?: string;
  coveredExchangeCount?: number;
  compressionCount?: number;
  longTermSummary?: string;
  recentTranscript?: string;
  recentQuestionTranscript?: string;
  originalQuestion?: string;
  historyContext?: string;
  rewriteQuestion?: string;
  rawModelOutput?: string;
  rewriteOverrideEnabled?: boolean;
  rewriteTemperature?: number;
  rewriteTopP?: number;
  rewriteThinking?: boolean;
}

export interface SubQuestion {
  index: number;
  question: string;
  referenceCount: number;
  documentCount: number;
  fusedCandidateCount: number;
  parentCandidateCount: number;
  rerankedCandidateCount: number;
  channelTraces: ChannelTrace[];
  references: Reference[];
}

export interface ChannelTrace {
  channelName: string;
  recalledCount: number;
  acceptedCount: number;
}

export interface Reference {
  referenceId: string;
  documentName: string;
  title: string;
  sectionPath: string;
  channel: string;
}
