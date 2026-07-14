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
  streamChat(params: ChatReq): Promise<Response<void>> {
    return axios.post('/chat/stream', params)
  },

  getDocumentOptions(): Promise<Response<DocumentOption[]>> {
    return axios.post('/chat/document/options')
  },

  stopConversation(params: ConversationIdentityReq): Promise<Response<ConversationStopResp>> {
    return axios.post('/chat/session/stop', params)
  },

  getSessionDetail(params: ConversationIdentityReq): Promise<Response<SessionDetail>> {
    return axios.post('/chat/session/detail', params)
  },

  listSessions(params?: ConversationSessionListReq): Promise<Response<PageResult<SessionListItem>>> {
    return axios.post('/chat/session/list', params || {})
  },

  resetConversation(params: ConversationIdentityReq): Promise<Response<ConversationResetResp>> {
    return axios.post('/chat/session/reset', params)
  },

  rebuildSummary(params: ConversationIdentityReq): Promise<Response<ConversationResetResp>> {
    return axios.post('/chat/session/summary/rebuild', params)
  },

  getExchangeDetail(params: ConversationExchangeDetailQueryReq): Promise<Response<GetExchangeDetailResponse>> {
    return axios.post('/chat/exchange/detail', params)
  },

  getRetrievalResults(params: RetrievalObserveReq): Promise<Response<RetrievalResult[]>> {
    return axios.post('/chat/exchange/retrieval/results', params)
  },

  getChannelExecutions(params: RetrievalObserveReq): Promise<Response<ChannelExecution[]>> {
    return axios.post('/chat/exchange/channel/executions', params)
  },
}
