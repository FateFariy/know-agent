export interface Response<T> {
  code: number
  msg: string
  data?: T
}

export * from './request'
export * from './response'
