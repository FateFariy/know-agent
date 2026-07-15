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

interface CandidateItem {
  score?: number
  documentId?: number
  documentName?: string
  reason?: string
  [key: string]: unknown
}

interface NormalizedCandidate extends CandidateItem {
  scoreNumber: number | null
  scoreText: string
}

interface RouteTraceRecord {
  mode?: unknown
  topScopesJson?: unknown
  topTopicsJson?: unknown
  topDocumentsJson?: unknown
  confidence?: unknown
  routeStatus?: unknown
  selectedDocumentId?: unknown
  createTime?: unknown
  hitSelectedDocument?: unknown
  errorMsg?: unknown
  exchangeId?: unknown
  [key: string]: unknown
}

export interface NormalizedRouteTrace extends RouteTraceRecord {
  mode: string
  modeLabel: string
  scopes: NormalizedCandidate[]
  topics: NormalizedCandidate[]
  documents: NormalizedCandidate[]
  topDocument: NormalizedCandidate | null
  selectedDocumentId: string
  selectedDocument: NormalizedCandidate | null
  confidenceNumber: number | null
  confidenceText: string
  confidenceBand: ConfidenceBand
  statusKey: RouteStatusKey
  statusLabel: string
  statusTone: StatusTone
  reasonText: string
  createTimeNumber: number
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
  topDocuments: NormalizedCandidate[]
  scopePreview: NormalizedCandidate[]
  topicPreview: NormalizedCandidate[]
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

function asString(value: unknown): string {
  if (value == null) {
    return ''
  }
  return String(value).trim()
}

function toNumber(value: unknown): number | null {
  const num = Number(value)
  return Number.isFinite(num) ? num : null
}

function parseCandidateList(rawValue: unknown): CandidateItem[] {
  const normalized = asString(rawValue)
  if (!normalized) {
    return []
  }

  try {
    const parsed = JSON.parse(normalized)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function resolveRouteStatusMeta(value: unknown): RouteStatusMeta {
  const alias = ROUTE_STATUS_ALIAS[asString(value)] || 'FAILED'
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

function normalizeCandidate(item: CandidateItem = {}): NormalizedCandidate {
  const scoreNumber = toNumber(item.score)
  return {
    ...item,
    scoreNumber,
    scoreText: scoreNumber == null ? '-' : scoreNumber.toFixed(4)
  }
}

function average(numbers: number[]): number | null {
  if (!numbers.length) {
    return null
  }
  const total = numbers.reduce((sum, value) => sum + value, 0)
  return total / numbers.length
}

export function formatRouteMode(value: unknown): string {
  return ROUTE_MODE_LABELS[asString(value) as RouteMode] || asString(value) || '未知路由模式'
}

export function normalizeRouteTrace(record: RouteTraceRecord = {}): NormalizedRouteTrace {
  const scopes = parseCandidateList(record.topScopesJson).map(normalizeCandidate)
  const topics = parseCandidateList(record.topTopicsJson).map(normalizeCandidate)
  const documents = parseCandidateList(record.topDocumentsJson).map(normalizeCandidate)
  const confidenceNumber = toNumber(record.confidence)
  const statusMeta = resolveRouteStatusMeta(record.routeStatus)
  const selectedDocumentId = asString(record.selectedDocumentId)
  const selectedDocument = selectedDocumentId
    ? documents.find((item) => asString(item.documentId) === selectedDocumentId) ?? null
    : null
  const topDocument = documents[0] ?? null
  const createTimeNumber = toNumber(record.createTime) ?? 0
  const hitSelectedDocument = asString(record.hitSelectedDocument)
  const mode = asString(record.mode)

  return {
    ...record,
    mode,
    modeLabel: formatRouteMode(mode),
    scopes,
    topics,
    documents,
    topDocument,
    selectedDocumentId,
    selectedDocument,
    confidenceNumber,
    confidenceText: confidenceNumber == null ? '-' : confidenceNumber.toFixed(4),
    confidenceBand: resolveConfidenceBand(confidenceNumber),
    statusKey: statusMeta.key,
    statusLabel: statusMeta.label,
    statusTone: statusMeta.tone,
    reasonText: asString(record.errorMsg) || asString(topDocument?.reason),
    createTimeNumber,
    hitTop3: hitSelectedDocument === '1',
    missedTop3: hitSelectedDocument === '0',
    candidateDocumentCount: documents.length,
    candidateTopicCount: topics.length,
    candidateScopeCount: scopes.length,
    lowConfidenceWidened: mode === 'auto' && confidenceNumber != null && confidenceNumber < 0.8 && documents.length >= 5
  }
}

export function buildRouteTraceLookup(records: RouteTraceRecord[] = []): Record<string, NormalizedRouteTrace> {
  return records.reduce((lookup, item) => {
    const normalized = normalizeRouteTrace(item)
    const exchangeId = asString(normalized.exchangeId)
    if (!exchangeId) {
      return lookup
    }
    const existing = lookup[exchangeId]
    if (!existing) {
      lookup[exchangeId] = normalized
      return lookup
    }
    if (normalized.mode === 'auto' && existing.mode !== 'auto') {
      lookup[exchangeId] = normalized
      return lookup
    }
    if (normalized.createTimeNumber >= existing.createTimeNumber) {
      lookup[exchangeId] = normalized
    }
    return lookup
  }, {} as Record<string, NormalizedRouteTrace>)
}

export function buildChatRouteExplain(record: RouteTraceRecord | null | undefined): RouteExplain | null {
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
    if (trace.reasonText) {
      notes.push(`路由依据：${trace.reasonText}`)
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
    if (trace.reasonText) {
      notes.push(`影子路由依据：${trace.reasonText}`)
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

export function summarizeRouteTraceRecords(records: RouteTraceRecord[] = []): RouteTraceSummary {
  const normalized = records.map(normalizeRouteTrace)
  const autoCount = normalized.filter((item) => item.mode === 'auto').length
  const shadowCount = normalized.filter((item) => item.mode === 'shadow').length
  const successCount = normalized.filter((item) => item.statusKey === 'SUCCESS').length
  const lowConfidenceCount = normalized.filter((item) => item.statusKey === 'LOW_CONFIDENCE').length
  const failedCount = normalized.filter((item) => item.statusKey === 'FAILED').length
  const highConfidenceCount = normalized.filter((item) => (item.confidenceNumber ?? 0) >= 0.8).length
  const confidenceValues = normalized
    .map((item) => item.confidenceNumber)
    .filter((item): item is number => item != null)
  const averageConfidence = average(confidenceValues)
  const shadowSamples = normalized.filter((item) => item.mode === 'shadow' && (item.hitTop3 || item.missedTop3))
  const shadowHitCount = shadowSamples.filter((item) => item.hitTop3).length
  const shadowHitRate = shadowSamples.length ? (shadowHitCount / shadowSamples.length) * 100 : null
  const widenedCount = normalized.filter((item) => item.lowConfidenceWidened).length
  const avgDocumentCount = average(normalized.map((item) => item.candidateDocumentCount))
  const avgTopicCount = average(normalized.map((item) => item.candidateTopicCount))
  const avgScopeCount = average(normalized.map((item) => item.candidateScopeCount))
  const uniqueTopDocuments = new Set(
    normalized
      .map((item) => item.topDocument?.documentId || item.topDocument?.documentName || '')
      .filter(Boolean)
  )
  const successRate = normalized.length ? (successCount / normalized.length) * 100 : null
  const lowConfidenceRate = normalized.length ? ((lowConfidenceCount + failedCount) / normalized.length) * 100 : null

  return {
    total: normalized.length,
    autoCount,
    shadowCount,
    successCount,
    lowConfidenceCount,
    failedCount,
    highConfidenceCount,
    widenedCount,
    uniqueTopDocumentCount: uniqueTopDocuments.size,
    averageConfidenceText: averageConfidence == null ? '-' : averageConfidence.toFixed(4),
    averageDocumentCountText: avgDocumentCount == null ? '-' : avgDocumentCount.toFixed(1),
    averageTopicCountText: avgTopicCount == null ? '-' : avgTopicCount.toFixed(1),
    averageScopeCountText: avgScopeCount == null ? '-' : avgScopeCount.toFixed(1),
    successRateText: successRate == null ? '-' : `${successRate.toFixed(1)}%`,
    lowConfidenceRateText: lowConfidenceRate == null ? '-' : `${lowConfidenceRate.toFixed(1)}%`,
    shadowHitRateText: shadowHitRate == null ? '-' : `${shadowHitRate.toFixed(1)}%`
  }
}

export function buildTopDocumentDistribution(records: RouteTraceRecord[] = []): DocumentDistributionItem[] {
  const rows = records
    .map(normalizeRouteTrace)
    .filter((item) => item.topDocument)
    .reduce((map, item) => {
      const documentId = item.topDocument.documentId || item.topDocument.documentName || 'unknown'
      const existing = map.get(documentId) || {
        documentId,
        documentName: item.topDocument.documentName || item.topDocument.documentId || '未知文档',
        count: 0,
        confidenceTotal: 0,
        confidenceCount: 0,
        lowConfidenceCount: 0
      }
      existing.count += 1
      if (item.confidenceNumber != null) {
        existing.confidenceTotal += item.confidenceNumber
        existing.confidenceCount += 1
      }
      if (item.statusKey !== 'SUCCESS') {
        existing.lowConfidenceCount += 1
      }
      map.set(documentId, existing)
      return map
    }, new Map<string, DocumentDistributionItem>())

  return [...rows.values()]
    .map((item) => ({
      ...item,
      averageConfidenceText: item.confidenceCount ? (item.confidenceTotal / item.confidenceCount).toFixed(4) : '-'
    }))
    .sort((left, right) => right.count - left.count)
    .slice(0, 6)
}
