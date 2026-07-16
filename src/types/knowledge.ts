// 知识库类型
/** 保存知识范围请求 */
export interface KnowledgeScopeSaveReq {
  id?: string;
  scopeCode: string;
  scopeName: string;
  parentScopeCode?: string;
  description?: string;
  aliases?: string;
  examples?: string;
  sortOrder?: number;
  operatorId?: string;
}

/** 删除知识范围请求 */
export interface KnowledgeScopeDeleteReq {
  scopeCode: string;
  operatorId?: string;
}

/** 保存知识主题请求 */
export interface KnowledgeTopicSaveReq {
  id?: string;
  topicCode: string;
  topicName: string;
  scopeCode: string;
  description?: string;
  aliases?: string;
  examples?: string;
  answerShape?: string;
  executionPreference?: string;
  sortOrder?: number;
  operatorId?: string;
}

/** 删除知识主题请求 */
export interface KnowledgeTopicDeleteReq {
  topicCode: string;
  operatorId?: string;
}

/** 查询知识主题列表请求 */
export interface KnowledgeTopicListReq {
  scopeCode?: string;
}

/** 查询主题文档关联列表请求 */
export interface TopicDocumentRelationListReq {
  topicCode: string;
}

/** 保存主题文档关联请求 */
export interface TopicDocumentRelationSaveReq {
  topicCode: string;
  documentId: string;
  relationScore?: number;
  relationSource?: string;
  reason?: string;
  operatorId?: string;
}

/** 移除主题文档关联请求 */
export interface TopicDocumentRelationRemoveReq {
  topicCode: string;
  documentId: string;
  operatorId?: string;
}

/** 分页查询知识路由追踪请求 */
export interface KnowledgeRouteTracePageReq {
  conversationId?: string;
  mode?: string;
  routeStatus?: number;
  pageNo?: number;
  pageSize?: number;
}

// ====================== 内嵌子响应类型 ======================
/** 知识路由追踪明细 */
export interface KnowledgeRouteTraceItem {
  id: string;
  conversationId: string;
  exchangeId: string;
  question: string;
  rewriteQuestion: string;
  mode: string;
  topScopesJson: string;
  topTopicsJson: string;
  topDocumentsJson: string;
  selectedDocumentId: string;
  hitSelectedDocument: number;
  confidence: number;
  routeStatus: string;
  errorMsg: string;
  createTime: string;
}

/** 路由候选基础类型 */
export interface BaseRouteCandidate {
  score: number
  reason: string
  scoreText: string
}

/** 知识范围路由候选 */
export interface ScopeRouteCandidate extends BaseRouteCandidate {
  scopeCode: string
  scopeName: string
}

/** 主题路由候选 */
export interface TopicRouteCandidate extends BaseRouteCandidate {
  topicCode: string
  topicName: string
  scopeCode: string
}

/** 文档路由候选 */
export interface DocumentRouteCandidate extends BaseRouteCandidate {
  documentId: string
  documentName: string
  lastIndexTaskId: string
}

// ====================== 顶层响应类型 ======================
/** 知识范围响应 */
export interface KnowledgeScopeResp {
  id: string;
  scopeCode: string;
  scopeName: string;
  parentScopeCode: string;
  description: string;
  aliases: string;
  examples: string;
  sortOrder: number;
}

/** 知识主题响应 */
export interface KnowledgeTopicResp {
  id: string;
  topicCode: string;
  topicName: string;
  scopeCode: string;
  description: string;
  aliases: string;
  examples: string;
  answerShape: string;
  executionPreference: string;
  sortOrder: number;
}

/** 主题文档关联响应 */
export interface TopicDocumentRelationResp {
  topicCode: string;
  documentId: string;
  documentName: string;
  knowledgeScopeCode: string;
  knowledgeScopeName: string;
  businessCategory: string;
  documentTags: string;
  relationScore: number;
  relationSource: string;
  reason: string;
}

/** 分页查询知识路由追踪响应 */
export interface KnowledgeRouteTracePageResp {
  pageNo: number;
  pageSize: number;
  total: number;
  totalPages: number;
  records: KnowledgeRouteTraceItem[];
}
