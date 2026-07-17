import type {
  ChatDebugTrace,
  ChatModelUsageTrace,
  ChatToolTrace,
  ConversationExchange,
  ConversationSessionResp,
  IntentResolution,
  SearchReference
} from '@/types'
import type {ExchangeStage, ListBlock, TextBlock} from './types'
import {
  formatAnswerShape,
  formatChannelName,
  formatChatMode,
  formatConfidence,
  formatExecutionMode,
  formatLatency,
  formatRelationType,
  formatRetrievalMode,
  formatStageStateLabel,
  formatToolName,
  turnStatusTone,
  uniqueStrings
} from './utils'
import {
  buildChips,
  buildMetrics,
  buildOutcomeSummary,
  pushListBlock,
  pushTextBlock
} from './builders'

export function buildExchangeStatusNarrative(exchange: ConversationExchange | undefined): string {
  if (!exchange) {
    return ''
  }
  const trace = exchange.debugTrace
  const intent = trace?.intentResolution || null
  const parts: string[] = [
    `当前查看的是 exchange ${exchange.exchangeId}。`,
    `执行路径是“${formatExecutionMode(trace?.executionMode || '')}”。`
  ]

  if (intent?.relationType) {
    parts.push(`系统把这句判定为“${formatRelationType(intent.relationType)}”。`)
  }
  if (intent?.retrievalMode) {
    parts.push(`检索策略是“${formatRetrievalMode(intent.retrievalMode)}”。`)
  }
  if (exchange.turnStatus === 3 && exchange.errorMessage) {
    parts.push(`当前结束原因：${exchange.errorMessage}`)
  } else if (exchange.turnStatus === 2) {
    parts.push('这轮已经成功完成，默认先看“结果与诊断”和“执行过程”两块。')
  }
  return parts.join(' ')
}

