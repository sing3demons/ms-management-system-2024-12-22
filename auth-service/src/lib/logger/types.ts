export interface ConfigLog {
  size?: number
  path: string
  console: boolean
  file: boolean
  rawData?: boolean
}

export interface LogConfig {
  projectName?: string
  namespace?: string
  summary: ConfigLog
  detail: ConfigLog
}

export interface LogData {}

export interface InputOutput {
  Invoke: string
  Event: string
  Protocol?: string
  Type: string
  RawData?: any
  Data?: any
  ResTime?: string
}

export type IDetailLog = {
  LogType: string
  Host: string
  AppName: string
  Instance: number
  Session: string
  InitInvoke: string
  Scenario: string | null
  InputTimeStamp: string | null
  Input: InputOutput[]
  OutputTimeStamp: string | null
  Output: InputOutput[]
  ProcessingTime: string | null
}

export interface MaskingConfig {
  highlight?: string[]
  mark?: string[]
}


export type SummaryResult = { resultCode: string; resultDesc: string; count?: number }

export interface BlockDetail {
  node: string
  cmd: string
  result: SummaryResult[]
  count: number
}

export interface OptionalFields {
  [key: string]: any
}
