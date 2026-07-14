import { defineStore } from 'pinia'
import { ref } from 'vue'
import { chatApi } from '@/api/chat'
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
    const res = await chatApi.getDocumentOptions()
    documentOptions.value = res
  }

  async function createSession(documentId?: number) {
    const res = await chatApi.streamChat({
      question: '',
      chatMode: 'document',
      selectedDocumentId: documentId,
    })
    return res.conversationId || ''
  }

  async function fetchSessionDetail(conversationId: string) {
    const res = await chatApi.getSessionDetail({ conversationId })
    currentSession.value = res
    exchanges.value = res.exchanges
    selectedDocumentId.value = res.selectedDocumentId
  }

  async function fetchSessions(params?: { chatMode?: string; keyword?: string; pageNo?: number; pageSize?: number }) {
    const res = await chatApi.listSessions(params)
    sessions.value = res.records
    total.value = res.totalSize
    pageNo.value = res.pageNo
    pageSize.value = res.pageSize
  }

  async function sendMessage(conversationId: string, question: string, documentId?: number) {
    isLoading.value = true
    try {
      const res = await chatApi.streamChat({
        question,
        conversationId,
        chatMode: 'document',
        selectedDocumentId: documentId,
      })
      await fetchSessionDetail(conversationId)
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
    createSession,
    fetchSessionDetail,
    fetchSessions,
    sendMessage,
    deleteSession,
    selectDocument,
  }
})
