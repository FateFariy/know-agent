import type { ConversationExchange, ConversationTraceStage, Snapshot, SubQuestion } from '@/types'
import type { StageInspector, StageInspectorSection, StageInspectorTable, TextBlock } from './types'
import JSONbig from 'json-bigint'
import {
  formatAnswerShape,
  formatChannelName,
  formatConfidence,
  formatRelationType,
  formatRetrievalMode,
  formatToolName,
  formatUsageStageName
} from './utils'
import {
  buildRagRetrieveRows,
  buildReferenceDecisionRows,
  formatSubQuestions,
  pushPair,
  stageUsageDetails
} from './builders'

interface StageContent {
  summaryItems: TextBlock[]
  listSections: StageInspectorSection[]
  tableSections: StageInspectorTable[]
  advancedItems: TextBlock[]
}

export function buildTraceStageInspector(stageTrace: ConversationTraceStage | null, exchange: ConversationExchange | null): StageInspector | null {
  if (!stageTrace) {
    return null
  }

  const snapshot = JSONbig({ storeAsString: true }).parse(stageTrace.snapshotJson) as Snapshot

  const {
    summaryItems,
    listSections,
    tableSections,
    advancedItems
  } = handleStageType(stageTrace.stageCode, snapshot, exchange)

  const rawSnapshot = stageTrace.snapshotJson
  if (rawSnapshot) {
    pushPair(advancedItems, '原始阶段快照 JSON', rawSnapshot, { code: true })
  }

  return {
    title: stageTrace.stageName,
    summary: stageTrace.summaryText || '',
    stageState: stageTrace.stageState,
    startTime: stageTrace.startTime,
    endTime: stageTrace.endTime,
    durationMs: stageTrace.durationMs,
    summaryItems,
    listSections: listSections.filter(section => section.items.length > 0),
    tableSections: tableSections.filter((section) => section.rows && section.rows.length > 0),
    advancedItems
  }
}

function handleStageType(
  stageCode: string,
  snapshot: Snapshot,
  exchange: ConversationExchange | null
): StageContent {
  switch (stageCode) {
    case 'MEMORY':
      return handleMemoryStage(snapshot, exchange)
    case 'INTENT':
      return handleIntentStage(snapshot, exchange)
    case 'REWRITE':
      return handleRewriteStage(snapshot, exchange)
    case 'ROUTE':
      return handleRouteStage(snapshot)
    case 'RAG_RETRIEVE':
      return handleRagRetrieveStage(snapshot)
    case 'EVIDENCE_BUDGET':
      return handleEvidenceBudgetStage(snapshot)
    case 'ANSWER_GENERATE':
      return handleAnswerGenerateStage(snapshot, exchange)
    case 'REACT_AGENT':
      return handleReactAgentStage(snapshot)
    case 'RECOMMENDATION':
      return handleRecommendationStage(snapshot, exchange)
    case 'FINALIZE':
      return handleFinalizeStage(snapshot)
    default: {
      const summaryItems: TextBlock[] = []
      pushPair(summaryItems, '阶段摘要', stageCode || '')
      return { summaryItems, listSections: [], tableSections: [], advancedItems: [] }
    }
  }
}

/** 处理 MEMORY 阶段 */
function handleMemoryStage(snapshot: Snapshot, exchange: ConversationExchange | null): StageContent {
  const summaryItems: TextBlock[] = []
  const listSections: StageInspectorSection[] = []
  const advancedItems: TextBlock[] = []

  pushPair(summaryItems, '是否命中长期摘要', snapshot.compressionApplied ? '是' : '否')
  pushPair(summaryItems, '摘要覆盖到的最后一轮', snapshot.coveredExchangeId)
  pushPair(summaryItems, '摘要覆盖轮次', snapshot.coveredExchangeCount)
  pushPair(summaryItems, '累计压缩次数', snapshot.compressionCount)
  pushPair(advancedItems, '长期摘要文本', snapshot.longTermSummary, { code: true })
  pushPair(advancedItems, '最近原文窗口', snapshot.recentTranscript, { code: true })
  pushPair(advancedItems, '回答阶段最近上下文', snapshot.recentQuestionTranscript, { code: true })
  listSections.push({
    label: '这一阶段的模型使用',
    items: stageUsageDetails(exchange, ['summary']),
    ordered: false
  })

  return { summaryItems, listSections, tableSections: [], advancedItems }
}

