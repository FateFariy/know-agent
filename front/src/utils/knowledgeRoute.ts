import type {
  DocumentRouteCandidate,
  KnowledgeRouteTraceItem,
  ScopeRouteCandidate,
  TopicRouteCandidate
} from '@/types'
import JSONbig from 'json-bigint'
import {formatNum, formatPercent} from '@/utils/format.ts'

type RouteMode = 'auto' | 'shadow'

type RouteStatusKey = 'SUCCESS' | 'LOW_CONFIDENCE' | 'FAILED'

type StatusTone = 'success' | 'warning' | 'danger'

interface RouteStatusMeta {
  key: RouteStatusKey
  label: string
  tone: StatusTone
}

interface ConfidenceBand {
  label: string
  tone: StatusTone
}

export interface NormalizedRouteTrace extends KnowledgeRouteTraceItem {
  modeLabel: string
  scopes: ScopeRouteCandidate[]
  topics: TopicRouteCandidate[]
  documents: DocumentRouteCandidate[]
  topDocument: DocumentRouteCandidate | null
  selectedDocument: DocumentRouteCandidate | null
  confidenceText: string
  confidenceBand: ConfidenceBand
  statusKey: RouteStatusKey
  statusLabel: string
  statusTone: StatusTone
  reason: string
  hitTop3: boolean
  missedTop3: boolean
  candidateDocumentCount: number
  candidateTopicCount: number
  candidateScopeCount: number
  lowConfidenceWidened: boolean
}

export interface RouteExplain extends NormalizedRouteTrace {
  summary: string
  notes: string[]
  topDocuments: DocumentRouteCandidate[]
  scopePreview: ScopeRouteCandidate[]
  topicPreview: TopicRouteCandidate[]
}

export interface RouteTraceSummary {
  total: number
  autoCount: number
  shadowCount: number
  successCount: number
  lowConfidenceCount: number
  failedCount: number
  highConfidenceCount: number
  widenedCount: number
  uniqueTopDocumentCount: number
  averageConfidenceText: string
  averageDocumentCountText: string
  averageTopicCountText: string
  averageScopeCountText: string
  successRateText: string
  lowConfidenceRateText: string
  shadowHitRateText: string
}

export interface DocumentDistributionItem {
  documentId: string
  documentName: string
  count: number
  confidenceTotal: number
  confidenceCount: number
  lowConfidenceCount: number
  averageConfidenceText: string
}

const ROUTE_MODE_LABELS: Record<RouteMode, string> = {
  auto: '自动知识路由',
  shadow: '影子路由对比'
}

const ROUTE_STATUS_ALIAS: Record<string, RouteStatusKey> = {
  '1': 'SUCCESS',
  '2': 'LOW_CONFIDENCE',
  '3': 'FAILED',
  SUCCESS: 'SUCCESS',
  LOW_CONFIDENCE: 'LOW_CONFIDENCE',
  FAILED: 'FAILED'
}

const ROUTE_STATUS_META: Record<RouteStatusKey, RouteStatusMeta> = {
  SUCCESS: {
    key: 'SUCCESS',
    label: '成功',
    tone: 'success'
  },
  LOW_CONFIDENCE: {
    key: 'LOW_CONFIDENCE',
    label: '低置信',
    tone: 'warning'
  },
  FAILED: {
    key: 'FAILED',
    label: '失败',
    tone: 'danger'
  }
}

type RouteCandidate = ScopeRouteCandidate | TopicRouteCandidate | DocumentRouteCandidate

function parseCandidateList(rawValue: string): RouteCandidate[] {
  if (!rawValue) {
    return []
  }

  try {
    const parsed = JSONbig({ storeAsString: true }).parse(rawValue)
    if (!Array.isArray(parsed)) return []
    return parsed as RouteCandidate[]
  } catch {
    return []
  }
}

function resolveRouteStatusMeta(value: string): RouteStatusMeta {
  const alias = ROUTE_STATUS_ALIAS[value] || 'FAILED'
  return ROUTE_STATUS_META[alias] || ROUTE_STATUS_META.FAILED
}

function resolveConfidenceBand(value: number | null): ConfidenceBand {
  if (value == null || value <= 0) {
    return {
      label: '未形成有效置信度',
      tone: 'danger'
    }
  }
  if (value >= 0.8) {
    return {
      label: '高置信',
      tone: 'success'
    }
  }
  if (value >= 0.55) {
    return {
      label: '可用但偏保守',
      tone: 'warning'
    }
  }
  return {
    label: '需要扩范围',
    tone: 'danger'
  }
}

function normalizeCandidate(item: RouteCandidate): RouteCandidate {
  const scoreText = item.score == null ? '-' : item.score.toFixed(4)
  return { ...item, scoreText }
}

