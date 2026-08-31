// 知识库类型
/** 保存知识范围请求 */
export interface KnowledgeScopeSaveReq {
  id?: string;
  knowledgeBaseId: string;
  scopeName: string;
  parentScopeId?: string;
  description?: string;
  aliases?: string;
  examples?: string;
  sortOrder?: number;
  operatorId?: string;
}

/** 删除知识范围请求 */
export interface KnowledgeScopeDeleteReq {
  id?: string;
  knowledgeBaseId?: string;
  operatorId?: string;
}

/** 查询知识范围列表请求 */
export interface KnowledgeScopeListReq {
  knowledgeBaseId?: string;
}

/** 保存知识主题请求 */
export interface KnowledgeTopicSaveReq {
  id?: string;
  knowledgeBaseId: string;
  scopeId: string;
  topicName: string;
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
  id: string;
  knowledgeBaseId: string;
  operatorId?: string;
}

/** 查询知识主题列表请求 */
export interface KnowledgeTopicListReq {
  knowledgeBaseId: string;
  scopeId: string;
}

/** 查询主题文档关联列表请求 */
export interface TopicDocumentRelationListReq {
  topicId: string;
  knowledgeBaseId: string;
}

/** 保存主题文档关联请求 */
export interface TopicDocumentRelationSaveReq {
  topicId: string;
  knowledgeBaseId: string;
  documentId: string;
  relationScore?: number;
  relationSource?: string;
  reason?: string;
  operatorId?: string;
}

/** 移除主题文档关联请求 */
export interface TopicDocumentRelationRemoveReq {
  topicId: string;
  knowledgeBaseId: string;
  documentId: string;
  operatorId?: string;
}

/** 分页查询知识路由追踪请求 */
export interface KnowledgeRouteTracePageReq {
  /** 会话ID（后端必填） */
  conversationId: string;
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
  reason: string;
  createTime: string;
}

/** 路由候选基础类型（由 top*Json 反序列化后再归一化得到） */
export interface BaseRouteCandidate {
  score: number
  reason: string
  source: string
  features?: Record<string, number>
  /** 前端归一化补充的展示用分数文本 */
  scoreText: string
}

/** 知识范围路由候选 */
export interface ScopeRouteCandidate extends BaseRouteCandidate {
  scopeId: string
  scopeName: string
}

/** 主题路由候选 */
export interface TopicRouteCandidate extends BaseRouteCandidate {
  topicId: string
  topicName: string
  scopeId: string
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
  knowledgeBaseId: string;
  scopeName: string;
  parentScopeId: string;
  description: string;
  aliases: string;
  examples: string;
  sortOrder: number;
}

/** 知识主题响应 */
export interface KnowledgeTopicResp {
  id: string;
  knowledgeBaseId: string;
  topicName: string;
  scopeId: string;
  description: string;
  aliases: string;
  examples: string;
  answerShape: string;
  executionPreference: string;
  sortOrder: number;
}

/** 主题文档关联响应 */
export interface TopicDocumentRelationResp {
  knowledgeBaseId: string;
  topicId: string;
  topicName: string;
  scopeId: string;
  scopeName: string;
  documentId: string;
  documentName: string;
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

/** 知识库明细响应 */
export interface KnowledgeBaseItemResp {
  id: string;
  baseName: string;
  description: string;
  embeddingModel: string;
  retrievalConfigJson: string;
  graphRagConfigJson: string;
  raptorConfigJson: string;
  metadataFilterJson: string;
  isDefault: number;
  sortOrder: number;
  documentCount: number;
  retrievableDocumentCount: number;
}

/** 知识库选项响应 */
export interface KnowledgeBaseOptionResp {
  id: string;
  baseName: string;
  description: string;
  isDefault: number;
  retrievableDocumentCount: number;
}
