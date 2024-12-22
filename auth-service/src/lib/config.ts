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

export interface DbConfig {}
export interface RedisConfig {}
export interface KafkaConfig {}
export interface MailServer {}

export interface IConfig {
  Addr?: string
  Db?: DbConfig
  Env?: string
  Name?: string
  RedisCfg?: RedisConfig
  KafkaCfg?: KafkaConfig
  LogConfig?: LogConfig
  MailServer?: MailServer
}


import 'dotenv/config'

class Config  {
  private config: IConfig
  constructor(config: IConfig) {
    this.config = config
  }
}

export default Config