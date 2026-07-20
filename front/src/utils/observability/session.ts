import type {ConversationExchange, ConversationSessionResp} from '@/types'
import {truncate} from './utils'

function latestExchangeQuestion(session: ConversationSessionResp | undefined): string {
  const exchanges = session?.exchanges ?? []
  for (let index = exchanges.length - 1; index >= 0; index -= 1) {
    if (exchanges[index]?.question) {
      return exchanges[index]?.question || ''
    }
  }
  return ''
}

function latestExchangeAnswer(session: ConversationSessionResp | undefined): string {
  const exchanges = session?.exchanges ?? []
  for (let index = exchanges.length - 1; index >= 0; index -= 1) {
    if (exchanges[index]?.answer) {
      return exchanges[index]?.answer || ''
    }
  }
  return ''
}

export function sessionTitle(session: ConversationSessionResp | undefined): string {
  const latestUserMessage = session?.latestUserMessage || latestExchangeQuestion(session)
  const latestAssistantMessage = session?.latestAssistantMessage || latestExchangeAnswer(session)
  return truncate(latestUserMessage || latestAssistantMessage || '未命名会话', 28)
}

export function sessionPreview(session: ConversationSessionResp | undefined): string {
  const latestAssistantMessage = session?.latestAssistantMessage || latestExchangeAnswer(session)
  const latestUserMessage = session?.latestUserMessage || latestExchangeQuestion(session)
  return truncate(latestAssistantMessage || latestUserMessage || '暂无内容', 72)
}

export function sessionMessageCount(session: ConversationSessionResp | undefined): number {
  if (session?.messageCount) {
    return session.messageCount
  }
  const exchanges = session?.exchanges ?? []
  return exchanges.reduce((count, exchange) => {
    let num = count
    if (exchange.question) num++
    if (exchange.answer) num++
    return num
  }, 0)
}

export function listAssistantExchanges(session: ConversationSessionResp | null): ConversationExchange[] {
  return (session?.exchanges ?? []).filter((item) => item && item.turnStatus)
}
