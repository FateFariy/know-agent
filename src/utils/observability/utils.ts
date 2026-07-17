import {APIError} from '@/api/axios'
import {formatNum} from '@/utils/format'
import type {
  AnswerShapeType,
  ChannelType,
  ExecutionModeType,
  RelationType,
  RetrievalModeType,
  ToolType
} from './types'
import {
  ANSWER_SHAPE_LABELS,
  CHANNEL_LABELS,
  EXECUTION_MODE_LABELS,
  EXECUTION_STATE_LABELS,
  RELATION_TYPE_LABELS,
  RETRIEVAL_MODE_LABELS,
  STAGE_STATE_LABELS,
  STAGE_STATE_TONES, STAGE_USAGE_NAMES,
  TOOL_LABELS,
  TURN_STATUS_LABELS,
  TURN_STATUS_TONES
} from './constants'

export function normalizeError(error: unknown, fallbackMessage: string): string {
  if (error instanceof APIError && error.message) {
    return error.message
  }
  if (error instanceof Error && error.message) {
    return error.message
  }
  return fallbackMessage
}

export function truncate(value: string | undefined, maxLength: number): string {
  if (!value) {
    return ''
  }
  return value.length > maxLength ? `${value.slice(0, maxLength)}...` : value
}

export function formatTime(value: string | undefined): string {
  if (!value) {
    return '刚刚'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '刚刚'
  }
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

export function formatChatMode(value: string | undefined): string {
  if (value === 'document') {
    return '当前文档问答'
  }
  if (value === 'auto_document') {
    return '自动知识问答'
  }
  if (value === 'open_chat') {
    return '开放式提问'
  }
  return value || '未知模式'
}

export function formatStageStateLabel(value: number | undefined): string {
  return STAGE_STATE_LABELS[value || 0] || '未知状态'
}

export function formatTurnStatusLabel(value: number | undefined): string {
  return TURN_STATUS_LABELS[value || 0] || '未知状态'
}

export function turnStatusTone(value: number | undefined): string {
  return TURN_STATUS_TONES[value || 0] || 'idle'
}

export function stageStateTone(value: number | undefined): string {
  return STAGE_STATE_TONES[value || 0] || 'idle'
}

export function formatExecutionMode(value: string | undefined): string {
  return EXECUTION_MODE_LABELS[value as ExecutionModeType] || '未知'
}

export function formatRelationType(value: string | undefined): string {
  return RELATION_TYPE_LABELS[value as RelationType] || '未知'
}

export function formatUsageStageName(stageName: string | undefined): string {
  return STAGE_USAGE_NAMES[stageName || ''] || stageName || '未知阶段'
}

export function formatRetrievalMode(value: string | undefined): string {
  return RETRIEVAL_MODE_LABELS[value as RetrievalModeType] || '未知'
}

export function formatAnswerShape(value: string | undefined): string {
  return ANSWER_SHAPE_LABELS[value as AnswerShapeType] || '未知'
}

export function formatChannelName(value: string | undefined): string {
  return CHANNEL_LABELS[value as ChannelType] || '未知通道'
}

export function formatChannelType(value: string | undefined): string {
  return CHANNEL_LABELS[value as ChannelType] || '未知通道'
}

export function formatToolName(value: string | undefined): string {
  return TOOL_LABELS[value as ToolType] || '未知工具'
}

export function formatExecutionState(value: number | undefined): string {
  return EXECUTION_STATE_LABELS[value || 0] || '未知'
}

export function formatScore(value: number | undefined): string {
  return formatNum(value, 4)
}

export function formatRank(value: number | string | undefined): string {
  if (value == null || value === '') {
    return '-'
  }
  return String(value)
}

export function formatLatency(value: number | undefined): string {
  if (!value || value <= 0) {
    return '无'
  }
  return `${value} ms`
}

export function formatConfidence(value: number | undefined): string {
  if (value == null || Number.isNaN(Number(value))) {
    return ''
  }
  return `${Math.round(Number(value) * 100)}%`
}

export function asList<T>(value: T[] | undefined): T[] {
  return Array.isArray(value) ? value.filter(Boolean) : []
}

export function uniqueStrings(values: string[]): string[] {
  const result: string[] = []
  const seen = new Set<string>()
  values.filter(Boolean).forEach((item) => {
    if (seen.has(item)) {
      return
    }
    seen.add(item)
    result.push(item)
  })
  return result
}