export function buildExchangeStages(session: ConversationSessionResp | null, exchange: ConversationExchange | null): ExchangeStage[] {
  if (!exchange) {
    return []
  }

  const trace = exchange.debugTrace
  const intent = trace?.intentResolution || null
  const references = exchange.references || []
  const recommendations = exchange.recommendations || []
  const thinkingSteps = exchange.thinkingSteps || []
  const retrievalNotes = trace?.retrievalNotes || []
  const modelUsageTraces = trace?.modelUsageTraces || []
  const executionNotes = uniqueStrings([...thinkingSteps, ...retrievalNotes])
  const toolTraces = trace?.toolTraces.map(item => ({
    ...item,
    toolName: formatToolName(item.toolName)
  })) || []

  const requestPrimaryBlocks: TextBlock[] = []
  pushTextBlock(requestPrimaryBlocks, '用户原始问题', trace?.originalQuestion || exchange.question)
  pushTextBlock(requestPrimaryBlocks, '当前日期锚点', trace?.currentDateText)

  const requestAdvancedBlocks: TextBlock[] = []
  pushTextBlock(requestAdvancedBlocks, 'Agent 增强问题', trace?.agentQuestion, {code: true})

  const planningPrimaryBlocks: TextBlock[] = []
  pushTextBlock(planningPrimaryBlocks, '系统理解后的问题', trace?.retrievalQuestion)
  pushTextBlock(planningPrimaryBlocks, '信息需求', intent?.informationNeed)
  pushTextBlock(planningPrimaryBlocks, '判定说明', intent?.rationale)
  pushTextBlock(planningPrimaryBlocks, '检索锚点主问题', trace?.retrievalAnchorResolvedQuestion)

  const planningPrimaryLists: ListBlock[] = []
  if ((trace?.subQuestions || []).length > 1) {
    pushListBlock(planningPrimaryLists, '最终检索子问题', trace?.subQuestions || [], {ordered: true})
  }

  const planningAdvancedBlocks: TextBlock[] = []
  pushTextBlock(planningAdvancedBlocks, 'Rewrite 独立问题', trace?.rewriteQuestion)
  pushTextBlock(planningAdvancedBlocks, '长期摘要', trace?.longTermSummary, {code: true})
  pushTextBlock(planningAdvancedBlocks, '回答承接上下文', trace?.answerHistoryContext, {code: true})
  pushTextBlock(planningAdvancedBlocks, '规划历史摘要', trace?.historySummary, {code: true})
  pushTextBlock(planningAdvancedBlocks, '根主题', trace?.retrievalAnchorRootTopic)
  pushTextBlock(planningAdvancedBlocks, '根章节标题', trace?.retrievalAnchorRootSectionTitle)
  pushTextBlock(planningAdvancedBlocks, '目标章节提示', trace?.retrievalAnchorTargetSectionHint)
  pushTextBlock(planningAdvancedBlocks, '编号项文本', trace?.retrievalAnchorItemText)

  const planningAdvancedLists: ListBlock[] = []
  pushListBlock(planningAdvancedLists, 'Rewrite 子问题拆分', trace?.rewriteSubQuestions, {ordered: true})
  pushListBlock(planningAdvancedLists, '软章节提示', intent?.softSectionHints)
  pushListBlock(planningAdvancedLists, '上下文提示词', intent?.queryContextHints)

  const executionPrimaryLists: ListBlock[] = []
  pushListBlock(executionPrimaryLists, '关键执行节点', executionNotes)

  const executionAdvancedLists: ListBlock[] = []
  pushListBlock(executionAdvancedLists, '原始 thinking 事件', thinkingSteps)
  pushListBlock(executionAdvancedLists, '原始检索/Agent 轨迹', retrievalNotes)

  const generationPrimaryBlocks: TextBlock[] = []
  pushTextBlock(generationPrimaryBlocks, '回答预览', exchange.answer, {code: true})

  const generationAdvancedBlocks: TextBlock[] = []
  pushTextBlock(generationAdvancedBlocks, '系统 Prompt', trace?.ragSystemPrompt, {code: true})
  pushTextBlock(generationAdvancedBlocks, '用户 Prompt', trace?.ragUserPrompt, {code: true})

  const outcomePrimaryBlocks: TextBlock[] = []
  pushTextBlock(outcomePrimaryBlocks, '排障结论', buildOutcomeSummary(exchange, references))
  pushTextBlock(outcomePrimaryBlocks, '结束说明', exchange.errorMessage)

  const outcomeAdvancedLists: ListBlock[] = []
  pushListBlock(outcomeAdvancedLists, '推荐追问', recommendations, {ordered: true})

  const stages: ExchangeStage[] = [
    buildOutcomeStage(exchange, references, recommendations, outcomePrimaryBlocks, outcomeAdvancedLists),
    buildExecutionStage(trace, exchange, executionNotes, toolTraces, executionPrimaryLists, executionAdvancedLists),
    buildPlanningStage(trace, intent, planningPrimaryBlocks, planningPrimaryLists, planningAdvancedBlocks, planningAdvancedLists),
    buildRequestStage(session, trace, exchange, requestPrimaryBlocks, requestAdvancedBlocks),
    buildGenerationStage(exchange, references, generationPrimaryBlocks, generationAdvancedBlocks),
    buildUsageStage(trace, modelUsageTraces)
  ]

  return stages.filter((stage) => {
    return stage.chips?.length
      || stage.metrics?.length
      || stage.textBlocks?.length
      || stage.listBlocks?.length
      || stage.toolTraces?.length
      || stage.references?.length
      || stage.advancedTextBlocks?.length
      || stage.advancedListBlocks?.length
  })
}

function buildOutcomeStage(
  exchange: ConversationExchange,
  references: SearchReference[],
  recommendations: string[],
  textBlocks: TextBlock[],
  advancedListBlocks: ListBlock[]
): ExchangeStage {
  return {
    key: 'outcome',
    eyebrow: '1. 排障结论',
    title: '结果与诊断',
    subtitle: '先看这块，快速判断这轮到底是成功、失败、停止，还是证据不足。',
    tone: turnStatusTone(exchange.turnStatus),
    chips: buildChips(
      {
        label: '最终状态',
        value: formatStageStateLabel(exchange.turnStatus),
        tone: turnStatusTone(exchange.turnStatus)
      },
      {
        label: '引用情况',
        value: references.length ? `${references.length} 条证据` : '未看到最终引用',
        tone: references.length ? 'success' : 'warning'
      }
    ),
    metrics: buildMetrics(
      {label: '最终引用数', value: references.length ? `${references.length}` : '', mono: true},
      {
        label: '推荐追问',
        value: recommendations.length ? `${recommendations.length}` : '',
        mono: true
      }
    ),
    textBlocks,
    listBlocks: [],
    references: references,
    advancedTextBlocks: [],
    advancedListBlocks
  }
}

