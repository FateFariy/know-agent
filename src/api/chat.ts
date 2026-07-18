import axios, { baseURL } from './axios'
import type {
  Response,
  ConversationSessionResp,
  ConversationSessionListResp,
  RetrievalResultResp,
  ChannelExecutionResp,
  ConversationStopResp,
  ConversationResetResp,
  ConversationMemorySummaryResp,
  ConversationExchangeDetailResp,
  ChatReq,
  ConversationIdentityReq,
  ConversationExchangeDetailQueryReq,
  ConversationSessionListReq,
  RetrievalObserveReq,
  SearchReference,
} from '@/types'

/** SSE 流式回调：后端约定事件类型 meta / text / thinking / status / reference / recommend / finish / cancel。 */
export interface StreamHandlers {
  onMeta?: (payload: { conversationId?: string; taskId?: string; exchangeId?: string | number }) => void
  /** 普通文本增量（与 onMessage 同义，保留 onMessage 用于旧后端协议） */
  onText?: (payload: { delta: string }) => void
  onMessage?: (payload: { delta: string }) => void
  /** 思考阶段增量：前端要把它实时拼到消息的 thinking 字段。 */
  onThinking?: (payload: { delta: string }) => void
  /** 阶段状态：例如"检索文档中…"，可用于在 UI 提示。 */
  onStatus?: (payload: { stage: string; message?: string }) => void
  /** 引用来源（支持多次累加） */
  onReference?: (payload: { items: SearchReference[] }) => void
  /** 推荐问题：渲染为按钮，点击可发送。 */
  onRecommend?: (payload: { items: string[] }) => void
  onFinish?: (payload: { messageId?: string; title?: string }) => void
  onCancel?: (payload: { messageId?: string; title?: string }) => void
  onError?: (error: Error) => void
}

interface SseEvent {
  event: string
  data: string
}

interface StreamEnvet {
  type: string
  content: any
  timestamp: string
  conversationId: string
  exchangeId: number
  count: number
}

export const chatApi = {
  /**
   * 开启 SSE 流式对话。
   * - 内部走 fetch + ReadableStream，绕过 axios（axios 不支持原生 SSE）。
   * - 返回 { stop } 闭包，调用即主动中断前端读取并通过 AbortController 关闭连接。
   * - 任何阶段抛错（网络、CORS、非 2xx、解析失败）都会通过 handlers.onError 通知；非主动停止时调用 onError。
   */
  streamChat(req: ChatReq, handlers: StreamHandlers): { stop: () => void } {
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

    const url = `${baseURL}/chat/stream`
    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream'
      },
      body: JSON.stringify(req),
      signal: controller.signal
    })
      .then(async (response) => {
        if (!response.ok || !response.body) {
          throw new Error(`流式连接失败 (${response.status})`)
        }
        const reader = response.body.getReader()
        const decoder = new TextDecoder('utf-8')
        let buffer = ''
        while (!stopped) {
          const { value, done } = await reader.read()
          if (done) break
          buffer += decoder.decode(value, { stream: true })
          // 持续从 buffer 中提取事件,支持 SSE(\n\n) 与裸 JSON({...}) 两种格式
          while (true) {
            const boundary = findEventBoundary(buffer)
            if (!boundary) break
            const rawEvent = buffer.slice(0, boundary.end)
            buffer = buffer.slice(boundary.end)
            parseSseEvent(rawEvent, handlers)
          }
        }
      })
      .catch((error) => {
        // 主动 abort 触发的 AbortError 不视为错误：调用方已自行处理状态。
        if (!stopped) {
          handlers.onError?.(error instanceof Error ? error : new Error(String(error)))
        }
      })

    return { stop }
  },

  // 停止会话
  stopConversation(params: ConversationIdentityReq): Promise<Response<ConversationStopResp>> {
    return axios.post('/chat/session/stop', params)
  },

  // 获取会话详情
  getSessionDetail(params: ConversationIdentityReq): Promise<Response<ConversationSessionResp>> {
    return axios.post('/chat/session/detail', params)
  },

  // 获取会话列表
  listSessions(params?: ConversationSessionListReq): Promise<Response<ConversationSessionListResp>> {
    return axios.post('/chat/session/list', params || {})
  },

  // 重置会话
  resetConversation(params: ConversationIdentityReq): Promise<Response<ConversationResetResp>> {
    return axios.post('/chat/session/reset', params)
  },

  // 重建构建会话摘要
  rebuildSummary(params: ConversationIdentityReq): Promise<Response<ConversationMemorySummaryResp>> {
    return axios.post('/chat/session/summary/rebuild', params)
  },

  // 获取会话详情
  getExchangeDetail(params: ConversationExchangeDetailQueryReq): Promise<Response<ConversationExchangeDetailResp>> {
    return axios.post('/chat/exchange/detail', params)
  },

  // 获取检索结果
  getRetrievalResults(params: RetrievalObserveReq): Promise<Response<RetrievalResultResp[]>> {
    return axios.post('/chat/exchange/retrieval/results', params)
  },

  // 获取渠道执行记录
  getChannelExecutions(params: RetrievalObserveReq): Promise<Response<ChannelExecutionResp[]>> {
    return axios.post('/chat/exchange/channel/executions', params)
  },
}

