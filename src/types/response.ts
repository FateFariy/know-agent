export interface DocumentInfo {
  documentId: number
  documentName: string
  originalFileName: string
  fileType: number
  fileTypeName: string
  fileSize: number
  charCount: number
  tokenCount: number
  parseStatus: number
  parseStatusName: string
  strategyStatus: number
  strategyStatusName: string
  indexStatus: number
  indexStatusName: string
  parseErrorMsg: string
  knowledgeScopeCode: string
  knowledgeScopeName: string
  businessCategory: string
  documentTags: string
  currentPlanId: number
  lastIndexTaskId: number
  latestTaskId: number
  latestTaskType: number
  latestTaskTypeName: string
  latestTaskStatus: number
  latestTaskStatusName: string
  createTime: string
  updateTime: string
}

export interface DocumentChunk {
  chunkId: number
  parentBlockId: number
  parentBlockNo: number
  parentChildCount: number
  parentStartChunkNo: number
  parentEndChunkNo: number
  chunkNo: number
  sectionPath: string
  sourceType: number
  sourceTypeName: string
  charCount: number
  tokenCount: number
  vectorStatus: number
  vectorStatusName: string
  chunkText: string
}

export interface DocumentParentBlock {
  parentBlockId: number
  parentBlockNo: number
  sectionPath: string
  sourceType: number
  sourceTypeName: string
  charCount: number
  tokenCount: number
  childCount: number
  startChunkNo: number
  endChunkNo: number
  parentText: string
}

export interface DocumentProfile {
  documentId: number
  documentSummary: string
  documentType: string
  coreTopics: string
  exampleQuestions: string
  graphFriendly: number
  supportsGraphOutline: number
  supportsItemLookup: number
  supportsGraphAssist: number
  profileSource: string
  profileStatus: number
  errorMsg: string
}

export interface DocumentOption {
  documentId: number
  documentName: string
  knowledgeScopeName: string
  businessCategory: string
  documentTags: string[]
}

export interface Exchange {
  exchangeId: number
  question: string
  answer: string
  thinkingSteps: string[]
  references: Reference[]
  recommendations: string[]
  usedTools: string[]
  debugTrace: string
  status: number
  errorMessage: string
  firstResponseTimeMs: number
  totalResponseTimeMs: number
  createTime: string
  updateTime?: string
}

export interface Reference {
  referenceId: string
  sourceType: string
  title: string
  url: string
  snippet: string
  documentId: number
  documentName: string
  chunkId: number
  parentBlockId: number
  parentBlockNo: number
  chunkNo: number
  sectionPath: string
  structureNodeId: number
  structureNodeType: number
  canonicalPath: string
  itemIndex: number
  score: number
  subQuestionIndex: number
  subQuestion: string
  channel: string
  toolName: string
  knowledgeScopeCode: string
  knowledgeScopeName: string
}

export interface SessionDetail {
  conversationId: string
  chatMode: string
  checkpointCount: number
  createdTime: string
  updatedTime: string
  exchanges: Exchange[]
  messageCount: number
  running: boolean
  selectedDocumentId: number
  selectedDocumentName: string
  latestUserMessage: string
  latestAssistantMessage: string
  latestExchangeId: number
  latestTurnStatus: string
  latestTurnErrorMessage: string
  memorySummary: MemorySummary
}

export interface MemorySummary {
  conversationId: string
  IsCompressed: boolean
  coveredExchangeId: number
  coveredExchangeCount: number
  compressionCount: number
  summaryVersion: number
  summaryText: string
  summaryPayload: SummaryPayload
  lastSourceUpdateTime: string
  updateTime: string
}

export interface SummaryPayload {
  summary: string
  conversationGoal: string
  stableFacts: string[]
  userPreferences: string[]
  resolvedPoints: string[]
  pendingQuestions: string[]
  retrievalHints: string[]
}

export interface SessionListItem {
  conversationId: string
  running: boolean
  checkpointCount: number
  messageCount: number
  latestUserMessage: string
  latestAssistantMessage: string
  latestExchangeId: number
  latestTurnStatus: string
  latestTurnErrorMessage: string
  chatMode: string
  selectedDocumentId: number
  selectedDocumentName: string
  createdTime: string
  updatedTime: string
  exchanges: Exchange[]
  memorySummary: MemorySummary
}

export interface KnowledgeScope {
  id: number
  scopeCode: string
  scopeName: string
  parentScopeCode: string
  description: string
  aliases: string
  examples: string
  sortOrder: number
}

export interface KnowledgeTopic {
  id: number
  topicCode: string
  topicName: string
  scopeCode: string
  description: string
  aliases: string
  examples: string
  answerShape: string
  executionPreference: string
  sortOrder: number
}

export interface TopicDocumentRelation {
  topicCode: string
  documentId: number
  documentName: string
  knowledgeScopeCode: string
  knowledgeScopeName: string
  businessCategory: string
  documentTags: string
  relationScore: number
  relationSource: string
  reason: string
}

export interface PageResult<T> {
  pageNo: number
  pageSize: number
  total: number
  totalPages: number
  records: T[]
}

