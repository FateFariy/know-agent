export interface ChatReq {
  question: string
  conversationId?: string
  chatMode?: string
  selectedDocumentId?: number
}

export interface ConversationIdentityReq {
  conversationId: string
}

export interface ConversationExchangeDetailQueryReq {
  conversationId: string
  exchangeId: number
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
  exchangeId: number
}

export interface UploadDocumentReq {
  documentName?: string
  operatorId?: number
  knowledgeScopeCode?: string
  knowledgeScopeName?: string
  businessCategory?: string
  documentTags?: string
}

export interface QueryDocumentPageReq {
  pageNo?: number
  pageSize?: number
  keyword?: string
}

export interface QueryDocumentDetailReq {
  documentId: number
}

export interface DeleteDocumentReq {
  documentId: number
}

export interface QueryStrategyPlanReq {
  documentId: number
}

export interface ConfirmStrategyReq {
  documentId: number
  basePlanId: number
  operatorId?: number
  adjustNote?: string
  parentSteps: StrategyStepItem[]
  childSteps: StrategyStepItem[]
}

export interface BuildIndexReq {
  documentId: number
  planId: number
  operatorId?: number
}

export interface QueryDocumentChunksReq {
  documentId: number
  taskId?: number
  pageNo?: number
  pageSize?: number
}

export interface QueryDocumentChunkDetailReq {
  documentId: number
  chunkId: number
  taskId?: number
}

export interface QueryTaskLogsReq {
  taskId: number
  pageNo?: number
  pageSize?: number
}

export interface DocumentProfileDetailReq {
  documentId: number
}

export interface DocumentProfileRegenerateReq {
  documentId: number
  operatorId?: string
}

export interface DocumentProfileBatchRegenerateReq {
  documentIds: number[]
  operatorId?: string
}

export interface KnowledgeScopeSaveReq {
  id?: number
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
  id?: number
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
  documentId: number
  relationScore?: number
  relationSource?: string
  reason?: string
  operatorId?: string
}

export interface TopicDocumentRelationRemoveReq {
  topicCode: string
  documentId: number
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