import type {
  AnswerShapeType,
  ChannelType,
  ExecutionModeType,
  RelationType,
  RetrievalModeType,
  ToolType
} from './types'

export const STAGE_STATE_LABELS: Record<number, string> = {
  1: '进行中',
  2: '已完成',
  3: '失败',
  4: '跳过'
}

export const TURN_STATUS_LABELS: Record<number, string> = {
  1: '进行中',
  2: '已完成',
  3: '失败',
  4: '已停止'
}

export const TURN_STATUS_TONES: Record<number, string> = {
  1: 'running',
  2: 'completed',
  3: 'failed',
  4: 'stopped'
}

export const STAGE_STATE_TONES: Record<number, string> = {
  1: 'running',
  2: 'completed',
  3: 'failed',
  4: 'idle'
}

export const EXECUTION_MODE_LABELS: Record<ExecutionModeType, string> = {
  RAG_CHAT: '文档检索问答',
  REACT_AGENT: 'Agent 自主执行',
  CLARIFICATION: '路由澄清'
}

export const RELATION_TYPE_LABELS: Record<RelationType, string> = {
  FOLLOW_UP: '承接上文追问',
  TOPIC_SWITCH: '切换到新主题',
  FRESH_TOPIC: '独立新问题',
  UNKNOWN: '未识别'
}

export const RETRIEVAL_MODE_LABELS: Record<RetrievalModeType, string> = {
  DIRECT_QUERY: '直接检索',
  SECTION_FOCUSED: '定向查章节',
  ANALYTIC_DECOMPOSITION: '拆成多个子问题',
  UNKNOWN: '未识别'
}

export const ANSWER_SHAPE_LABELS: Record<AnswerShapeType, string> = {
  LIST: '列表型回答',
  STEPS: '步骤型回答',
  OUTLINE: '提纲型回答',
  COMPARISON: '对比型回答',
  EXPLANATION: '解释型回答',
  JUDGMENT: '判断型回答',
  FACT: '事实型回答',
  UNKNOWN: '未识别'
}

export const CHANNEL_LABELS: Record<ChannelType, string> = {
  keyword: '关键词检索',
  vector: '向量检索',
  rerank: '重排精排',
  hybrid: '融合结果',
  'web-search': '网页搜索'
}

export const EXECUTION_STATE_LABELS: Record<string, string> = {
  '1': '成功',
  '2': '失败',
  '3': '超时',
  '4': '跳过'
}

export const TOOL_LABELS: Record<ToolType, string> = {
  tavily_search: 'Tavily 联网搜索',
  keyword: '关键词检索通道',
  vector: '向量检索通道',
  rerank: '重排精排'
}

export const STAGE_USAGE_NAMES: Record<string, string> = {
  intent: '意图分析',
  rewrite: '问题改写',
  summary: '会话记忆压缩',
  rag_answer: '回答生成',
  recommendation: '推荐问题',
  react_agent_turn: 'Agent 推理'
}
