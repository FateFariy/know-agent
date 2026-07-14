import axios from './axios'
import type { KnowledgeScope, KnowledgeTopic, TopicDocumentRelation, PageResult, RouteTrace } from '@/types'

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

export const knowledgeApi = {
  listScopes() {
    return axios.post<KnowledgeScope[]>('/manage/knowledge/scope/list')
  },

  saveScope(params: KnowledgeScopeSaveReq) {
    return axios.post<KnowledgeScope>('/manage/knowledge/scope/save', params)
  },

  deleteScope(params: KnowledgeScopeDeleteReq) {
    return axios.post<boolean>('/manage/knowledge/scope/delete', params)
  },

  listTopics(params?: KnowledgeTopicListReq) {
    return axios.post<KnowledgeTopic[]>('/manage/knowledge/topic/list', params || {})
  },

  saveTopic(params: KnowledgeTopicSaveReq) {
    return axios.post<KnowledgeTopic>('/manage/knowledge/topic/save', params)
  },

  deleteTopic(params: KnowledgeTopicDeleteReq) {
    return axios.post<boolean>('/manage/knowledge/topic/delete', params)
  },

  listTopicDocuments(params: TopicDocumentRelationListReq) {
    return axios.post<TopicDocumentRelation[]>('/manage/knowledge/topic/document/list', params)
  },

  saveTopicDocument(params: TopicDocumentRelationSaveReq) {
    return axios.post<TopicDocumentRelation>('/manage/knowledge/topic/document/save', params)
  },

  removeTopicDocument(params: TopicDocumentRelationRemoveReq) {
    return axios.post<boolean>('/manage/knowledge/topic/document/remove', params)
  },

  queryRouteTracePage(params?: KnowledgeRouteTracePageReq) {
    return axios.post<PageResult<RouteTrace>>('/manage/knowledge/route/trace/page/query', params || {})
  },
}
