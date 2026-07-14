import { defineStore } from 'pinia'
import { ref } from 'vue'
import { documentApi } from '@/api/document'
import type { DocumentInfo, DocumentChunk, DocumentProfile, UploadFile, UploadDocumentReq } from '@/types'

export const useDocumentStore = defineStore('document', () => {
  const documents = ref<DocumentInfo[]>([])
  const currentDocument = ref<DocumentInfo | null>(null)
  const chunks = ref<DocumentChunk[]>([])
  const profile = ref<DocumentProfile | null>(null)
  const uploadFiles = ref<UploadFile[]>([])
  const total = ref(0)
  const pageNo = ref(1)
  const pageSize = ref(10)

  async function fetchDocuments(params?: { keyword?: string; pageNo?: number; pageSize?: number }) {
    const res = await documentApi.queryPage(params)
    documents.value = res.data?.records || []
    total.value = res.data?.totalSize || 0
    pageNo.value = res.data?.pageNo || 1
    pageSize.value = res.data?.pageSize || 10
  }

  async function fetchDocumentDetail(documentId: number) {
    const res = await documentApi.queryDetail({ documentId })
    currentDocument.value = res.data || null
  }

  async function fetchChunks(documentId: number, params?: { pageNo?: number; pageSize?: number }) {
    const res = await documentApi.queryChunks({ documentId, ...params })
    chunks.value = res.data?.records || []
  }

  async function fetchProfile(documentId: number) {
    const res = await documentApi.getProfile({ documentId })
    profile.value = res.data || null
  }

  async function deleteDocument(documentId: number) {
    await documentApi.deleteDocument({ documentId })
    await fetchDocuments()
  }

  async function uploadFile(file: File, data?: UploadDocumentReq) {
    const uploadFile: UploadFile = {
      file,
      fileName: file.name,
      fileSize: file.size,
      progress: 0,
      status: 'pending',
    }
    uploadFiles.value.push(uploadFile)

    uploadFile.status = 'uploading'
    try {
      const res = await documentApi.uploadFile(file, data, (progress) => {
        uploadFile.progress = progress
      })
      uploadFile.status = 'success'
      uploadFile.documentId = res.data?.documentId || 0
      await fetchDocuments()
    } catch {
      uploadFile.status = 'error'
      uploadFile.errorMessage = '上传失败'
    }
  }

  function removeUploadFile(fileName: string) {
    uploadFiles.value = uploadFiles.value.filter((f) => f.fileName !== fileName)
  }

  return {
    documents,
    currentDocument,
    chunks,
    profile,
    uploadFiles,
    total,
    pageNo,
    pageSize,
    fetchDocuments,
    fetchDocumentDetail,
    fetchChunks,
    fetchProfile,
    deleteDocument,
    uploadFile,
    removeUploadFile,
  }
})