function average(numbers: number[]): number | null {
  if (!numbers.length) {
    return null
  }
  const total = numbers.reduce((sum, value) => sum + value, 0)
  return total / numbers.length
}

export function normalizeRouteTrace(record: KnowledgeRouteTraceItem): NormalizedRouteTrace {
  const scopes = parseCandidateList(record.topScopesJson).map(normalizeCandidate)
  const topics = parseCandidateList(record.topTopicsJson).map(normalizeCandidate)
  const documents = parseCandidateList(record.topDocumentsJson).map(normalizeCandidate)
  const statusMeta = resolveRouteStatusMeta(record.routeStatus)
  const selectedDocument = record.selectedDocumentId
    ? documents.find((item) => (item as DocumentRouteCandidate).documentId === record.selectedDocumentId) ?? null
    : null
  const topDocument = documents[0] ?? null
  return {
    ...record,
    modeLabel: ROUTE_MODE_LABELS[record.mode as RouteMode] || record.mode || '未知路由模式',
    scopes: scopes as ScopeRouteCandidate[],
    topics: topics as TopicRouteCandidate[],
    documents: documents as DocumentRouteCandidate[],
    topDocument: topDocument as DocumentRouteCandidate | null,
    selectedDocument: selectedDocument as DocumentRouteCandidate | null,
    confidenceText: record.confidence == null ? '-' : record.confidence.toFixed(4),
    confidenceBand: resolveConfidenceBand(record.confidence),
    statusKey: statusMeta.key,
    statusLabel: statusMeta.label,
    statusTone: statusMeta.tone,
    reason: record.errorMsg || topDocument?.reason || '',
    hitTop3: record.hitSelectedDocument === 1,
    missedTop3: record.hitSelectedDocument === 0,
    candidateDocumentCount: documents.length,
    candidateTopicCount: topics.length,
    candidateScopeCount: scopes.length,
    lowConfidenceWidened: record.mode === 'auto' && !record.confidence && record.confidence < 0.8 && documents.length >= 5
  }
}

export function buildRouteTraceLookup(records: KnowledgeRouteTraceItem[] = []): Map<string, NormalizedRouteTrace> {
  const map = new Map<string, NormalizedRouteTrace>();
  const normalizedList = records.map(normalizeRouteTrace);

  for (const item of normalizedList) {
    const { exchangeId } = item;
    if (exchangeId == null) continue;

    const old = map.get(exchangeId);
    if (!old) {
      map.set(exchangeId, item);
      continue;
    }

    // 优先级1：auto模式优先覆盖非auto
    if (item.mode === 'auto' && old.mode !== 'auto') {
      map.set(exchangeId, item);
      continue;
    }

    // 优先级2：同模式下取创建时间更新的
    if (item.createTime >= old.createTime) {
      map.set(exchangeId, item);
    }
  }
  return map;
}

export function buildChatRouteExplain(record: KnowledgeRouteTraceItem | null | undefined): RouteExplain | null {
  if (!record) {
    return null
  }

  const trace = normalizeRouteTrace(record)
  const topDocuments = trace.documents.slice(0, 5)
  const scopePreview = trace.scopes.slice(0, 3)
  const topicPreview = trace.topics.slice(0, 3)
  const notes: string[] = []
  let summary = ''

  if (trace.mode === 'auto') {
    summary = trace.topDocument
      ? `系统先做知识范围预选，再把 ${trace.candidateDocumentCount} 份候选文档交给稳定检索链路；当前主候选是「${trace.topDocument.documentName || trace.topDocument.documentId}」。`
      : '系统先做知识范围预选，再进入稳定检索链路；本轮没有形成稳定的显式主候选文档。'

    if (trace.lowConfidenceWidened) {
      notes.push('当前置信度偏低，系统已放宽候选范围后再进入稳定检索。')
    }
    if (!trace.documents.length) {
      notes.push('原始路由没有产出显式候选文档，执行期会回退到可检索文档池。')
    }
    if (trace.reason) {
      notes.push(`路由依据：${trace.reason}`)
    }
  } else if (trace.mode === 'shadow') {
    summary = trace.topDocument
      ? `系统对这轮问题做了影子路由对比，影子 Top1 是「${trace.topDocument.documentName || trace.topDocument.documentId}」，但实际回答仍固定使用你手动选择的当前文档。`
      : '系统对这轮问题做了影子路由对比，但没有形成稳定的影子候选文档。'

    if (trace.hitTop3) {
      notes.push('影子路由 Top3 已覆盖当前文档，说明自动路由与人工选文档基本一致。')
    }
    if (trace.missedTop3) {
      notes.push('影子路由 Top3 未覆盖当前文档，说明这轮问题更像跨文档或元数据仍需补强。')
    }
    if (trace.reason) {
      notes.push(`影子路由依据：${trace.reason}`)
    }
  } else {
    return null
  }

  return {
    ...trace,
    summary,
    notes,
    topDocuments,
    scopePreview,
    topicPreview
  }
}

