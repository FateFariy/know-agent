import { defineStore } from 'pinia'
import { ref } from 'vue'
import { documentApi } from '@/api/document'
import type { DocumentInfo, DocumentChunk, DocumentProfile, UploadFile } from '@/types'

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
    documents.value = res.records
    total.value = res.total
    pageNo.value = res.pageNo
    pageSize.value = res.pageSize
  }

  async function fetchDocumentDetail(documentId: number) {
    const res = await documentApi.queryDetail({ documentId })
    currentDocument.value = res
  }

  async function fetchChunks(documentId: number, params?: { pageNo?: number; pageSize?: number }) {
    const res = await documentApi.queryChunks({ documentId, ...params })
    chunks.value = res.records
  }

  async function fetchProfile(documentId: number) {
    const res = await documentApi.getProfile({ documentId })
    profile.value = res
  }

  async function deleteDocument(documentId: number) {
    await documentApi.deleteDocument({ documentId })
    await fetchDocuments()
  }

  async function uploadFile(file: File) {
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
      const res = await documentApi.uploadFile(file, (progress) => {
        uploadFile.progress = progress
      })
      uploadFile.status = 'success'
      uploadFile.documentId = res.data.documentId
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
