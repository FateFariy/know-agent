import axios from './axios'
import type {
  Response,
  KnowledgeScope,
  KnowledgeTopic,
  TopicDocumentRelation,
  PageResult,
  RouteTrace,
  KnowledgeScopeSaveReq,
  KnowledgeScopeDeleteReq,
  KnowledgeTopicSaveReq,
  KnowledgeTopicDeleteReq,
  KnowledgeTopicListReq,
  TopicDocumentRelationListReq,
  TopicDocumentRelationSaveReq,
  TopicDocumentRelationRemoveReq,
  KnowledgeRouteTracePageReq
} from '@/types'

export const knowledgeApi = {
  listScopes(): Promise<Response<KnowledgeScope[]>> {
    return axios.post('/manage/knowledge/scope/list')
  },

  saveScope(params: KnowledgeScopeSaveReq): Promise<Response<KnowledgeScope>> {
    return axios.post('/manage/knowledge/scope/save', params)
  },

  deleteScope(params: KnowledgeScopeDeleteReq): Promise<Response<boolean>> {
    return axios.post('/manage/knowledge/scope/delete', params)
  },

  listTopics(params?: KnowledgeTopicListReq): Promise<Response<KnowledgeTopic[]>> {
    return axios.post('/manage/knowledge/topic/list', params || {})
  },

  saveTopic(params: KnowledgeTopicSaveReq): Promise<Response<KnowledgeTopic>> {
    return axios.post('/manage/knowledge/topic/save', params)
  },

  deleteTopic(params: KnowledgeTopicDeleteReq): Promise<Response<boolean>> {
    return axios.post('/manage/knowledge/topic/delete', params)
  },

  listTopicDocuments(params: TopicDocumentRelationListReq): Promise<Response<TopicDocumentRelation[]>> {
    return axios.post('/manage/knowledge/topic/document/list', params)
  },

  saveTopicDocument(params: TopicDocumentRelationSaveReq): Promise<Response<TopicDocumentRelation>> {
    return axios.post('/manage/knowledge/topic/document/save', params)
  },

  removeTopicDocument(params: TopicDocumentRelationRemoveReq): Promise<Response<boolean>> {
    return axios.post('/manage/knowledge/topic/document/remove', params)
  },

  queryRouteTracePage(params?: KnowledgeRouteTracePageReq): Promise<Response<PageResult<RouteTrace>>> {
    return axios.post('/manage/knowledge/route/trace/page/query', params || {})
  },
}
