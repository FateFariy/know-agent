import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { chatApi, type StreamHandlers } from '@/api/chat'
import type { SearchReference } from '@/types/chat'
import type {
  ChatReq,
  ConversationExchange,
  ConversationIdentityReq,
  ConversationSessionResp
} from '@/types'

/**
 * 内部会话摘要：UI 层只关心列表展示需要的字段，避免把后端完整 DTO 漏到组件里。
 */
export interface ChatSessionItem {
  id: string
  title: string
  lastTime?: string
  latestUserMessage?: string
  latestAssistantMessage?: string
  running?: boolean
}

/**
 * 内部统一消息结构：
 * 1) 历史会话来自 ConversationExchange；
 * 2) 流式会话来自增量事件（assistant-* 临时 id，流结束后回填真实 exchangeId）。
 */
export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  thinking?: string
  thinkingDuration?: number
  isDeepThinking?: boolean
  isThinking?: boolean
  feedback?: 'like' | 'dislike' | null
  status: 'streaming' | 'done' | 'cancelled' | 'error'
  createdAt?: string
  exchangeId?: string
  /** 当前阶段：'thinking' | 'retrieving' | 'generating' | 'done' | 'error' ... */
  stage?: string
  /** 检索引用：每次 reference 事件都会累加 */
  references?: SearchReference[]
  /** 后端推荐问题：finish 后出现 */
  recommendations?: string[]
}

function computeThinkingDuration(startAt?: number | null) {
  if (!startAt) return undefined
  const seconds = Math.round((Date.now() - startAt) / 1000)
  return Math.max(1, seconds)
}

function mapExchangeToMessages(exchange: ConversationExchange): ChatMessage[] {
  const createdAt = exchange.createTime || undefined
  const userMessage: ChatMessage = {
    id: `exchange-${exchange.exchangeId}-user`,
    role: 'user',
    content: exchange.question || '',
    status: 'done',
    createdAt,
    exchangeId: exchange.exchangeId
  }
  const assistantMessage: ChatMessage = {
    id: `exchange-${exchange.exchangeId}-assistant`,
    role: 'assistant',
    content: exchange.answer || '',
    thinking: Array.isArray(exchange.thinkingSteps) && exchange.thinkingSteps.length > 0
      ? exchange.thinkingSteps.join('\n\n')
      : undefined,
    thinkingDuration: undefined,
    isDeepThinking: Array.isArray(exchange.thinkingSteps) && exchange.thinkingSteps.length > 0,
    feedback: null,
    status: exchange.turnStatus === -1 ? 'error' : 'done',
    createdAt,
    exchangeId: exchange.exchangeId,
    references: exchange.references
  }
  return [userMessage, assistantMessage]
}

function toSessionItem(session: ConversationSessionResp): ChatSessionItem {
  return {
    id: session.conversationId,
    title: session.latestUserMessage?.slice(0, 24) || '新对话',
    lastTime: session.updatedTime,
    latestUserMessage: session.latestUserMessage,
    latestAssistantMessage: session.latestAssistantMessage,
    running: session.running
  }
}

