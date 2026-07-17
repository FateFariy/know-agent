import type { RetrievalResultResp } from '@/types'
import type { GroupedSubQuestion, SubQuestionChannel } from './types'

export function groupResultsBySubQuestion(results: RetrievalResultResp[] | undefined): GroupedSubQuestion[] {
  if (!results || !results.length) {
    return []
  }

  const grouped = new Map<number, {
    index: number
    question: string
    channels: Map<string, SubQuestionChannel>
  }>()

  results.forEach((result) => {
    const index = result.subQuestionIndex || 1
    if (!grouped.has(index)) {
      grouped.set(index, {
        index,
        question: result.subQuestion || `子问题 ${index}`,
        channels: new Map()
      })
    }

    const subQ = grouped.get(index)!
    const channelType = result.channelType || 'unknown'

    if (!subQ.channels.has(channelType)) {
      subQ.channels.set(channelType, {
        type: channelType,
        results: []
      })
    }

    subQ.channels.get(channelType)!.results.push(result)
  })

  return Array.from(grouped.values()).map((subQ) => ({
    index: subQ.index,
    question: subQ.question,
    channels: Array.from(subQ.channels.values())
  }))
}