export function summarizeRouteTraceRecords(records: KnowledgeRouteTraceItem[] = []): RouteTraceSummary {
  const normalized = records.map(normalizeRouteTrace)
  const total = normalized.length

  let autoCount = 0
  let shadowCount = 0
  let successCount = 0
  let lowConfidenceCount = 0
  let failedCount = 0
  let highConfidenceCount = 0
  let widenedCount = 0
  let shadowHitCount = 0
  let shadowSampleTotal = 0
  const confidenceList: number[] = []
  const docCountList: number[] = []
  const topicCountList: number[] = []
  const scopeCountList: number[] = []
  const topDocIdSet = new Set<string>()

  for (const item of normalized) {
    // 模式计数
    if (item.mode === 'auto') autoCount++
    if (item.mode === 'shadow') shadowCount++

    // 状态计数
    if (item.statusKey === 'SUCCESS') successCount++
    if (item.statusKey === 'LOW_CONFIDENCE') lowConfidenceCount++
    if (item.statusKey === 'FAILED') failedCount++

    // 高置信度
    const conf = item.confidence ?? 0
    confidenceList.push(conf)
    if (conf >= 0.8) highConfidenceCount++

    // 扩量标记
    if (item.lowConfidenceWidened) widenedCount++

    // 候选文档/主题/范围数量
    docCountList.push(item.candidateDocumentCount ?? 0)
    topicCountList.push(item.candidateTopicCount ?? 0)
    scopeCountList.push(item.candidateScopeCount ?? 0)

    // 影子样本命中统计
    if (item.mode === 'shadow' && (item.hitTop3 || item.missedTop3)) {
      shadowSampleTotal++
      if (item.hitTop3) shadowHitCount++
    }

    if (item.topDocument) {
      const uniqueKey = item.topDocument.documentId || item.topDocument.documentName
      if (uniqueKey) topDocIdSet.add(uniqueKey)
    }
  }

  // 均值计算
  const averageConfidence = average(confidenceList)
  const avgDocumentCount = average(docCountList)
  const avgTopicCount = average(topicCountList)
  const avgScopeCount = average(scopeCountList)

  // 比率计算
  const successRate = total ? (successCount / total) * 100 : null
  const lowConfidenceRate = total ? ((lowConfidenceCount + failedCount) / total) * 100 : null
  const shadowHitRate = shadowSampleTotal ? (shadowHitCount / shadowSampleTotal) * 100 : null

  return {
    total,
    autoCount,
    shadowCount,
    successCount,
    lowConfidenceCount,
    failedCount,
    highConfidenceCount,
    widenedCount,
    uniqueTopDocumentCount: topDocIdSet.size,
    averageConfidenceText: formatNum(averageConfidence, 4),
    averageDocumentCountText: formatNum(avgDocumentCount, 1),
    averageTopicCountText: formatNum(avgTopicCount, 1),
    averageScopeCountText: formatNum(avgScopeCount, 1),
    successRateText: formatPercent(successRate),
    lowConfidenceRateText: formatPercent(lowConfidenceRate),
    shadowHitRateText: formatPercent(shadowHitRate)
  }
}

export function buildTopDocumentDistribution(records: KnowledgeRouteTraceItem[] = []): DocumentDistributionItem[] {
  const rows = records
    .map(normalizeRouteTrace)
    .filter((item) => item.topDocument)
    .reduce((map, item) => {
      const documentId = item.topDocument?.documentId || item.topDocument?.documentName || 'unknown'
      const existing = map.get(documentId) || {
        documentId,
        documentName: item.topDocument?.documentName || item.topDocument?.documentId || '未知文档',
        count: 0,
        confidenceTotal: 0,
        confidenceCount: 0,
        lowConfidenceCount: 0,
        averageConfidenceText: ''
      }
      existing.count += 1
      if (item.confidence != null) {
        existing.confidenceTotal += item.confidence
        existing.confidenceCount += 1
      }
      if (item.statusKey !== 'SUCCESS') {
        existing.lowConfidenceCount += 1
      }
      map.set(documentId, existing)
      return map
    }, new Map<string, DocumentDistributionItem>())

  return [...rows.values()]
    .map((item) => {
      const averageConfidenceText = formatNum(item.confidenceTotal / item.confidenceCount, 4)
      return ({ ...item, averageConfidenceText })
    })
    .sort((left, right) => right.count - left.count)
    .slice(0, 6)
}