export interface UploadFile {
  file: File
  fileName: string
  fileSize: number
  progress: number
  status: 'pending' | 'uploading' | 'success' | 'error'
  errorMessage?: string
  documentId?: number
}

export interface StageTrace {
  id: number
  traceId: string
  stageCode: string
  stageName: string
  stageOrder: number
  stageLevel: number
  parentStageId: number
  executionMode: string
  stageState: string
  startTime: string
  endTime: string
  durationMs: number
  summaryText: string
  errorMessage: string
  snapshot: string
}

export interface RetrievalResult {
  id: number
  conversationId: string
  exchangeId: number
  traceId: string
  subQuestionIndex: number
  subQuestion: string
  channelType: string
  channelRank: number
  rrfRank: number
  finalRank: number
  originalScore: number
  rrfScore: number
  rerankScore: number
  gatePassed: number
  isElevated: number
  isSelected: number
  selectionReason: string
  chunkId: number
  chunkNo: number
  parentBlockId: number
  parentBlockNo: number
  sectionPath: string
  chunkTextPreview: string
  chunkCharCount: number
  createTime: string
}

export interface ChannelExecution {
  id: number
  conversationId: string
  exchangeId: number
  traceId: string
  subQuestionIndex: number
  subQuestion: string
  channelType: string
  executionState: number
  startTime: string
  endTime: string
  durationMs: number
  recalledCount: number
  acceptedCount: number
  finalSelectedCount: number
  avgScore: number
  maxScore: number
  minScore: number
  errorMessage: string
  createTime: string
}

export interface ConversationResetResp {
  conversationId: string
  stopped: boolean
  removedDialogueCount: number
  removedExchangeCount: number
  removedCheckpointCount: number
  message: string
}

export interface ConversationStopResp {
  conversationId: string
  stopped: boolean
  message: string
}

export interface RouteTrace {
  id: number
  conversationId: string
  exchangeId: number
  question: string
  rewriteQuestion: string
  mode: string
  topScopesJson: string
  topTopicsJson: string
  topDocumentsJson: string
  selectedDocumentId: number
  hitSelectedDocument: number
  confidence: number
  routeStatus: string
  errorMsg: string
  createTime: string
}

export interface DocumentStrategyStep {
  stepNo: number
  pipelineType: string
  pipelineTypeName: string
  strategyType: number
  strategyTypeName: string
  strategyRole: number
  strategyRoleName: string
  sourceType: number
  sourceTypeName: string
  executeStatus: number
  executeStatusName: string
  recommendReason: string
}

export interface DocumentStrategyPipeline {
  pipelineType: string
  pipelineTypeName: string
  strategySnapshot: string
  steps: DocumentStrategyStep[]
}

export interface DocumentStrategyPlan {
  planId: number
  planVersion: number
  planSource: number
  planSourceName: string
  planStatus: number
  planStatusName: string
  strategySnapshot: string
  recommendReason: string
  parentPipeline: DocumentStrategyPipeline
  childPipeline: DocumentStrategyPipeline
}

export interface QueryStrategyPlanResp {
  documentId: number
  documentName: string
  parseStatus: number
  parseStatusName: string
  strategyStatus: number
  strategyStatusName: string
  indexStatus: number
  indexStatusName: string
  parseErrorMsg: string
  planReady: boolean
  plan: DocumentStrategyPlan
}

export interface ConfirmStrategyResp {
  planId: number
  documentId: number
  planVersion: number
  strategyStatus: number
  strategyStatusName: string
  normalized: boolean
  parentPipeline: DocumentStrategyPipeline
  childPipeline: DocumentStrategyPipeline
}

export interface TaskLog {
  id: number
  stageType: number
  stageTypeName: string
  eventType: number
  eventTypeName: string
  logLevel: number
  logLevelName: string
  content: string
  detailJson: string
  createTime: string
}

export interface QueryTaskLogsResp {
  taskId: number
  documentId: number
  taskType: number
  taskTypeName: string
  taskStatus: number
  taskStatusName: string
  currentStage: number
  currentStageName: string
  startTime: string
  finishTime: string
  costMillis: number
  errorCode: string
  errorMsg: string
  total: number
  logs: TaskLog[]
}

export interface UploadDocumentResp {
  documentId: number
  taskId: number
  documentName: string
  parseStatus: number
  strategyStatus: number
  indexStatus: number
}

export interface BuildIndexResp {
  taskId: number
  documentId: number
  taskType: number
  taskTypeName: string
  taskStatus: number
  taskStatusName: string
  indexStatus: number
  indexStatusName: string
}

export interface QueryDocumentChunksResp {
  documentId: number
  taskId: number
  planId: number
  pageNo: number
  pageSize: number
  total: number
  records: DocumentChunk[]
}

export interface QueryDocumentChunkDetailResp {
  documentId: number
  taskId: number
  planId: number
  chunk: DocumentChunk
  parentBlock: DocumentParentBlock
  siblingChunks: DocumentChunk[]
}

export interface DeleteDocumentResp {
  documentId: number
  documentName: string
}

export interface GetExchangeDetailResponse {
  conversationId: string
  exchange: Exchange
  stageTraces: StageTrace[]
}
