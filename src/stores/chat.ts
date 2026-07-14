import { defineStore } from 'pinia'
import { ref } from 'vue'
import { chatApi } from '@/api/chat'
import { documentApi } from '@/api/document'
import type { SessionDetail, SessionListItem, Exchange, DocumentOption } from '@/types'

export const useChatStore = defineStore('chat', () => {
  const sessions = ref<SessionListItem[]>([])
  const currentSession = ref<SessionDetail | null>(null)
  const exchanges = ref<Exchange[]>([])
  const documentOptions = ref<DocumentOption[]>([])
  const selectedDocumentId = ref<number | null>(null)
  const isLoading = ref(false)
  const total = ref(0)
  const pageNo = ref(1)
  const pageSize = ref(10)

  async function fetchDocumentOptions() {
    const res = await documentApi.getDocumentOptions()
    documentOptions.value = res.data?.map((doc) => ({
      documentId: doc.documentId,
      documentName: doc.documentName,
      knowledgeScopeName: doc.knowledgeScopeName,
      businessCategory: doc.businessCategory,
      documentTags: doc.documentTags,
    })) || []
  }

  // async function createSession(documentId?: number) {
  //   const res = await chatApi.streamChat({
  //     question: '',
  //     chatMode: 'document',
  //     selectedDocumentId: documentId,
  //   })
  //   return res.conversationId || ''
  // }

  async function fetchSessionDetail(conversationId: string) {
    const res = await chatApi.getSessionDetail({ conversationId })
    currentSession.value = res.data || null
    exchanges.value = res.data?.exchanges || []
    selectedDocumentId.value = res.data?.selectedDocumentId || null
  }

  async function fetchSessions(params?: { chatMode?: string; keyword?: string; pageNo?: number; pageSize?: number }) {
    const res = await chatApi.listSessions(params)
    sessions.value = res.data?.records || []
    total.value = res.data?.totalSize || 0
    pageNo.value = res.data?.pageNo || 1
    pageSize.value = res.data?.pageSize || 10
  }

  async function sendMessage(conversationId: string, question: string, documentId?: number) {
    isLoading.value = true
    try {
      const res = await chatApi.streamChat({
        question,
        conversationId: conversationId || undefined,
        chatMode: 'document',
        selectedDocumentId: documentId,
      })
      if (conversationId) {
        await fetchSessionDetail(conversationId)
      }
      return res
    } finally {
      isLoading.value = false
    }
  }

  async function deleteSession(conversationId: string) {
    await chatApi.resetConversation({ conversationId })
    await fetchSessions()
  }

  function selectDocument(documentId: number | null) {
    selectedDocumentId.value = documentId
  }

  return {
    sessions,
    currentSession,
    exchanges,
    documentOptions,
    selectedDocumentId,
    isLoading,
    total,
    pageNo,
    pageSize,
    fetchDocumentOptions,
    // createSession,
    fetchSessionDetail,
    fetchSessions,
    sendMessage,
    deleteSession,
    selectDocument,
  }
})
