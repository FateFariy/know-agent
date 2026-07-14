import axios from './axios'
import type { DocumentOption, SessionDetail, SessionListItem, PageResult, StageTrace, RetrievalResult, ChannelExecution, ConversationStopResp, ConversationResetResp, GetExchangeDetailResponse, ChatReq, ConversationIdentityReq, ConversationExchangeDetailQueryReq, ConversationSessionListReq, RetrievalObserveReq } from '@/types'

export const chatApi = {
  streamChat(params: ChatReq) {
    return axios.post('/chat/stream', params)
  },

  getDocumentOptions() {
    return axios.post<DocumentOption[]>('/chat/document/options')
  },

  stopConversation(params: ConversationIdentityReq) {
    return axios.post<ConversationStopResp>('/chat/session/stop', params)
  },

  getSessionDetail(params: ConversationIdentityReq) {
    return axios.post<SessionDetail>('/chat/session/detail', params)
  },

  listSessions(params?: ConversationSessionListReq) {
    return axios.post<PageResult<SessionListItem>>('/chat/session/list', params || {})
  },

  resetConversation(params: ConversationIdentityReq) {
    return axios.post<ConversationResetResp>('/chat/session/reset', params)
  },

  rebuildSummary(params: ConversationIdentityReq) {
    return axios.post<ConversationResetResp>('/chat/session/summary/rebuild', params)
  },

  getExchangeDetail(params: ConversationExchangeDetailQueryReq) {
    return axios.post<GetExchangeDetailResponse>('/chat/exchange/detail', params)
  },

  getRetrievalResults(params: RetrievalObserveReq) {
    return axios.post<RetrievalResult[]>('/chat/exchange/retrieval/results', params)
  },

  getChannelExecutions(params: RetrievalObserveReq) {
    return axios.post<ChannelExecution[]>('/chat/exchange/channel/executions', params)
  },
}