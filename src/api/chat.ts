import axios from './axios'
import type {
  Response,
  DocumentOption,
  SessionDetail,
  SessionListItem,
  PageResult,
  RetrievalResult,
  ChannelExecution,
  ConversationStopResp,
  ConversationResetResp,
  GetExchangeDetailResponse,
  ChatReq,
  ConversationIdentityReq,
  ConversationExchangeDetailQueryReq,
  ConversationSessionListReq,
  RetrievalObserveReq
} from '@/types'

export const chatApi = {
  // 流式聊天
  streamChat(params: ChatReq): Promise<Response<{ conversationId: string }>> {
    return axios.post('/chat/stream', params)
  },

  // 获取文档选项
  getDocumentOptions(): Promise<Response<DocumentOption[]>> {
    return axios.post('/chat/document/options')
  },

  // 停止会话
  stopConversation(params: ConversationIdentityReq): Promise<Response<ConversationStopResp>> {
    return axios.post('/chat/session/stop', params)
  },

  // 获取会话详情
  getSessionDetail(params: ConversationIdentityReq): Promise<Response<SessionDetail>> {
    return axios.post('/chat/session/detail', params)
  },

  // 获取会话列表
  listSessions(params?: ConversationSessionListReq): Promise<Response<PageResult<SessionListItem>>> {
    return axios.post('/chat/session/list', params || {})
  },

  // 重置会话
  resetConversation(params: ConversationIdentityReq): Promise<Response<ConversationResetResp>> {
    return axios.post('/chat/session/reset', params)
  },

  // 重建构建会话摘要
  rebuildSummary(params: ConversationIdentityReq): Promise<Response<ConversationResetResp>> {
    return axios.post('/chat/session/summary/rebuild', params)
  },

  // 获取会话详情
  getExchangeDetail(params: ConversationExchangeDetailQueryReq): Promise<Response<GetExchangeDetailResponse>> {
    return axios.post('/chat/exchange/detail', params)
  },

  // 获取检索结果
  getRetrievalResults(params: RetrievalObserveReq): Promise<Response<RetrievalResult[]>> {
    return axios.post('/chat/exchange/retrieval/results', params)
  },

  // 获取渠道执行记录
  getChannelExecutions(params: RetrievalObserveReq): Promise<Response<ChannelExecution[]>> {
    return axios.post('/chat/exchange/channel/executions', params)
  },
}