/** 解析一段 SSE 原始块并派发到对应 handler。模块作用域私有工具。
 * 支持两种输入格式：
 * 1) SSE 标准格式：`event: xxx\ndata: {...}\n\n`（后端老协议）
 * 2) 裸 JSON 格式：单行/多行 JSON 对象，后端 streamEvent 结构体直接序列化的形式
 */
function parseSseEvent(rawBlock: string, handlers: StreamHandlers) {
  const trimmed = rawBlock.trim()
  if (!trimmed) return

  // 路径 1：整块是裸 JSON（以 { 开头），后端 streamEvent 直接序列化的形式
  if (trimmed.startsWith('{')) {
    let payload: StreamEnvet
    payload = JSON.parse(trimmed)
    dispatchPayload(payload, handlers)
    return
  }

  // 路径 2：SSE 标准格式，逐行解析 event / data 字段
  const lines = rawBlock.split('\n')
  const event: SseEvent = { event: 'message', data: '' }
  let sawStructuredLine = false
  for (const line of lines) {
    if (!line) continue
    if (line.startsWith('event:')) {
      event.event = line.slice(6).trim()
      sawStructuredLine = true
    } else if (line.startsWith('data:')) {
      event.data += line.slice(5).trim()
      sawStructuredLine = true
    } else if (line.startsWith(':')) {
      // 注释行，忽略
      sawStructuredLine = true
    } else if (line.startsWith('{')) {
      // 容错：某些实现会把 data 后的 JSON 写在新行，合并到 data 字段
      event.data += (event.data ? '\n' : '') + line
      sawStructuredLine = true
    }
  }
  if (!event.data) return
  if (!sawStructuredLine) {
    // 整块是纯文本，当作一个无类型的 fallback
    dispatchPayload({ raw: event.data }, handlers)
    return
  }
  let payload: any
  try {
    payload = JSON.parse(event.data)
  } catch {
    payload = { raw: event.data }
  }
  dispatchPayload(payload, handlers)
}

/** 根据 payload.type 派发到对应 handler。模块作用域私有工具。 */
function dispatchPayload(payload: any, handlers: StreamHandlers) {
  // 兼容两种后端实现：SSE event 字段 OR payload.type 字段
  const type = (payload?.type || '').toString().toLowerCase()
  // 字段兼容：content 优先（后端新协议），fallback 到 delta（旧协议）
  const delta = payload?.content ?? payload?.delta ?? ''
  // 后端 reference / recommend 事件把数组放在 content 字段，旧协议用 items
  const rawItems = Array.isArray(payload?.items)
    ? payload.items
    : Array.isArray(payload?.content)
      ? payload.content
      : []
  switch (type) {
    case 'meta':
      handlers.onMeta?.(payload)
      break
    case 'text':
    case 'message':
    case 'response':
      handlers.onText?.({ delta })
      break
    case 'thinking':
    case 'think':
      handlers.onThinking?.({ delta })
      break
    case 'status':
      handlers.onStatus?.({
        stage: payload?.stage ?? 'pending',
        message: delta || payload?.message
      })
      break
    case 'reference':
      handlers.onReference?.({ items: rawItems })
      break
    case 'recommend':
      handlers.onRecommend?.({ items: rawItems.filter((x: any) => typeof x === 'string') })
      break
    case 'finish':
    case 'done':
      handlers.onFinish?.(payload)
      break
    case 'cancel':
      handlers.onCancel?.(payload)
      break
    case 'error':
      // 后端 error 事件：作为可读消息上抛，业务层可在 onText 或 onError 中处理
      handlers.onText?.({ delta: delta || payload?.message || '生成出错' })
      handlers.onError?.(new Error(delta || payload?.message || '生成出错'))
      break
    default:
      // 未知事件忽略
      break
  }
}

/** 在 buffer 中寻找下一个事件结束位置。
 * 优先匹配 SSE 的 \n\n 分隔；若不存在，按大括号配对定位首个完整 JSON 对象。
 * 返回 { end } 给出该事件在 buffer 中应被切走的字符下标（含分隔符），找不到则返回 null。
 */
function findEventBoundary(buffer: string): { end: number } | null {
  // 1) SSE：\n\n 分隔
  const sse = buffer.indexOf('\n\n')
  if (sse >= 0) {
    return { end: sse + 2 }
  }

  // 2) 裸 JSON：跳过前导空白后，按大括号配对定位首个完整 JSON 对象
  let i = 0
  while (i < buffer.length && /\s/.test(buffer[i] || '')) i++
  if (buffer[i] !== '{') return null

  let depth = 0
  let inString = false
  let escaped = false
  for (; i < buffer.length; i++) {
    const c = buffer[i]
    if (inString) {
      if (escaped) {
        escaped = false
      } else if (c === '\\') {
        escaped = true
      } else if (c === '"') {
        inString = false
      }
      continue
    }
    if (c === '"') {
      inString = true
      continue
    }
    if (c === '{') {
      depth++
    } else if (c === '}') {
      depth--
      if (depth === 0) {
        return { end: i + 1 }
      }
    }
  }
  return null
}
