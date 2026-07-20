export interface Response<T> {
  code: number
  msg: string
  data?: T
}

export * from './document'
export * from './knowledge'
export * from './chat'
