import type {ConversationExchange, SearchReference, SubQuestion} from '@/types'
import type {Chip, ListBlock, Metric, TextBlock} from './types'
import {formatNum} from '@/utils/format'
import {formatChannelName} from './utils'

export function pushTextBlock(target: TextBlock[], label: string, value: string | undefined, options: {
  code?: boolean
} = {}): void {
  if (!value) {
    return
  }
  target.push({
    label,
    value,
    code: Boolean(options.code)
  })
}

export function pushListBlock(target: ListBlock[], label: string, items: string[] | undefined, options: {
  ordered?: boolean
} = {}): void {
  const values = items || []
  if (!values.length) {
    return
  }
  target.push({
    label,
    items: values,
    ordered: Boolean(options.ordered)
  })
}

export function pushPair(target: TextBlock[], label: string, value: string | number | undefined, options: {
  code?: boolean
} = {}): void {
  if (value == null || value === '') {
    return
  }
  target.push({
    label,
    value: String(value),
    code: Boolean(options.code)
  })
}

export function buildChips(...entries: (Chip | null | undefined)[]): Chip[] {
  return entries.flat()
    .filter(item => item && item.value)
    .map((item) => ({
      label: item?.label || '',
      value: item?.value || '',
      tone: item?.tone || 'neutral'
    }))
}

export function buildMetrics(...entries: (Metric | null | undefined)[]): Metric[] {
  return entries.flat()
    .filter(item => item && item.value && item.value !== '无')
    .map((item) => ({
      label: item?.label || '',
      value: item?.value || '',
      mono: Boolean(item?.mono)
    }))
}

export function buildOutcomeSummary(exchange: ConversationExchange | undefined, references: SearchReference[]): string {
  if (!exchange) {
    return ''
  }
  if (exchange.turnStatus === 2) {
    if (references.length > 0) {
      return `本轮已完成，并基于 ${references.length} 条最终证据生成回答。排障时优先核对这些引用是否真的支撑了答案。`
    }
    return '本轮已完成，但没有看到最终引用，适合继续检查检索或 Prompt 组装阶段。'
  }
  if (exchange.turnStatus === 3) {
    return exchange.errorMessage ? `本轮执行失败，结束原因是：${exchange.errorMessage}` : '本轮执行失败，但当前没有拿到更具体的错误说明。'
  }
  if (exchange.turnStatus === 4) {
    return exchange.errorMessage ? `本轮被主动停止，结束说明是：${exchange.errorMessage}` : '本轮被主动停止。'
  }
  return '这是一条正在执行中的轮次，建议优先关注执行过程提示和实时状态。'
}

export function stageUsageDetails(exchange: ConversationExchange | null, stageNames: string[]): string[] {
  const traces = exchange?.debugTrace?.modelUsageTraces || []
  return traces
    .filter(item => stageNames.includes(item?.stageName || ''))
    .map(item => {
      const tokens = item?.totalTokens ? `总Token ${item.totalTokens}` : ''
      const prompt = item?.promptTokens ? `输入 ${item.promptTokens}` : ''
      const completion = item?.completionTokens ? `输出 ${item.completionTokens}` : ''
      const cost = item?.estimatedCost ? `成本约 ¥${formatNum(item.estimatedCost, 4)}` : ''
      const duration = item?.durationMs ? `耗时 ${item.durationMs} ms` : ''
      return `${item?.stageName || 'unknown'} | ${item?.provider || 'unknown'} / ${item?.model || 'unknown'} | ${[prompt, completion, tokens, cost, duration].filter(Boolean).join('，')}`
    })
}

export function formatSubQuestions(subQuestions: SubQuestion[]): string[] {
  return subQuestions.map((item) => {
    const channelTraces = item.channelTraces || []
    const channelTraceText = channelTraces.map((trace) => {
      return `${formatChannelName(trace.channelName)} raw=${trace.recalledCount || 0} accepted=${trace.acceptedCount || 0}`
    }).filter(Boolean).join('；')
    return `${item.index}. ${item.question} | 通道 ${channelTraceText || '无'} | fused ${item.fusedCandidateCount || 0} | parent ${item.parentCandidateCount || 0} | rerank ${item.rerankedCandidateCount || 0} | 文档 ${item.documentCount || 0} | 引用 ${item.referenceCount || 0}`
  }).filter(Boolean);
}

export function buildRagRetrieveRows(subQuestions: SubQuestion[]) {
  return subQuestions.map((item) => {
    const keywordTrace = item.channelTraces.find((trace) => trace.channelName === 'keyword')
    const vectorTrace = item.channelTraces.find((trace) => trace.channelName === 'vector')
    return {
      cells: [
        `${item.index}. ${item.question}`,
        `${keywordTrace?.recalledCount ?? 0} / ${keywordTrace?.acceptedCount ?? 0}`,
        `${vectorTrace?.recalledCount ?? 0} / ${vectorTrace?.acceptedCount ?? 0}`,
        String(item.fusedCandidateCount ?? 0),
        String(item.parentCandidateCount ?? 0),
        String(item.rerankedCandidateCount ?? 0),
        String(item.referenceCount ?? 0)
      ]
    }
  });
}

export function buildReferenceDecisionRows(details: string[] = []): {
  reference: string;
  reason: string
}[] {
  return details.filter(Boolean).map((detail) => {
    const text = String(detail || '')
    const index = text.lastIndexOf(' | ')
    if (index === -1) {
      return {
        reference: text,
        reason: ''
      }
    }
    return {
      reference: text.slice(0, index),
      reason: text.slice(index + 3)
    }
  })
}