/** 处理 INTENT 阶段 */
function handleIntentStage(snapshot: Snapshot, exchange: ConversationExchange | null): StageContent {
  const summaryItems: TextBlock[] = []
  const listSections: StageInspectorSection[] = []

  pushPair(summaryItems, '原始问题', snapshot.originalQuestion)
  pushPair(summaryItems, '关系判定', formatRelationType(snapshot.relationType))
  pushPair(summaryItems, '当前主题', snapshot.resolvedTopic)
  pushPair(summaryItems, '当前面向', snapshot.resolvedFacet)
  pushPair(summaryItems, '信息需求', snapshot.informationNeed)
  pushPair(summaryItems, '答案形态', formatAnswerShape(snapshot.answerShape))
  pushPair(summaryItems, '检索模式', formatRetrievalMode(snapshot.retrievalMode))
  pushPair(summaryItems, '检索查询', snapshot.retrievalQuery)
  pushPair(summaryItems, '置信度', formatConfidence(snapshot.confidence))
  pushPair(summaryItems, '判定理由', snapshot.rationale)
  listSections.push({
    label: '分析时参考的上轮锚点',
    items: snapshot.previousAnchorDescription ? [String(snapshot.previousAnchorDescription)] : [],
    ordered: false
  })
  listSections.push({
    label: '规划出的检索子问题',
    items: snapshot.retrievalSubQuestions || [],
    ordered: true
  })
  listSections.push({
    label: '软章节提示',
    items: snapshot.softSectionHints || [],
    ordered: false
  })
  listSections.push({
    label: '上下文提示词',
    items: snapshot.queryContextHints || [],
    ordered: false
  })
  listSections.push({
    label: '这一阶段的模型使用',
    items: stageUsageDetails(exchange, ['intent']),
    ordered: false
  })

  return { summaryItems, listSections, tableSections: [], advancedItems: [] }
}

/** 处理 REWRITE 阶段 */
function handleRewriteStage(snapshot: Snapshot, exchange: ConversationExchange | null): StageContent {
  const summaryItems: TextBlock[] = []
  const listSections: StageInspectorSection[] = []

  pushPair(summaryItems, '原始问题', exchange?.question || '')
  pushPair(summaryItems, '改写后问题', snapshot.rewriteQuestion)
  pushPair(summaryItems, '改写参考历史', snapshot.historyContext, { code: true })
  pushPair(summaryItems, '参数覆盖', snapshot.rewriteOverrideEnabled === true ? '已启用' : '未启用')
  pushPair(summaryItems, 'Temperature', snapshot.rewriteTemperature)
  pushPair(summaryItems, 'TopP', snapshot.rewriteTopP)
  pushPair(summaryItems, 'Thinking', String(snapshot.rewriteThinking) || '')
  listSections.push({
    label: '改写拆分出的子问题',
    items: (snapshot?.subQuestions as string[] || []),
    ordered: true
  })
  listSections.push({
    label: '这一阶段的模型使用',
    items: stageUsageDetails(exchange, ['rewrite']),
    ordered: false
  })

  return { summaryItems, listSections, tableSections: [], advancedItems: [] }
}

/** 处理 ROUTE 阶段 */
function handleRouteStage(snapshot: Snapshot): StageContent {
  const summaryItems: TextBlock[] = []
  const listSections: StageInspectorSection[] = []

  pushPair(summaryItems, '原始问题', snapshot.originalQuestion)
  pushPair(summaryItems, '最终执行路径', formatRetrievalMode(snapshot.executionMode || ''))
  pushPair(summaryItems, '最终检索问题', snapshot.retrievalQuestion)
  pushPair(summaryItems, '根主题', snapshot.rootTopic)
  pushPair(summaryItems, '根章节编码', snapshot.rootSectionCode)
  pushPair(summaryItems, '根章节标题', snapshot.rootSectionTitle)
  pushPair(summaryItems, '目标章节提示', snapshot.targetSectionHint)
  pushPair(summaryItems, '是否使用锚点', snapshot.anchorApplied ? '是' : '否')
  listSections.push({
    label: '最终检索子问题',
    items: snapshot.retrievalSubQuestions || [],
    ordered: true
  })

  return { summaryItems, listSections, tableSections: [], advancedItems: [] }
}

