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
  // 查询知识范围列表
  listScopes(): Promise<Response<KnowledgeScope[]>> {
    return axios.post('/manage/knowledge/scope/list')
  },

  // 保存知识范围
  saveScope(params: KnowledgeScopeSaveReq): Promise<Response<KnowledgeScope>> {
    return axios.post('/manage/knowledge/scope/save', params)
  },

  // 删除知识范围
  deleteScope(params: KnowledgeScopeDeleteReq): Promise<Response<boolean>> {
    return axios.post('/manage/knowledge/scope/delete', params)
  },

  // 查询知识主题列表
  listTopics(params?: KnowledgeTopicListReq): Promise<Response<KnowledgeTopic[]>> {
    return axios.post('/manage/knowledge/topic/list', params || {})
  },

  // 保存知识主题
  saveTopic(params: KnowledgeTopicSaveReq): Promise<Response<KnowledgeTopic>> {
    return axios.post('/manage/knowledge/topic/save', params)
  },

  // 删除知识主题
  deleteTopic(params: KnowledgeTopicDeleteReq): Promise<Response<boolean>> {
    return axios.post('/manage/knowledge/topic/delete', params)
  },

  // 查询知识主题下的文档列表
  listTopicDocuments(params: TopicDocumentRelationListReq): Promise<Response<TopicDocumentRelation[]>> {
    return axios.post('/manage/knowledge/topic/document/list', params)
  },

  // 保存知识主题下的文档
  saveTopicDocument(params: TopicDocumentRelationSaveReq): Promise<Response<TopicDocumentRelation>> {
    return axios.post('/manage/knowledge/topic/document/save', params)
  },

  // 删除知识主题下的文档
  removeTopicDocument(params: TopicDocumentRelationRemoveReq): Promise<Response<boolean>> {
    return axios.post('/manage/knowledge/topic/document/remove', params)
  },

  // 查询知识路由轨迹分页列表
  queryRouteTracePage(params?: KnowledgeRouteTracePageReq): Promise<Response<PageResult<RouteTrace>>> {
    return axios.post('/manage/knowledge/route/trace/page/query', params || {})
  },
}
