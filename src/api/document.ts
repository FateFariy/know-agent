import axios from './axios'
import type { Response, DocumentInfo, DocumentProfile, PageResult, UploadDocumentResp, BuildIndexResp, QueryStrategyPlanResp, ConfirmStrategyResp, QueryDocumentChunksResp, QueryDocumentChunkDetailResp, QueryTaskLogsResp, DeleteDocumentResp, UploadDocumentReq, QueryDocumentPageReq, QueryDocumentDetailReq, DeleteDocumentReq, QueryStrategyPlanReq, ConfirmStrategyReq, BuildIndexReq, QueryDocumentChunksReq as DocChunksReq, QueryDocumentChunkDetailReq as DocChunkDetailReq, QueryTaskLogsReq, DocumentProfileDetailReq, DocumentProfileRegenerateReq, DocumentProfileBatchRegenerateReq } from '@/types'

export const documentApi = {
  // 上传文档（带完整表单数据）
  uploadDocument(formData: FormData): Promise<Response<UploadDocumentResp>> {
    return axios.post('/manage/document/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
  },

  // 上传文件
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

  // 查询文档分页
  queryPage(params?: QueryDocumentPageReq): Promise<Response<PageResult<DocumentInfo>>> {
    return axios.post('/manage/document/page/query', params || {})
  },

  // 查询文档详情
  queryDetail(params: QueryDocumentDetailReq): Promise<Response<DocumentInfo>> {
    return axios.post('/manage/document/detail/query', params)
  },

  // 删除文档
  deleteDocument(params: DeleteDocumentReq): Promise<Response<DeleteDocumentResp>> {
    return axios.post('/manage/document/delete', params)
  },

  // 查询策略计划
  queryStrategyPlan(params: QueryStrategyPlanReq): Promise<Response<QueryStrategyPlanResp>> {
    return axios.post('/manage/document/strategy/plan/query', params)
  },

  // 确认策略计划
  confirmStrategy(params: ConfirmStrategyReq): Promise<Response<ConfirmStrategyResp>> {
    return axios.post('/manage/document/strategy/confirm', params)
  },

  // 构建索引
  buildIndex(params: BuildIndexReq): Promise<Response<BuildIndexResp>> {
    return axios.post('/manage/document/index/build', params)
  },

  // 查询文档分块
  queryChunks(params: DocChunksReq): Promise<Response<QueryDocumentChunksResp>> {
    return axios.post('/manage/document/chunk/query', params)
  },

  // 查询文档分块详情
  queryChunkDetail(params: DocChunkDetailReq): Promise<Response<QueryDocumentChunkDetailResp>> {
    return axios.post('/manage/document/chunk/detail/query', params)
  },

  // 查询任务日志
  queryTaskLogs(params: QueryTaskLogsReq): Promise<Response<QueryTaskLogsResp>> {
    return axios.post('/manage/document/task/log/query', params)
  },

  // 查询文档配置
  getProfile(params: DocumentProfileDetailReq): Promise<Response<DocumentProfile>> {
    return axios.post('/manage/document/profile/detail', params)
  },

  // 重新生成文档配置
  regenerateProfile(params: DocumentProfileRegenerateReq): Promise<Response<DocumentProfile>> {
    return axios.post('/manage/document/profile/regenerate', params)
  },

  // 批量重新生成文档配置
  batchRegenerateProfile(params: DocumentProfileBatchRegenerateReq): Promise<Response<DocumentProfile[]>> {
    return axios.post('/manage/document/profile/batch/regenerate', params)
  },
}
