import axios from './axios'
import type {
  Response,
  KnowledgeScopeResp,
  KnowledgeTopicResp,
  TopicDocumentRelationResp,
  KnowledgeRouteTracePageResp,
  KnowledgeScopeSaveReq,
  KnowledgeScopeDeleteReq,
  KnowledgeTopicSaveReq,
  KnowledgeTopicDeleteReq,
  KnowledgeTopicListReq,
  TopicDocumentRelationListReq,
  TopicDocumentRelationSaveReq,
  TopicDocumentRelationRemoveReq,
  KnowledgeRouteTracePageReq,
} from '@/types'

export const knowledgeApi = {
  // 查询知识范围列表
  listScopes(): Promise<Response<KnowledgeScopeResp[]>> {
    return axios.post('/manage/knowledge/scope/list')
  },

  // 保存知识范围
  saveScope(params: KnowledgeScopeSaveReq): Promise<Response<KnowledgeScopeResp>> {
    return axios.post('/manage/knowledge/scope/save', params)
  },

  // 删除知识范围
  deleteScope(params: KnowledgeScopeDeleteReq): Promise<Response<boolean>> {
    return axios.post('/manage/knowledge/scope/delete', params)
  },

  // 查询知识主题列表
  listTopics(params?: KnowledgeTopicListReq): Promise<Response<KnowledgeTopicResp[]>> {
    return axios.post('/manage/knowledge/topic/list', params || {})
  },

  // 保存知识主题
  saveTopic(params: KnowledgeTopicSaveReq): Promise<Response<KnowledgeTopicResp>> {
    return axios.post('/manage/knowledge/topic/save', params)
  },

  // 删除知识主题
  deleteTopic(params: KnowledgeTopicDeleteReq): Promise<Response<boolean>> {
    return axios.post('/manage/knowledge/topic/delete', params)
  },

  // 查询知识主题下的文档列表
  listTopicDocuments(params: TopicDocumentRelationListReq): Promise<Response<TopicDocumentRelationResp[]>> {
    return axios.post('/manage/knowledge/topic/document/list', params)
  },

  // 保存知识主题下的文档
  saveTopicDocument(params: TopicDocumentRelationSaveReq): Promise<Response<TopicDocumentRelationResp>> {
    return axios.post('/manage/knowledge/topic/document/save', params)
  },

  // 删除知识主题下的文档
  removeTopicDocument(params: TopicDocumentRelationRemoveReq): Promise<Response<boolean>> {
    return axios.post('/manage/knowledge/topic/document/remove', params)
  },

  // 查询知识路由轨迹分页列表
  queryRouteTracePage(params?: KnowledgeRouteTracePageReq): Promise<Response<KnowledgeRouteTracePageResp>> {
    return axios.post('/manage/knowledge/route/trace/page/query', params || {})
  },
}
