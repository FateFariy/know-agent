import { normalizeCode } from './manageFormat'

export interface StrategyItem {
  type: string
  label: string
  description: string
}

export interface StrategyPipelineItem {
  key: string
  code: string
  label: string
  description: string
}

export interface StrategyPreviewItem extends StrategyItem {
  index: number
  order: string
}

export interface PipelineStep {
  stepNo: string
  strategyType: string
}

export interface PipelineSteps {
  steps: { strategyType: string }[]
}

export interface DocumentStrategyPlan {
  parentPipeline?: PipelineSteps | null
  childPipeline?: PipelineSteps | null
}

export const STRATEGY_LIBRARY: StrategyItem[] = [
  {
    type: '1',
    label: '基于文档结构切块',
    description: '优先保留标题和章节边界'
  },
  {
    type: '2',
    label: '递归分块',
    description: '对超长内容继续裁剪兜底'
  },
  {
    type: '3',
    label: '语义分块',
    description: '优化主题边界和段落完整性'
  },
  {
    type: '4',
    label: '大模型智能切块',
    description: '处理复杂内容和低质量文本'
  }
]

export const STRATEGY_PIPELINE_LIBRARY: StrategyPipelineItem[] = [
  {
    key: 'parent',
    code: 'PARENT',
    label: '父块流水线',
    description: '决定回答阶段看到的父块边界'
  },
  {
    key: 'child',
    code: 'CHILD',
    label: '子块流水线',
    description: '决定检索召回使用的子块边界'
  }
]

export function normalizeStrategyTypeList(selectedTypes: unknown[], strategyLibrary: StrategyItem[] = STRATEGY_LIBRARY): string[] {
  const seen = new Set<string>()
  const availableTypes = new Set(strategyLibrary.map((item) => item.type))
  const orderedTypes: string[] = [];
  (selectedTypes ?? []).forEach((item) => {
    const strategyType = normalizeCode(item)
    if (!strategyType || seen.has(strategyType) || !availableTypes.has(strategyType)) {
      return
    }
    seen.add(strategyType)
    orderedTypes.push(strategyType)
  })

  return orderedTypes
}

export function buildStrategyPreview(
  selectedTypes: unknown[],
  strategyLibrary: StrategyItem[] = STRATEGY_LIBRARY
): StrategyPreviewItem[] {
  return normalizeStrategyTypeList(selectedTypes, strategyLibrary)
    .map((type, index) => {
      const strategy = strategyLibrary.find((item) => item.type === type)
      return strategy ? { ...strategy, index, order: String(index + 1).padStart(2, '0') } : null
    })
    .filter((item): item is StrategyPreviewItem => item !== null)
}

export function buildStrategySignature(
  selectedTypes: unknown[],
  strategyLibrary: StrategyItem[] = STRATEGY_LIBRARY
): string {
  return normalizeStrategyTypeList(selectedTypes, strategyLibrary).join('|')
}

export function resolvePlanPipeline(plan: DocumentStrategyPlan | null | undefined, pipelineKey: string | null | undefined): PipelineSteps | null {
  if (!plan || !pipelineKey) {
    return null
  }
  return pipelineKey === 'parent' ? plan.parentPipeline ?? null : plan.childPipeline ?? null
}

export function extractPipelineStrategyTypes(plan: DocumentStrategyPlan | null | undefined, pipelineKey: string | null | undefined,
                                             strategyLibrary: StrategyItem[] = STRATEGY_LIBRARY): string[] {
  const pipeline = resolvePlanPipeline(plan, pipelineKey)
  return Array.isArray(pipeline?.steps)
    ? normalizeStrategyTypeList(pipeline.steps.map((item) => item.strategyType), strategyLibrary)
    : []
}

export function buildPipelineStepPayload(
  selectedTypes: unknown[],
  strategyLibrary: StrategyItem[] = STRATEGY_LIBRARY
): PipelineStep[] {
  return buildStrategyPreview(selectedTypes, strategyLibrary).map((item, index) => ({
    stepNo: String(index + 1),
    strategyType: item.type
  }))
}
