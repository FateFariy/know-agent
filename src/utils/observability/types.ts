import type {ChatToolTrace, RetrievalResultResp, SearchReference} from '@/types'

export type ExecutionModeType = 'graph_only' | 'graph_then_evidence' | 'retrieval' | 'react_agent' | 'clarification'
export type RelationType = 'FOLLOW_UP' | 'TOPIC_SWITCH' | 'FRESH_TOPIC' | 'UNKNOWN'
export type RetrievalModeType =
  'DIRECT_QUERY'
  | 'SECTION_FOCUSED'
  | 'ANALYTIC_DECOMPOSITION'
  | 'UNKNOWN'
export type AnswerShapeType =
  'LIST'
  | 'STEPS'
  | 'OUTLINE'
  | 'COMPARISON'
  | 'EXPLANATION'
  | 'JUDGMENT'
  | 'FACT'
  | 'UNKNOWN'
export type ChannelType = 'keyword' | 'vector' | 'rerank' | 'hybrid' | 'web-search'
export type ToolType = 'tavily_search' | 'keyword' | 'vector' | 'rerank'

export interface TextBlock {
  label: string
  value: string
  code?: boolean
}

export interface ListBlock {
  label: string
  items: string[]
  ordered: boolean
}

export interface Chip {
  label: string
  value: string
  tone: string
}

export interface Metric {
  label: string
  value: string
  mono: boolean
}

export interface ExchangeStage {
  key: string
  eyebrow: string
  title: string
  subtitle: string
  tone: string
  chips: Chip[]
  metrics: Metric[]
  textBlocks: TextBlock[]
  listBlocks: ListBlock[]
  references?: SearchReference[]
  toolTraces?: ChatToolTrace[]
  advancedTextBlocks: TextBlock[]
  advancedListBlocks: ListBlock[]
  advancedToolTraces?: ChatToolTrace[]
  advancedReferences?: SearchReference[]
}

export interface StageInspectorSection {
  label: string
  items: string[]
  ordered: boolean
}

export interface TableRow {
  cells: string[]
}

export interface StageInspectorTable {
  label: string
  columns: string[]
  rows: TableRow[]
}

export interface StageInspector {
  title: string
  summary: string
  stageState: number
  startTime: string
  endTime: string
  durationMs: number | undefined
  summaryItems: TextBlock[]
  listSections: StageInspectorSection[]
  tableSections: StageInspectorTable[]
  advancedItems: TextBlock[]
}

export interface SubQuestionChannel {
  type: string
  results: RetrievalResultResp[]
}

export interface GroupedSubQuestion {
  index: number
  question: string
  channels: SubQuestionChannel[]
}