function buildExecutionStage(
  trace: ChatDebugTrace | null,
  exchange: ConversationExchange,
  executionNotes: string[],
  toolTraces: ChatToolTrace[],
  listBlocks: ListBlock[],
  advancedListBlocks: ListBlock[]
): ExchangeStage {
  const isAgentMode = trace?.executionMode === 'REACT_AGENT'
  return {
    key: 'execution',
    eyebrow: '2. 执行过程',
    title: isAgentMode ? 'Agent 执行' : '检索执行',
    subtitle: isAgentMode
      ? '如果结果不对，先看 Agent 有没有调用工具、工具回来了什么。'
      : '如果结果不对，先看检索通道、执行节点和最终证据组织是否正常。',
    tone: 'warning',
    chips: buildChips(
      ...(trace?.usedChannels || []).map((item) => ({
        label: '使用通道',
        value: formatChannelName(item),
        tone: 'success'
      })),
      ...(exchange?.usedTools || []).map((item) => ({
        label: '使用组件',
        value: formatToolName(item),
        tone: 'warning'
      })),
      trace?.limitStats?.modelCallsRunLimit ? {
        label: 'ModelHook',
        value: `${trace?.limitStats?.modelCallsUsed || 0}/${trace?.limitStats?.modelCallsRunLimit || 0}`,
        tone: trace?.limitStats?.limitTriggered ? 'warning' : 'neutral'
      } : null,
      trace?.limitStats?.toolCallsRunLimit ? {
        label: 'ToolHook',
        value: `${trace?.limitStats?.toolCallsUsed || 0}/${trace?.limitStats?.toolCallsRunLimit || 0}`,
        tone: trace?.limitStats?.limitTriggered ? 'warning' : 'neutral'
      } : null
    ),
    metrics: buildMetrics(
      {
        label: '关键节点数',
        value: executionNotes.length ? String(executionNotes.length) : '',
        mono: true
      },
      {
        label: '工具调用次数',
        value: toolTraces.length ? String(toolTraces.length) : '',
        mono: true
      }
    ),
    textBlocks: [],
    listBlocks,
    toolTraces: toolTraces as unknown as ExchangeStage['toolTraces'],
    advancedTextBlocks: [],
    advancedListBlocks
  }
}

function buildPlanningStage(trace: ChatDebugTrace | null, intent: IntentResolution | null, textBlocks: TextBlock[],
                            listBlocks: ListBlock[], advancedTextBlocks: TextBlock[], advancedListBlocks: ListBlock[]
): ExchangeStage {
  return {
    key: 'planning',
    eyebrow: '3. 系统理解',
    title: '前置编排',
    subtitle: '当你怀疑系统“理解错问题”时，看这块最直接。',
    tone: 'success',
    chips: buildChips(
      {label: '会话关系', value: formatRelationType(intent?.relationType), tone: 'primary'},
      {label: '检索方式', value: formatRetrievalMode(intent?.retrievalMode), tone: 'success'},
      {label: '答案形态', value: formatAnswerShape(intent?.answerShape), tone: 'neutral'},
      {label: '意图置信度', value: formatConfidence(intent?.confidence), tone: 'warning'},
      {
        label: '锚点应用',
        value: trace?.retrievalAnchorApplied ? '已使用锚点' : '未使用锚点',
        tone: trace?.retrievalAnchorApplied ? 'success' : 'neutral'
      }
    ),
    metrics: buildMetrics(
      {
        label: '摘要覆盖轮次',
        value: trace?.historyCoveredExchangeCount != null ? String(trace.historyCoveredExchangeCount) : '',
        mono: true
      },
      {
        label: '摘要压缩次数',
        value: trace?.historyCompressionCount != null ? String(trace.historyCompressionCount) : '',
        mono: true
      }
    ),
    textBlocks,
    listBlocks,
    advancedTextBlocks,
    advancedListBlocks
  }
}