export const useChatStore = defineStore('chat', () => {
  // ====== 状态 ======
  const sessions = ref<ChatSessionItem[]>([])
  const currentSessionId = ref<string | null>(null)
  const messages = ref<ChatMessage[]>([])
  const isLoading = ref(false)
  const sessionsLoaded = ref(false)
  const inputFocusKey = ref(0)
  const isStreaming = ref(false)
  const isCreatingNew = ref(false)
  const deepThinkingEnabled = ref(false)
  const thinkingStartAt = ref<number | null>(null)
  const streamAbort = ref<(() => void) | null>(null)
  const streamingMessageId = ref<string | null>(null)
  const cancelRequested = ref(false)

  // ====== 计算属性 ======
  const showWelcome = computed(() => messages.value.length === 0 && !isLoading.value)

  // ====== 会话管理 ======
  async function fetchSessions() {
    isLoading.value = true
    try {
      const res = await chatApi.listSessions({ pageNo: 1, pageSize: 50 })
      const records = res.data?.records || []
      sessions.value = records.map(toSessionItem).sort((a, b) => {
        const ta = a.lastTime ? new Date(a.lastTime).getTime() : 0
        const tb = b.lastTime ? new Date(b.lastTime).getTime() : 0
        return tb - ta
      })
    } catch {
      sessions.value = []
    } finally {
      isLoading.value = false
      sessionsLoaded.value = true
    }
  }

  /**
   * 创建一个"新会话占位"：仅清空前端状态，真正的 conversationId 由后端首次返回时回填。
   * 保持和 React 版 createSession 行为一致：返回当前会话 id（创建中为空）。
   */
  async function createSession(): Promise<string> {
    if (messages.value.length === 0 && !currentSessionId.value) {
      isCreatingNew.value = true
      isLoading.value = false
      thinkingStartAt.value = null
      deepThinkingEnabled.value = false
      return ''
    }
    if (isStreaming.value) {
      cancelGeneration()
    }
    currentSessionId.value = null
    messages.value = []
    isStreaming.value = false
    isLoading.value = false
    isCreatingNew.value = true
    deepThinkingEnabled.value = false
    thinkingStartAt.value = null
    streamAbort.value = null
    streamingMessageId.value = null
    cancelRequested.value = false
    return ''
  }

  async function selectSession(sessionId: string) {
    if (!sessionId) return
    if (currentSessionId.value === sessionId && messages.value.length > 0) return
    if (isStreaming.value) {
      cancelGeneration()
    }
    isLoading.value = true
    currentSessionId.value = sessionId
    isCreatingNew.value = false
    thinkingStartAt.value = null
    try {
      const res = await chatApi.getSessionDetail({ conversationId: sessionId })
      const detail = res.data
      if (currentSessionId.value !== sessionId) return
      const exchanges = detail?.exchanges || []
      messages.value = exchanges.flatMap(mapExchangeToMessages)
    } catch {
      messages.value = []
    } finally {
      if (currentSessionId.value === sessionId) {
        isLoading.value = false
        isStreaming.value = false
        streamingMessageId.value = null
        cancelRequested.value = false
      }
    }
  }

  async function deleteSession(sessionId: string) {
    try {
      const req: ConversationIdentityReq = { conversationId: sessionId }
      await chatApi.resetConversation(req)
      sessions.value = sessions.value.filter((s) => s.id !== sessionId)
      if (currentSessionId.value === sessionId) {
        messages.value = []
        currentSessionId.value = null
      }
    } catch {
      // 失败由 axios 拦截器统一提示
    }
    await fetchSessions().catch(() => null)
  }

  function setDeepThinkingEnabled(enabled: boolean) {
    deepThinkingEnabled.value = enabled
  }

  // ====== 流式响应辅助 ======
  function appendStreamContent(delta: string) {
    if (!delta) return
    const shouldFinalizeThinking = thinkingStartAt.value != null
    const duration = computeThinkingDuration(thinkingStartAt.value)
    if (shouldFinalizeThinking) {
      thinkingStartAt.value = null
    }
    messages.value = messages.value.map((m) => {
      if (m.id !== streamingMessageId.value) return m
      if (m.status === 'cancelled' || m.status === 'error') return m
      return {
        ...m,
        content: m.content + delta,
        isThinking: shouldFinalizeThinking ? false : m.isThinking,
        thinkingDuration: shouldFinalizeThinking && !m.thinkingDuration ? duration : m.thinkingDuration
      }
    })
  }

  function appendThinkingContent(delta: string) {
    if (!delta) return
    if (thinkingStartAt.value == null) {
      thinkingStartAt.value = Date.now()
    }
    messages.value = messages.value.map((m) => {
      if (m.id !== streamingMessageId.value) return m
      if (m.status === 'cancelled' || m.status === 'error') return m
      return {
        ...m,
        thinking: `${m.thinking ?? ''}${delta}`,
        isThinking: true
      }
    })
  }

  /** 累加 references */
  function appendReferences(items: SearchReference[]) {
    if (!items?.length) return
    messages.value = messages.value.map((m) => {
      if (m.id !== streamingMessageId.value) return m
      if (m.status === 'cancelled' || m.status === 'error') return m
      return { ...m, references: [...(m.references || []), ...items] }
    })
  }

  /** 覆盖式更新推荐问题（finish 时一次性下发）。 */
  function setRecommendations(items: string[]) {
    if (!items?.length) return
    messages.value = messages.value.map((m) => {
      if (m.id !== streamingMessageId.value) return m
      if (m.status === 'cancelled' || m.status === 'error') return m
      return { ...m, recommendations: items }
    })
  }

  /** 更新当前阶段。 */
  function setStage(stage: string) {
    messages.value = messages.value.map((m) => {
      if (m.id !== streamingMessageId.value) return m
      return { ...m, stage }
    })
  }

  function cancelGeneration() {
    if (!isStreaming.value) return
    const targetId = streamingMessageId.value
    const cid = currentSessionId.value
    cancelRequested.value = true

    // 1) 中止前端 SSE 连接
    if (streamAbort.value) {
      try {
        streamAbort.value()
      } catch {
        // ignore
      }
    }

    // 2) 通知后端真正停止生成（拿到 conversationId 时才调）
    if (cid) {
      chatApi.stopConversation({ conversationId: cid }).catch(() => null)
    }

    // 3) 本地立即把占位消息标记为 cancelled，清理流式状态
    //    后端稍后即便发来 cancel/finish 事件，下面的 onCancel/onFinish 也会因
    //    streamingMessageId 已置空 + status==='cancelled' 的双重判断而成为 no-op。
    if (targetId) {
      const duration = computeThinkingDuration(thinkingStartAt.value)
      messages.value = messages.value.map((m) => {
        if (m.id !== targetId) return m
        if (m.status === 'cancelled' || m.status === 'error') return m
        const suffix = m.content.includes('（已停止生成）') ? '' : '\n\n（已停止生成）'
        return {
          ...m,
          content: m.content + suffix,
          status: 'cancelled',
          isThinking: false,
          thinkingDuration: m.thinkingDuration ?? duration
        }
      })
      if (cid) {
        const lastTime = new Date().toISOString()
        sessions.value = sessions.value.map((s) =>
          s.id === cid ? { ...s, lastTime, running: false } : s
        )
      }
    }
    isStreaming.value = false
    thinkingStartAt.value = null
    streamAbort.value = null
    streamingMessageId.value = null
    cancelRequested.value = false
  }

  /**
   * 发送消息并通过 SSE 增量更新。
   * 后端约定：POST /chat/stream 返回 text/event-stream，事件类型包括 meta/message/think/finish/cancel。
   */
  async function sendMessage(content: string) {
    const trimmed = content.trim()
    if (!trimmed) return
    if (isStreaming.value) return

    const deepOn = deepThinkingEnabled.value
    inputFocusKey.value = Date.now()

    const userMessage: ChatMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: trimmed,
      status: 'done',
      createdAt: new Date().toISOString()
    }
    const assistantId = `assistant-${Date.now()}`
    const assistantMessage: ChatMessage = {
      id: assistantId,
      role: 'assistant',
      content: '',
      thinking: deepOn ? '' : undefined,
      isDeepThinking: deepOn,
      isThinking: deepOn,
      status: 'streaming',
      feedback: null,
      createdAt: new Date().toISOString()
    }

    messages.value = [...messages.value, userMessage, assistantMessage]
    isStreaming.value = true
    streamingMessageId.value = assistantId
    thinkingStartAt.value = null
    cancelRequested.value = false

    if (!currentSessionId.value) {
      currentSessionId.value = crypto.randomUUID().replace(/-/g, '')
    }
    if (!sessions.value.find((s) => s.id === currentSessionId.value)) {
      sessions.value = [{ id: currentSessionId.value, title: content.slice(0, 24) || '新对话', lastTime: new Date().toISOString(), running: true }, ...sessions.value]
    }

    const conversationId = currentSessionId.value
    const req: ChatReq = {
      question: trimmed,
      conversationId,
      chatMode: 'auto_document'
    }

    const handlers: StreamHandlers = {
      onText: (payload) => {
        appendStreamContent(payload.content)
      },
      onThinking: (payload) => {
        appendThinkingContent(payload.content)
      },
      onStatus: (payload) => {
        setStage(payload.stage)
      },
      onReference: (payload) => {
        appendReferences(payload.items)
      },
      onRecommend: (payload) => {
        setRecommendations(payload.items)
      },
      onFinish: (payload) => {
        if (streamingMessageId.value !== assistantId) return
        const duration = computeThinkingDuration(thinkingStartAt.value)
        const finalId = payload.messageId ? String(payload.messageId) : assistantId
        messages.value = messages.value.map((m) => {
          if (m.id !== assistantId) return m
          return {
            ...m,
            id: finalId,
            status: 'done',
            isThinking: false,
            thinkingDuration: m.thinkingDuration ?? duration
          }
        })
        streamingMessageId.value = finalId
        const cid = currentSessionId.value
        if (cid) {
          const lastTime = new Date().toISOString()
          const existing = sessions.value.find((s) => s.id === cid)
          const nextTitle = payload.title || existing?.title || '新对话'
          sessions.value = sessions.value.map((s) =>
            s.id === cid ? { ...s, title: nextTitle, lastTime, running: false } : s
          )
        }
      },
      onCancel: (payload) => {
        if (streamingMessageId.value !== assistantId) return
        const duration = computeThinkingDuration(thinkingStartAt.value)
        const finalId = payload.messageId ? String(payload.messageId) : assistantId
        messages.value = messages.value.map((m) => {
          if (m.id !== assistantId) return m
          const suffix = m.content.includes('（已停止生成）') ? '' : '\n\n（已停止生成）'
          return {
            ...m,
            id: finalId,
            content: m.content + suffix,
            status: 'cancelled',
            isThinking: false,
            thinkingDuration: m.thinkingDuration ?? duration
          }
        })
        isStreaming.value = false
        thinkingStartAt.value = null
        streamAbort.value = null
        streamingMessageId.value = null
        cancelRequested.value = false
      },
      onError: (error) => {
        // 后端连通失败或 SSE 解析异常时，给用户可见提示，并清掉流式占位。
        console.error('[chat] stream error', error)
        if (streamingMessageId.value === assistantId) {
          const duration = computeThinkingDuration(thinkingStartAt.value)
          messages.value = messages.value.map((m) => {
            if (m.id !== assistantId) return m
            return {
              ...m,
              status: 'error',
              isThinking: false,
              thinkingDuration: m.thinkingDuration ?? duration
            }
          })
          isStreaming.value = false
          thinkingStartAt.value = null
          streamAbort.value = null
          streamingMessageId.value = null
          cancelRequested.value = false
        }
        ElMessage.error(`对话请求失败：${error?.message || '未知错误'}`)
      },
      onClose: () => {
        if (!streamingMessageId.value) return
        const duration = computeThinkingDuration(thinkingStartAt.value)
        const targetId = streamingMessageId.value
        messages.value = messages.value.map((m) => {
          if (m.id !== targetId) return m
          if (m.status !== 'streaming') return m
          return { ...m, status: 'done', isThinking: false, thinkingDuration: m.thinkingDuration ?? duration }
        })
        const cid = currentSessionId.value
        if (cid) {
          const lastTime = new Date().toISOString()
          sessions.value = sessions.value.map((s) =>
            s.id === cid ? { ...s, lastTime, running: false } : s
          )
        }
        isStreaming.value = false
        thinkingStartAt.value = null
        streamAbort.value = null
        streamingMessageId.value = null
        cancelRequested.value = false
      }
    }

    try {
      // 流式连接的所有错误（包括初始化失败）都由 handlers.onError 通知，
      // 此处只负责把 stop 闭包交给 streamAbort 供 cancelGeneration 使用。
      const { stop } = chatApi.streamChat(req, handlers)
      streamAbort.value = stop
    } catch (error) {
      // 兜底：理论上 streamChat 不会同步 throw；若出现异常仍清理状态。
      console.error('[chat] streamChat failed', error)
      if (streamingMessageId.value === assistantId) {
        isStreaming.value = false
        thinkingStartAt.value = null
        streamAbort.value = null
        streamingMessageId.value = null
        cancelRequested.value = false
      }
    }
  }

  /**
   * 简单的反馈记录：仅在前端维护 vote 状态，真实接口不存在时静默失败。
   */
  async function submitFeedback(messageId: string, feedback: 'like' | 'dislike' | null) {
    const prev = messages.value.find((m) => m.id === messageId)?.feedback ?? null
    messages.value = messages.value.map((m) =>
      m.id === messageId ? { ...m, feedback } : m
    )
    if (feedback === null) {
      return
    }
    // 当前后端没有提供反馈接口，预留位即可；失败时不回滚。
    void prev
  }

  return {
    sessions,
    currentSessionId,
    messages,
    isLoading,
    sessionsLoaded,
    inputFocusKey,
    isStreaming,
    isCreatingNew,
    deepThinkingEnabled,
    showWelcome,
    fetchSessions,
    createSession,
    selectSession,
    deleteSession,
    setDeepThinkingEnabled,
    sendMessage,
    cancelGeneration,
    submitFeedback
  }
}, {
  // 刷新页面后仅保留当前会话 id，messages 仍走 selectSession 从后端重新拉取
  // 存储用 sessionStorage：标签页内生效，关闭后失效，避免跨标签污染会话上下文
  persist: {
    key: 'chat:currentSessionId',
    storage: sessionStorage,
    paths: ['currentSessionId']
  }
})
