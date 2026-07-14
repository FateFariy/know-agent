import axios from './axios'
import type { Response, DocumentInfo, DocumentProfile, PageResult, UploadDocumentResp, BuildIndexResp, QueryStrategyPlanResp, ConfirmStrategyResp, QueryDocumentChunksResp, QueryDocumentChunkDetailResp, QueryTaskLogsResp, DeleteDocumentResp, UploadDocumentReq, QueryDocumentPageReq, QueryDocumentDetailReq, DeleteDocumentReq, QueryStrategyPlanReq, ConfirmStrategyReq, BuildIndexReq, QueryDocumentChunksReq as DocChunksReq, QueryDocumentChunkDetailReq as DocChunkDetailReq, QueryTaskLogsReq, DocumentProfileDetailReq, DocumentProfileRegenerateReq, DocumentProfileBatchRegenerateReq } from '@/types'

export const documentApi = {
  uploadFile(file: File, data?: UploadDocumentReq, onProgress?: (progress: number) => void): Promise<Response<UploadDocumentResp>> {
    const formData = new FormData()
    formData.append('file', file)
    if (data) {
      Object.keys(data).forEach((key) => {
        const value = data[key as keyof UploadDocumentReq]
        if (value !== undefined) {
          formData.append(key, String(value))
        }
      })
    }

    return axios.post<UploadDocumentResp>('/manage/document/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        if (onProgress && progressEvent.total) {
          const progress = Math.round((progressEvent.loaded / progressEvent.total) * 100)
          onProgress(progress)
        }
      },
    })
  },

  queryPage(params?: QueryDocumentPageReq) {
    return axios.post<PageResult<DocumentInfo>>('/manage/document/page/query', params || {})
  },

  queryDetail(params: QueryDocumentDetailReq) {
    return axios.post<DocumentInfo>('/manage/document/detail/query', params)
  },

  deleteDocument(params: DeleteDocumentReq) {
    return axios.post<DeleteDocumentResp>('/manage/document/delete', params)
  },

  queryStrategyPlan(params: QueryStrategyPlanReq) {
    return axios.post<QueryStrategyPlanResp>('/manage/document/strategy/plan/query', params)
  },

  confirmStrategy(params: ConfirmStrategyReq) {
    return axios.post<ConfirmStrategyResp>('/manage/document/strategy/confirm', params)
  },

  buildIndex(params: BuildIndexReq) {
    return axios.post<BuildIndexResp>('/manage/document/index/build', params)
  },

  queryChunks(params: DocChunksReq) {
    return axios.post<QueryDocumentChunksResp>('/manage/document/chunk/query', params)
  },

  queryChunkDetail(params: DocChunkDetailReq) {
    return axios.post<QueryDocumentChunkDetailResp>('/manage/document/chunk/detail/query', params)
  },

  queryTaskLogs(params: QueryTaskLogsReq) {
    return axios.post<QueryTaskLogsResp>('/manage/document/task/log/query', params)
  },

  getProfile(params: DocumentProfileDetailReq) {
    return axios.post<DocumentProfile>('/manage/document/profile/detail', params)
  },

  regenerateProfile(params: DocumentProfileRegenerateReq) {
    return axios.post<DocumentProfile>('/manage/document/profile/regenerate', params)
  },

  batchRegenerateProfile(params: DocumentProfileBatchRegenerateReq) {
    return axios.post<DocumentProfile[]>('/manage/document/profile/batch/regenerate', params)
  },
}
