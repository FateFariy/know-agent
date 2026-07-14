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

    return axios.post('/manage/document/upload', formData, {
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

  queryPage(params?: QueryDocumentPageReq): Promise<Response<PageResult<DocumentInfo>>> {
    return axios.post('/manage/document/page/query', params || {})
  },

  queryDetail(params: QueryDocumentDetailReq): Promise<Response<DocumentInfo>> {
    return axios.post('/manage/document/detail/query', params)
  },

  deleteDocument(params: DeleteDocumentReq): Promise<Response<DeleteDocumentResp>> {
    return axios.post('/manage/document/delete', params)
  },

  queryStrategyPlan(params: QueryStrategyPlanReq): Promise<Response<QueryStrategyPlanResp>> {
    return axios.post('/manage/document/strategy/plan/query', params)
  },

  confirmStrategy(params: ConfirmStrategyReq): Promise<Response<ConfirmStrategyResp>> {
    return axios.post('/manage/document/strategy/confirm', params)
  },

  buildIndex(params: BuildIndexReq): Promise<Response<BuildIndexResp>> {
    return axios.post('/manage/document/index/build', params)
  },

  queryChunks(params: DocChunksReq): Promise<Response<QueryDocumentChunksResp>> {
    return axios.post('/manage/document/chunk/query', params)
  },

  queryChunkDetail(params: DocChunkDetailReq): Promise<Response<QueryDocumentChunkDetailResp>> {
    return axios.post('/manage/document/chunk/detail/query', params)
  },

  queryTaskLogs(params: QueryTaskLogsReq): Promise<Response<QueryTaskLogsResp>> {
    return axios.post('/manage/document/task/log/query', params)
  },

  getProfile(params: DocumentProfileDetailReq): Promise<Response<DocumentProfile>> {
    return axios.post('/manage/document/profile/detail', params)
  },

  regenerateProfile(params: DocumentProfileRegenerateReq): Promise<Response<DocumentProfile>> {
    return axios.post('/manage/document/profile/regenerate', params)
  },

  batchRegenerateProfile(params: DocumentProfileBatchRegenerateReq): Promise<Response<DocumentProfile[]>> {
    return axios.post('/manage/document/profile/batch/regenerate', params)
  },
}
