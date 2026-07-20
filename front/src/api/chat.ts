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

/** 后端约定的流式事件类型，对应 streamEvent.type 字段。 */
export type StreamEventType =
  | 'text'
  | 'thinking'
  | 'status'
  | 'reference'
  | 'recommend'
  | 'finish'
  | 'cancel'
  | 'error'

/** 已知事件类型集合（用于类型守卫 / 兜底校验）。 */
const STREAM_EVENT_TYPES: ReadonlySet<StreamEventType> = new Set<StreamEventType>([
  'text',
  'thinking',
  'status',
  'reference',
  'recommend',
  'finish',
  'cancel',
  'error'
])

/** 单条流式事件（对应后端 streamEvent 结构体序列化结果）。
 * 后端只通过两种线上格式下发：
 *  1) SSE 标准：`event: <type>\ndata: <此 JSON>\n\n`
 *  2) 裸 JSON：直接以 `{...}` 形式发送
 * 解析后归一化为本结构派发到 handlers，不再区分新老协议字段。
 */
export interface StreamEvent {
  type: StreamEventType
  /** 不同事件下语义不同：
   *  - text / thinking / status / finish / cancel / error：string
   *  - reference：SearchReference[]
   *  - recommend：string[]
   */
  content?: string | SearchReference[] | string[]
  timestamp?: string
  conversationId?: string
  exchangeId?: string
  count?: number
  /** status 事件专用：阶段标识。 */
  stage?: string
  /** finish / cancel 事件专用：对话标题。 */
  title?: string
}

/** SSE 帧的解析中间结构（私有，parseSseFrame 内部使用）。 */
interface SseEvent {
  event?: string
  data: string
}

/** 派发到业务层的标准化 payload。已统一字段命名，不再做新老协议兼容。 */
export interface StreamHandlers {
  onText?: (payload: { content: string }) => void
  onThinking?: (payload: { content: string }) => void
  onStatus?: (payload: { stage: string; content?: string }) => void
  onReference?: (payload: { items: SearchReference[]; count?: number }) => void
  onRecommend?: (payload: { items: string[]; count?: number }) => void
  onFinish?: (payload: { messageId?: string; title?: string }) => void
  onCancel?: (payload: { messageId?: string; title?: string }) => void
  onError?: (error: Error) => void
  /** SSE 连接已关闭（非用户主动 stop），用于后端没发 finish/cancel 就关流的兜底。 */
  onClose?: () => void
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
          // 持续从 buffer 中提取事件，支持 SSE(\n\n) 与裸 JSON({...}) 两种格式
          while (true) {
            const boundary = findEventBoundary(buffer)
            if (!boundary) break
            const rawEvent = buffer.slice(0, boundary.end)
            buffer = buffer.slice(boundary.end)
            parseStreamFrame(rawEvent, handlers)
          }
        }
        // 流被后端正常关闭（done: true），且不是用户主动 stop 时通知上层收尾。
        if (!stopped) {
          handlers.onClose?.()
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

/** 把一段原始块解析为标准 StreamEvent 并派发到对应 handler。模块作用域私有工具。
 *  - 以 `{` 开头：视为裸 JSON（streamEvent 直接序列化）；
 *  - 否则：按 SSE 标准格式解析 event / data 行后 JSON.parse data。
 */
function parseStreamFrame(rawBlock: string, handlers: StreamHandlers): void {
  const trimmed = rawBlock.trim()
  if (!trimmed) return

  if (trimmed.startsWith('{')) {
    // 路径 1：裸 JSON
    const event = parseJsonAsStreamEvent(trimmed)
    if (event) dispatchStreamEvent(event, handlers)
    return
  }

  // 路径 2：SSE 标准格式
  const frame = parseSseFrame(trimmed)
  if (!frame) return
  const event = parseJsonAsStreamEvent(frame.data)
  if (event) {
    dispatchStreamEvent(event, handlers)
  } else if (frame.event) {
    // 容错：data 不是合法 JSON 时，回退到 event 行作为类型，data 作为文本 content
    dispatchStreamEvent({ type: frame.event as StreamEventType, content: frame.data }, handlers)
  }
}

/** 解析 SSE 帧的 event / data 行。
 *  规范：连续多行 data: 用 `\n` 拼接；event: 取首个；`:` 开头为注释。
 *  返回 null 表示该块没有任何 data 可解析。
 */
function parseSseFrame(block: string): SseEvent | null {
  let event: string | undefined
  const dataLines: string[] = []
  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
    } else if (line.startsWith('data:')) {
      // 规范：去掉 data: 后第一个前导空格（若有）
      dataLines.push(line.startsWith('data: ') ? line.slice(6) : line.slice(5))
    } else if (line.startsWith(':')) {
      // 注释行，忽略
    } else if (line.startsWith('{')) {
      // 容错：data 行换行后丢前缀（多行 JSON）
      dataLines.push(line)
    }
    // 其他行忽略
  }
  if (dataLines.length === 0) return null
  return { event, data: dataLines.join('\n') }
}

function parseJsonAsStreamEvent(text: string): StreamEvent | null {
  try {
    const value: StreamEvent = JSON.parse(text)
    return isStreamEvent(value) ? value : null
  } catch {
    return null
  }
}

function isStreamEvent(value: unknown): value is StreamEvent {
  if (!value || typeof value !== 'object') return false
  const v = value as Record<string, unknown>
  return typeof v.type === 'string' && STREAM_EVENT_TYPES.has(v.type as StreamEventType)
}

const CHANNEL_LOCALIZATION: Record<string, string> = {
  keyword: '关键词检索',
  vector: '向量检索',
  rerank: '重排精排',
  hybrid: '融合结果',
  'web-search': '网页搜索'
}

const SOURCE_TYPE_LOCALIZATION: Record<string, string> = {
  document: '文档',
  web: '网页',
  knowledge: '知识库'
}

function localizeReference(item: SearchReference): SearchReference {
  return {
    ...item,
    channel: CHANNEL_LOCALIZATION[item.channel] || item.channel,
    sourceType: SOURCE_TYPE_LOCALIZATION[item.sourceType] || item.sourceType
  }
}

/** 把单个标准 StreamEvent 派发到对应 handler。模块作用域私有工具。 */
function dispatchStreamEvent(event: StreamEvent, handlers: StreamHandlers): void {
  switch (event.type) {
    case 'text':
      handlers.onText?.({ content: readString(event.content) })
      break
    case 'thinking':
      handlers.onThinking?.({ content: readString(event.content) })
      break
    case 'status':
      handlers.onStatus?.({
        stage: event.stage || 'pending',
        content: readString(event.content)
      })
      break
    case 'reference':
      handlers.onReference?.({
        items: readArray<SearchReference>(event.content).map(localizeReference),
        count: event.count
      })
      break
    case 'recommend':
      handlers.onRecommend?.({
        items: readArray(event.content).filter((x): x is string => typeof x === 'string'),
        count: event.count
      })
      break
    case 'finish':
      handlers.onFinish?.({ messageId: event.exchangeId, title: event.title })
      break
    case 'cancel':
      handlers.onCancel?.({ messageId: event.exchangeId, title: event.title })
      break
    case 'error':
      handlers.onError?.(new Error(readString(event.content) || '生成出错'))
      break
  }
}

function readString(content: StreamEvent['content']): string {
  return typeof content === 'string' ? content : ''
}

function readArray<T>(content: StreamEvent['content']): T[] {
  return Array.isArray(content) ? (content as T[]) : []
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
