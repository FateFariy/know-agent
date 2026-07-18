import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { chatApi } from '@/api/chat'
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
}

interface StreamHandlers {
  onMeta?: (payload: { conversationId: string; taskId: string }) => void
  onMessage?: (payload: { delta: string }) => void
  onThinking?: (payload: { delta: string }) => void
  onFinish?: (payload: { messageId?: string; title?: string }) => void
  onCancel?: (payload: { messageId?: string; title?: string }) => void
  onError?: (error: Error) => void
}

const baseURL = (import.meta.env.MODE === 'production')
  ? '/api/v1/'
  : 'http://localhost:8080'

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
    exchangeId: exchange.exchangeId
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

  function cancelGeneration() {
    if (!isStreaming.value) return
    cancelRequested.value = true
    if (streamAbort.value) {
      streamAbort.value()
    }
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

    const conversationId = currentSessionId.value || undefined
    const req: ChatReq = {
      question: trimmed,
      conversationId,
      chatMode: 'auto_document'
    }

    const handlers: StreamHandlers = {
      onMeta: (payload) => {
        if (streamingMessageId.value !== assistantId) return
        const nextId = payload.conversationId || currentSessionId.value
        if (!nextId) return
        const existing = sessions.value.find((s) => s.id === nextId)
        const lastTime = new Date().toISOString()
        currentSessionId.value = nextId
        isCreatingNew.value = false
        if (!existing) {
          sessions.value = [
            {
              id: nextId,
              title: trimmed.slice(0, 24) || '新对话',
              lastTime,
              latestUserMessage: trimmed,
              running: true
            },
            ...sessions.value
          ]
        } else {
          sessions.value = sessions.value.map((s) =>
            s.id === nextId ? { ...s, lastTime, running: true, latestUserMessage: trimmed } : s
          )
        }
      },
      onMessage: (payload) => {
        if (!payload || typeof payload !== 'object') return
        appendStreamContent(payload.delta || '')
      },
      onThinking: (payload) => {
        if (!payload || typeof payload !== 'object') return
        appendThinkingContent(payload.delta || '')
      },
      onFinish: (payload) => {
        if (streamingMessageId.value !== assistantId) return
        const duration = computeThinkingDuration(thinkingStartAt.value)
        const finalId = payload?.messageId ? String(payload.messageId) : assistantId
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
        const cid = currentSessionId.value
        if (cid) {
          const lastTime = new Date().toISOString()
          const existing = sessions.value.find((s) => s.id === cid)
          const nextTitle = payload?.title || existing?.title || '新对话'
          sessions.value = sessions.value.map((s) =>
            s.id === cid ? { ...s, title: nextTitle, lastTime, running: false } : s
          )
        }
      },
      onCancel: (payload) => {
        if (streamingMessageId.value !== assistantId) return
        const duration = computeThinkingDuration(thinkingStartAt.value)
        const finalId = payload?.messageId ? String(payload.messageId) : assistantId
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
      }
    }

    try {
      const stop = await startStream(req, handlers)
      streamAbort.value = stop
    } catch (error) {
      if ((error as Error).name !== 'AbortError') {
        handlers.onError?.(error as Error)
      }
    } finally {
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
    // state
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
    // actions
    fetchSessions,
    createSession,
    selectSession,
    deleteSession,
    setDeepThinkingEnabled,
    sendMessage,
    cancelGeneration,
    submitFeedback
  }
})

/**
 * 通过 fetch + ReadableStream 解析 text/event-stream。
 * 支持事件类型：meta / message / think / finish / cancel / done。
 * 返回一个 stop 闭包用于外部主动中断。
 */
async function startStream(req: ChatReq, handlers: StreamHandlers): Promise<() => void> {
  const controller = new AbortController()
  let stopped = false

  const stop = () => {
    if (stopped) return
    stopped = true
    try {
      controller.abort()
    } catch {
      // ignore
    }
  }

  const url = `${baseURL}chat/stream`
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream'
    },
    body: JSON.stringify(req),
    signal: controller.signal
  })

  if (!response.ok || !response.body) {
    throw new Error(`流式连接失败 (${response.status})`)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder('utf-8')
  let buffer = ''

  // 异步消费流，不 await 整个流，方便外部 stop() 后立即打断 UI。
  ;(async () => {
    try {
      while (!stopped) {
        const { value, done } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        let boundary = buffer.indexOf('\n\n')
        while (boundary >= 0) {
          const rawEvent = buffer.slice(0, boundary)
          buffer = buffer.slice(boundary + 2)
          parseSseEvent(rawEvent, handlers)
          boundary = buffer.indexOf('\n\n')
        }
      }
    } catch (error) {
      if (!stopped) {
        handlers.onError?.(error as Error)
      }
    }
  })()

  return stop
}

interface SseEvent {
  event: string
  data: string
}

function parseSseEvent(rawBlock: string, handlers: StreamHandlers) {
  const lines = rawBlock.split('\n')
  const event: SseEvent = { event: 'message', data: '' }
  for (const line of lines) {
    if (!line) continue
    if (line.startsWith('event:')) {
      event.event = line.slice(6).trim()
    } else if (line.startsWith('data:')) {
      event.data += line.slice(5).trim()
    } else if (line.startsWith(':')) {
      // 注释行，忽略
    }
  }
  if (!event.data) return
  let payload: any
  try {
    payload = JSON.parse(event.data)
  } catch {
    payload = { raw: event.data }
  }
  switch (event.event) {
    case 'meta':
      handlers.onMeta?.(payload)
      break
    case 'message':
    case 'response':
      handlers.onMessage?.({ delta: payload?.delta ?? payload?.content ?? '' })
      break
    case 'think':
    case 'thinking':
      handlers.onThinking?.({ delta: payload?.delta ?? payload?.content ?? '' })
      break
    case 'finish':
    case 'done':
      handlers.onFinish?.(payload)
      break
    case 'cancel':
      handlers.onCancel?.(payload)
      break
    default:
      // 未知事件忽略
      break
  }
}