/** 处理 RAG_RETRIEVE 阶段 */
function handleRagRetrieveStage(snapshot: Snapshot): StageContent {
  const summaryItems: TextBlock[] = []
  const listSections: StageInspectorSection[] = []
  const tableSections: StageInspectorTable[] = []
  const subQuestions: SubQuestion[] = snapshot.subQuestions as SubQuestion[] || []

  pushPair(summaryItems, '实际检索问题', snapshot.retrievalQuestion)
  pushPair(summaryItems, '最终证据条数', snapshot.referenceCount)
  pushPair(summaryItems, '子问题数量', snapshot.subQuestionCount)

  listSections.push({
    label: '使用通道',
    items: snapshot.usedChannels?.map(formatChannelName) || [],
    ordered: false
  })
  listSections.push({
    label: '检索过程说明',
    items: snapshot.retrievalNotes || [],
    ordered: false
  })

  listSections.push({
    label: '子问题检索明细',
    items: formatSubQuestions(subQuestions) || [],
    ordered: false
  })
  listSections.push({
    label: '最终证据概览',
    items: snapshot.references?.map((item) => {
      return `[${item.referenceId || '-'}] ${item.documentName || '未命名引用'} ${item.sectionPath ? `| ${item.sectionPath}` : ''} ${item.channel ? `| ${formatChannelName(item.channel as string)}` : ''}`.trim()
    }).filter(Boolean) || [],
    ordered: false
  })
  tableSections.push({
    label: '子问题检索链路',
    columns: ['子问题', '关键词 raw/accepted', '向量 raw/accepted', '融合', '父块', '重排', '最终引用'],
    rows: buildRagRetrieveRows(subQuestions)
  })
  tableSections.push({
    label: '最终证据表',
    columns: ['引用', '文档', '章节', '通道'],
    rows: snapshot.references?.map((item) => {
      return {
        cells: [
          String(item.referenceId || '-'),
          String(item.documentName || '未命名引用'),
          String(item.sectionPath || '未识别章节'),
          formatChannelName(item.channel)
        ]
      }
    }) || []
  })

  return { summaryItems, listSections, tableSections, advancedItems: [] }
}

/** 处理证据预算阶段 */
function handleEvidenceBudgetStage(snapshot: Snapshot): StageContent {
  const summaryItems: TextBlock[] = []
  const listSections: StageInspectorSection[] = []
  const tableSections: StageInspectorTable[] = []
  const advancedItems: TextBlock[] = []

  pushPair(summaryItems, '总预算', snapshot.totalBudget)
  pushPair(summaryItems, '单子问题预算', snapshot.perSubQuestionBudget)
  pushPair(summaryItems, '实际渲染引用', snapshot.renderedReferenceCount)
  pushPair(summaryItems, '被省略引用', snapshot.omittedReferenceCount)
  listSections.push({
    label: '已纳入 Prompt 的引用',
    items: snapshot.renderedReferenceDetails || [],
    ordered: false
  })
  listSections.push({
    label: '因预算被省略的引用',
    items: snapshot.omittedReferenceDetails || [],
    ordered: false
  })
  tableSections.push({
    label: '保留到 Prompt 的引用',
    columns: ['引用', '结果'],
    rows: buildReferenceDecisionRows(snapshot.renderedReferenceDetails).map((item) => ({
      cells: [item.reference, item.reason || '已纳入 Prompt']
    }))
  })
  tableSections.push({
    label: '因预算被裁掉的引用',
    columns: ['引用', '原因'],
    rows: buildReferenceDecisionRows(snapshot.omittedReferenceDetails).map((item) => ({
      cells: [item.reference, item.reason || '超出上下文预算']
    }))
  })
  pushPair(advancedItems, '系统 Prompt', snapshot.systemPrompt, { code: true })
  pushPair(advancedItems, '用户 Prompt', snapshot.userPrompt, { code: true })

  return { summaryItems, listSections, tableSections, advancedItems }
}

/** 处理 ANSWER_GENERATE 阶段 */
function handleAnswerGenerateStage(snapshot: Snapshot, exchange: ConversationExchange | null): StageContent {
  const summaryItems: TextBlock[] = []
  const listSections: StageInspectorSection[] = []
  const advancedItems: TextBlock[] = []

  pushPair(summaryItems, '首包耗时', snapshot.firstResponseTimeMs ? `${snapshot.firstResponseTimeMs} ms` : '')
  pushPair(summaryItems, '回答长度', snapshot.answerLength)
  pushPair(advancedItems, '本轮回答全文', exchange?.answer || '', { code: true })
  listSections.push({
    label: '这一阶段的模型使用',
    items: stageUsageDetails(exchange, ['rag_answer', 'react_agent_turn']),
    ordered: false
  })

  return { summaryItems, listSections, tableSections: [], advancedItems }
}

/** 处理 REACT_AGENT 阶段 */
function handleReactAgentStage(snapshot: Snapshot): StageContent {
  const summaryItems: TextBlock[] = []
  const listSections: StageInspectorSection[] = []

  pushPair(summaryItems, '使用组件数', snapshot.usedTools?.length)
  listSections.push({
    label: '使用组件',
    items: snapshot.usedTools?.map(formatToolName) || [],
    ordered: false
  })

  return { summaryItems, listSections, tableSections: [], advancedItems: [] }
}

