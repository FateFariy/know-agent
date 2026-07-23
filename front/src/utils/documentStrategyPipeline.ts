import type {DocumentStrategyPlan, DocumentStrategyStep, StrategyStepItem} from '@/types'

export interface StrategyItem {
  type: number
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

export const STRATEGY_LIBRARY: StrategyItem[] = [
  {
    type: 1,
    label: '基于文档结构切块',
    description: '优先保留标题和章节边界'
  },
  {
    type: 2,
    label: '递归分块',
    description: '对超长内容继续裁剪兜底'
  },
  {
    type: 3,
    label: '语义分块',
    description: '优化主题边界和段落完整性'
  },
  {
    type: 4,
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

export function normalizeStrategyTypeList(selectedTypes: number[], strategyLibrary: StrategyItem[] = STRATEGY_LIBRARY): number[] {
  const seen = new Set<number>()
  const availableTypes = new Set(strategyLibrary.map((item) => item.type))
  const orderedTypes: number[] = []
  selectedTypes.forEach((item) => {
    if (item && !seen.has(item) && availableTypes.has(item)) {
      seen.add(item)
      orderedTypes.push(item)
    }
  })

  return orderedTypes
}

export function buildStrategyPreview(selectedTypes: number[], strategyLibrary: StrategyItem[] = STRATEGY_LIBRARY): StrategyPreviewItem[] {
  const typeMap = new Map<number, StrategyItem>();
  strategyLibrary.forEach(item => typeMap.set(item.type, item))

  const typeList = normalizeStrategyTypeList(selectedTypes, strategyLibrary)

  return typeList.map((type, index) => {
    const strategy = typeMap.get(type)!;
    const order = String(index + 1).padStart(2, '0')
    return { ...strategy, index, order }
  })
}

export function buildStrategySignature(selectedTypes: number[], strategyLibrary: StrategyItem[] = STRATEGY_LIBRARY): string {
  return normalizeStrategyTypeList(selectedTypes, strategyLibrary).join('|')
}

export function resolveStrategySteps(plan: DocumentStrategyPlan | null | undefined, pipelineKey: string | null | undefined): DocumentStrategyStep[] {
  if (!plan || !pipelineKey) {
    return []
  }
  return pipelineKey === 'parent' ? plan.parentPipeline?.steps ?? [] : plan.childPipeline?.steps ?? []
}

export function extractPipelineStrategyTypes(plan: DocumentStrategyPlan | null | undefined, pipelineKey: string | null | undefined,
                                             strategyLibrary: StrategyItem[] = STRATEGY_LIBRARY): number[] {
  const steps = resolveStrategySteps(plan, pipelineKey)
  const types = steps.map((item) => item.strategyType)
  return normalizeStrategyTypeList(types, strategyLibrary)
}

export function buildPipelineStepPayload(selectedTypes: number[], strategyLibrary: StrategyItem[] = STRATEGY_LIBRARY): StrategyStepItem[] {
  const previewItems = buildStrategyPreview(selectedTypes, strategyLibrary)
  return previewItems.map((item, index) => ({
    stepNo: index + 1,
    strategyType: item.type
  }))
}
