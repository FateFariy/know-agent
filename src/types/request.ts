export interface ChatReq {
  question: string
  conversationId?: string
  chatMode?: string
  selectedDocumentId?: string
}

export interface ConversationIdentityReq {
  conversationId: string
}

export interface ConversationExchangeDetailQueryReq {
  conversationId: string
  exchangeId: string
}

export interface ConversationSessionListReq {
  pageNo?: number
  pageSize?: number
  keyword?: string
  chatMode?: string
  turnStatus?: string
}

export interface RetrievalObserveReq {
  conversationId: string
  exchangeId: string
}

export interface UploadDocumentReq {
  documentName?: string
  operatorId?: string
  knowledgeScopeCode?: string
  knowledgeScopeName?: string
  businessCategory?: string
  documentTags?: string
  file: File | null
}

export interface QueryDocumentPageReq {
  pageNo?: number
  pageSize?: number
  keyword?: string
}

export interface QueryDocumentDetailReq {
  documentId: string
}

export interface DeleteDocumentReq {
  documentId: string
}

export interface QueryStrategyPlanReq {
  documentId: string
}

export interface ConfirmStrategyReq {
  documentId: string
  basePlanId: string
  operatorId?: string
  adjustNote?: string
  parentSteps: StrategyStepItem[]
  childSteps: StrategyStepItem[]
}

export interface BuildIndexReq {
  documentId: string
  planId: string
  operatorId?: string
}

export interface QueryDocumentChunksReq {
  documentId: string
  taskId?: string
  pageNo?: number
  pageSize?: number
}

export interface QueryDocumentChunkDetailReq {
  documentId: string
  chunkId: string
  taskId?: string
}

export interface QueryTaskLogsReq {
  taskId: string
  pageNo?: number
  pageSize?: number
}

export interface DocumentProfileDetailReq {
  documentId: string
}

export interface DocumentProfileRegenerateReq {
  documentId: string
  operatorId?: string
}

export interface DocumentProfileBatchRegenerateReq {
  documentIds: string[]
  operatorId?: string
}

export interface KnowledgeScopeSaveReq {
  id?: string
  scopeCode: string
  scopeName: string
  parentScopeCode?: string
  description?: string
  aliases?: string
  examples?: string
  sortOrder?: number
  operatorId?: string
}

export interface KnowledgeScopeDeleteReq {
  scopeCode: string
  operatorId?: string
}

export interface KnowledgeTopicSaveReq {
  id?: string
  topicCode: string
  topicName: string
  scopeCode: string
  description?: string
  aliases?: string
  examples?: string
  answerShape?: string
  executionPreference?: string
  sortOrder?: number
  operatorId?: string
}

export interface KnowledgeTopicDeleteReq {
  topicCode: string
  operatorId?: string
}

export interface KnowledgeTopicListReq {
  scopeCode?: string
}

export interface TopicDocumentRelationListReq {
  topicCode: string
}

export interface TopicDocumentRelationSaveReq {
  topicCode: string
  documentId: string
  relationScore?: number
  relationSource?: string
  reason?: string
  operatorId?: string
}

export interface TopicDocumentRelationRemoveReq {
  topicCode: string
  documentId: string
  operatorId?: string
}

export interface KnowledgeRouteTracePageReq {
  conversationId?: string
  mode?: string
  routeStatus?: number
  pageNo?: number
  pageSize?: number
}

export interface StrategyStepItem {
  stepNo: number
  strategyType: number
}