/** 处理 RECOMMENDATION 阶段 */
function handleRecommendationStage(snapshot: Snapshot, exchange: ConversationExchange | null): StageContent {
  const summaryItems: TextBlock[] = []
  const listSections: StageInspectorSection[] = []

  pushPair(summaryItems, '推荐问题数量', snapshot.recommendationCount)
  listSections.push({
    label: '推荐问题列表',
    items: snapshot.recommendations || [],
    ordered: true
  })
  listSections.push({
    label: '这一阶段的模型使用',
    items: stageUsageDetails(exchange, ['recommendation']),
    ordered: false
  })

  return { summaryItems, listSections, tableSections: [], advancedItems: [] }
}

/** 处理 FINALIZE 阶段 */
function handleFinalizeStage(snapshot: Snapshot): StageContent {
  const summaryItems: TextBlock[] = []

  pushPair(summaryItems, '最终状态', snapshot.finalStatus)
  pushPair(summaryItems, '回答长度', snapshot.answerLength)
  pushPair(summaryItems, '引用数', snapshot.referenceCount)
  pushPair(summaryItems, '推荐问题数', snapshot.recommendationCount)
  pushPair(summaryItems, '结束原因', snapshot.reason || snapshot.errorMessage)

  return { summaryItems, listSections: [], tableSections: [], advancedItems: [] }
}

/** 构建模型使用阶段检查器 */
export function buildUsageStageInspector(exchange: ConversationExchange | null): StageInspector | null {
  if (!exchange) {
    return null
  }

  const usageTraces = exchange.debugTrace?.modelUsageTraces || []
  const limitStats = exchange.debugTrace?.limitStats || null
  const totalPromptTokens = usageTraces.reduce((sum, item) => sum + item?.promptTokens || 0, 0)
  const totalCompletionTokens = usageTraces.reduce((sum, item) => sum + item?.completionTokens || 0, 0)
  const totalTokens = usageTraces.reduce((sum, item) => sum + item?.totalTokens || 0, 0)
  const totalCost = usageTraces.reduce((sum, item) => sum + item?.estimatedCost || 0, 0)

  const rows = usageTraces.map((item) => ({
    cells: [
      formatUsageStageName(item.stageName),
      `${item.provider || 'unknown'} / ${item.model || 'unknown'}`,
      String(item.promptTokens ?? 0),
      String(item.completionTokens ?? 0),
      String(item.totalTokens ?? 0),
      item.estimatedCost ? `¥ ${Number(item.estimatedCost).toFixed(4)}` : '无',
      item.durationMs ? `${item.durationMs} ms` : '无',
      item.status || 'UNKNOWN'
    ]
  }))

  return {
    title: '模型使用与限制',
    summary: '这一轮里每一次模型调用都按阶段分组列在下面，便于排查到底哪个阶段最耗 token 和成本。',
    stageState: limitStats?.limitTriggered ? 'WARNING' : 'COMPLETED',
    startTime: exchange.createTime,
    endTime: exchange.updateTime || '',
    durationMs: exchange.totalResponseTimeMs,
    summaryItems: [
      {
        label: '模型调用次数',
        value: String(usageTraces.length)
      },
      {
        label: '输入 Token',
        value: String(totalPromptTokens)
      },
      {
        label: '输出 Token',
        value: String(totalCompletionTokens)
      },
      {
        label: '总 Token',
        value: String(totalTokens)
      },
      {
        label: '总成本',
        value: totalCost > 0 ? `¥ ${totalCost.toFixed(4)}` : '无'
      },
      {
        label: '模型运行上限',
        value: limitStats?.modelCallsRunLimit != null ? `${limitStats.modelCallsUsed || 0}/${limitStats.modelCallsRunLimit}` : ''
      },
      {
        label: '工具运行上限',
        value: limitStats?.toolCallsRunLimit != null ? `${limitStats.toolCallsUsed || 0}/${limitStats.toolCallsRunLimit}` : ''
      },
      {
        label: '限制触发',
        value: limitStats?.limitTriggered ? (limitStats.limitReason || '已触发') : '未触发'
      }
    ],
    listSections: [],
    tableSections: rows.length ? [{
      label: '按阶段分组的模型使用明细',
      columns: ['阶段', '模型', '输入 Token', '输出 Token', '总 Token', '成本', '耗时', '状态'],
      rows
    }] : [],
    advancedItems: [
      limitStats?.modelCallsThreadLimit != null
        ? { label: '线程级模型上限', value: String(limitStats.modelCallsThreadLimit) }
        : null,
      limitStats?.toolCallsThreadLimit != null
        ? { label: '线程级工具上限', value: String(limitStats.toolCallsThreadLimit) }
        : null
    ].filter((item): item is TextBlock => Boolean(item))
  }
}