function buildRequestStage(session: ConversationSessionResp | null, trace: ChatDebugTrace | null, exchange: ConversationExchange,
                           textBlocks: TextBlock[], advancedTextBlocks: TextBlock[]
): ExchangeStage {
  return {
    key: 'request',
    eyebrow: '4. 请求边界',
    title: '请求入口',
    subtitle: '确认用户原始问题、模式和文档边界有没有偏掉。',
    tone: 'primary',
    chips: buildChips(
      {
        label: '回答模式',
        value: formatChatMode(trace?.chatMode || session?.chatMode),
        tone: 'primary'
      },
      {
        label: '执行路径',
        value: formatExecutionMode(trace?.executionMode || ''),
        tone: 'success'
      },
      {
        label: '文档范围',
        value: session?.selectedDocumentName || (trace?.selectedDocumentId ? `文档 ${trace.selectedDocumentId}` : ''),
        tone: 'neutral'
      },
      {
        label: '时间解释',
        value: trace?.requiresCurrentDateAnchoring ? '按当前日期解释' : '',
        tone: 'warning'
      },
      {
        label: '实时核实',
        value: trace?.requiresFreshSearch ? '优先核实最新事实' : '',
        tone: 'warning'
      }
    ),
    metrics: buildMetrics(
      {label: '会话ID', value: session?.conversationId || '', mono: true},
      {
        label: '轮次ID',
        value: exchange.exchangeId ? String(exchange.exchangeId) : '',
        mono: true
      }
    ),
    textBlocks,
    listBlocks: [],
    advancedTextBlocks,
    advancedListBlocks: []
  }
}

function buildGenerationStage(exchange: ConversationExchange, references: SearchReference[],
                              textBlocks: TextBlock[], advancedTextBlocks: TextBlock[]
): ExchangeStage {
  return {
    key: 'generation',
    eyebrow: '5. 高级生成',
    title: '生成回答',
    subtitle: '默认只看耗时和回答预览；Prompt 已折叠到高级技术细节里。',
    tone: 'neutral',
    chips: buildChips(
      {
        label: '当前状态',
        value: formatStageStateLabel(exchange.turnStatus),
        tone: turnStatusTone(exchange.turnStatus)
      }
    ),
    metrics: buildMetrics(
      {label: '首包耗时', value: formatLatency(exchange.firstResponseTimeMs), mono: true},
      {label: '总耗时', value: formatLatency(exchange.totalResponseTimeMs), mono: true},
      {label: '引用数', value: references.length ? `${references.length}` : '', mono: true}
    ),
    textBlocks,
    listBlocks: [],
    advancedTextBlocks,
    advancedListBlocks: []
  }
}

function buildUsageStage(trace: ChatDebugTrace | null, modelUsageTraces: ChatModelUsageTrace[]): ExchangeStage {
  return {
    key: 'usage',
    eyebrow: '6. 模型用量',
    title: '模型使用与限制',
    subtitle: '这一块解释这轮回答消耗了多少模型资源，以及是否触发调用限制。',
    tone: 'neutral',
    chips: buildChips(
      trace?.limitStats?.limitTriggered ? {
        label: '限制触发',
        value: trace?.limitStats?.limitReason || '已触发调用限制',
        tone: 'warning'
      } : null
    ),
    metrics: buildMetrics(
      {
        label: '模型调用数',
        value: modelUsageTraces.length ? String(modelUsageTraces.length) : '',
        mono: true
      },
      {
        label: '总 Token',
        value: modelUsageTraces.length
          ? String(modelUsageTraces.reduce((sum, item) => sum + Number(item?.totalTokens || 0), 0))
          : '',
        mono: true
      },
      {
        label: '总成本',
        value: modelUsageTraces.length
          ? `¥ ${modelUsageTraces.reduce((sum, item) => sum + Number(item?.estimatedCost || 0), 0).toFixed(4)}`
          : '',
        mono: true
      }
    ),
    textBlocks: [],
    listBlocks: [],
    advancedTextBlocks: [],
    advancedListBlocks: [
      {
        label: '模型使用清单',
        ordered: false,
        items: modelUsageTraces.map((item) => {
          const tokenText = item?.totalTokens ? `，总Token ${item.totalTokens}` : ''
          const costText = item?.estimatedCost ? `，成本约 ¥${Number(item.estimatedCost).toFixed(4)}` : ''
          const durationText = item?.durationMs ? `，耗时 ${item.durationMs} ms` : ''
          return `${item?.stageName || 'unknown'} | ${item?.provider || 'unknown'} / ${item?.model || 'unknown'}${tokenText}${costText}${durationText}`
        })
      },
      trace?.limitStats?.limitReason ? {
        label: '限制说明',
        ordered: false,
        items: [trace?.limitStats?.limitReason || '']
      } : null
    ].filter((item): item is ListBlock => Boolean(item))
  }
}

export function stageHasAdvancedDetails(stage: ExchangeStage | undefined): boolean {
  if (!stage) {
    return false
  }
  return Boolean(
    stage.advancedTextBlocks?.length
    || stage.advancedListBlocks?.length
    || stage.advancedToolTraces?.length
    || stage.advancedReferences?.length
  )
}
