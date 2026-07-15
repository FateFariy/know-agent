export interface DocumentInfo {
  documentId: string
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
  currentPlanId: string
  lastIndexTaskId: string
  latestTaskId: string
  latestTaskType: number
  latestTaskTypeName: string
  latestTaskStatus: number
  latestTaskStatusName: string
  createTime: string
  updateTime: string
}

export interface DocumentChunk {
  chunkId: string
  parentBlockId: string
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
  parentBlockId: string
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
  documentId: string
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
  documentId: string
  documentName: string
  knowledgeScopeName: string
  businessCategory: string
  documentTags: string[]
}

export interface Exchange {
  exchangeId: string
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
  documentId: string
  documentName: string
  chunkId: string
  parentBlockId: string
  parentBlockNo: number
  chunkNo: number
  sectionPath: string
  structureNodeId: string
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
  selectedDocumentId: string
  selectedDocumentName: string
  latestUserMessage: string
  latestAssistantMessage: string
  latestExchangeId: string
  latestTurnStatus: string
  latestTurnErrorMessage: string
  memorySummary: MemorySummary
}

export interface MemorySummary {
  conversationId: string
  IsCompressed: boolean
  coveredExchangeId: string
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
  latestExchangeId: string
  latestTurnStatus: string
  latestTurnErrorMessage: string
  chatMode: string
  selectedDocumentId: string
  selectedDocumentName: string
  createdTime: string
  updatedTime: string
  exchanges: Exchange[]
  memorySummary: MemorySummary
}

export interface KnowledgeScope {
  id: string
  scopeCode: string
  scopeName: string
  parentScopeCode: string
  description: string
  aliases: string
  examples: string
  sortOrder: number
}

export interface KnowledgeTopic {
  id: string
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
  documentId: string
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
  documentId?: string
}

export interface StageTrace {
  id: string
  traceId: string
  stageCode: string
  stageName: string
  stageOrder: number
  stageLevel: number
  parentStageId: string
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
  id: string
  conversationId: string
  exchangeId: string
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
  chunkId: string
  chunkNo: number
  parentBlockId: string
  parentBlockNo: number
  sectionPath: string
  chunkTextPreview: string
  chunkCharCount: number
  createTime: string
}

export interface ChannelExecution {
  id: string
  conversationId: string
  exchangeId: string
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
  id: string
  conversationId: string
  exchangeId: string
  question: string
  rewriteQuestion: string
  mode: string
  topScopesJson: string
  topTopicsJson: string
  topDocumentsJson: string
  selectedDocumentId: string
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
  planId: string
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
  documentId: string
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
  planId: string
  documentId: string
  planVersion: number
  strategyStatus: number
  strategyStatusName: string
  normalized: boolean
  parentPipeline: DocumentStrategyPipeline
  childPipeline: DocumentStrategyPipeline
}

export interface TaskLog {
  id: string
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
  taskId: string
  documentId: string
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
  documentId: string
  taskId: string
  documentName: string
  parseStatus: number
  strategyStatus: number
  indexStatus: number
}

export interface BuildIndexResp {
  taskId: string
  documentId: string
  taskType: number
  taskTypeName: string
  taskStatus: number
  taskStatusName: string
  indexStatus: number
  indexStatusName: string
}

export interface QueryDocumentChunksResp {
  documentId: string
  taskId: string
  planId: string
  pageNo: number
  pageSize: number
  total: number
  records: DocumentChunk[]
}

export interface QueryDocumentChunkDetailResp {
  documentId: string
  taskId: string
  planId: string
  chunk: DocumentChunk
  parentBlock: DocumentParentBlock
  siblingChunks: DocumentChunk[]
}

export interface DeleteDocumentResp {
  documentId: string
  documentName: string
}

export interface GetExchangeDetailResponse {
  conversationId: string
  exchange: Exchange
  stageTraces: StageTrace[]
}
