/// <reference types="vite/client" />

interface ObsDetailState {
  loadingPage: boolean
  hasSession: boolean
  hasExchangeDetail: boolean
  conversationId: string
  exchangeId: string
  selectedTraceId: string
  traceDetailOpen: boolean
  overlayTitle: string
}

interface Window {
  __obsDetailState?: ObsDetailState
}

