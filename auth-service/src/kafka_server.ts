import { Kafka, KafkaMessage, Consumer, Producer, RecordMetadata, Message } from 'kafkajs'
import generateInternalTid from './lib/utils/generateInternalTid.js'
import { DetailLog, SummaryLog } from './lib/logger/index.js'
import NODE_NAME from './lib/constants/nodeName.js'

type ServerKafkaOptions = {
  client?: {
    clientId?: string
    brokers?: string[]
    logLevel?: number
    retry?: {
      initialRetryTime?: number
      retries?: number
    }
  }
  consumer?: {
    groupId?: string
    [key: string]: unknown
  }
  producer?: Record<string, unknown>
  parser?: unknown
  subscribe?: Record<string, unknown>
  run?: Record<string, unknown>
  send?: Record<string, unknown>
  postfixId?: string
}

type KafkaContext = {
  topic: string
  partition: number
  headers: KafkaMessage['headers']
  value: string | null
  commonLog: (
    scenario: string,
    identity?: string
  ) => {
    detailLog: DetailLog
    summaryLog: SummaryLog
    initInvoke: string
  }
  sendMessage: (
    topic: string,
    payload: any
  ) => Promise<{
    err: boolean
    result_desc: string
    result_data: never[] | RecordMetadata[]
  }>
}

export type MessageHandler = (data: unknown, context: KafkaContext) => Promise<unknown>

type ParsedMessage = {
  topic: string
  partition: number
  headers: KafkaMessage['headers']
  value: string | null
}

export class ServerKafka {
  private client: Kafka | null = null
  private consumer: Consumer | null = null
  private producer: Producer | null = null
  private brokers: string[]
  private clientId: string
  private groupId: string
  private options: ServerKafkaOptions
  private messageHandlers: Map<string, MessageHandler> = new Map()

  constructor(options: ServerKafkaOptions) {
    this.options = options
    const clientOptions = options.client || {}
    const consumerOptions = options.consumer || {}
    const postfixId = options.postfixId ?? '-server'

    this.brokers = clientOptions.brokers || ['localhost:9092']
    this.clientId = (clientOptions.clientId || 'kafka-client') + postfixId
    this.groupId = (consumerOptions.groupId || 'kafka-group') + postfixId
  }

  async listen(callback: (err?: Error) => void): Promise<void> {
    try {
      this.client = this.createClient()
      await this.start(callback)
    } catch (err) {
      callback(err as Error)
    }
  }

  async close(): Promise<void> {
    if (this.consumer) await this.consumer.disconnect()
    if (this.producer) await this.producer.disconnect()
    this.consumer = null
    this.producer = null
    this.client = null
  }

  private async start(callback: () => void): Promise<void> {
    const consumerOptions = { ...this.options.consumer, groupId: this.groupId }
    this.consumer = this.client!.consumer(consumerOptions)
    this.producer = this.client!.producer(this.options.producer)

    await this.consumer.connect()
    await this.producer.connect()

    await this.bindEvents(this.consumer)

    callback()
  }

  private createClient(): Kafka {
    return new Kafka({
      clientId: this.clientId,
      brokers: this.brokers,
      logLevel: 1,
      retry: {
        initialRetryTime: 300,
        retries: 8,
      },
      ...this.options.client,
    })
  }

  private async bindEvents(consumer: Consumer): Promise<void> {
    const registeredPatterns = [...this.messageHandlers.keys()]
    const consumerSubscribeOptions = this.options.subscribe || {}

    if (registeredPatterns.length > 0) {
      await consumer.subscribe({
        ...consumerSubscribeOptions,
        topics: registeredPatterns,
      })
    }

    const consumerRunOptions = {
      ...this.options.run,
      eachMessage: this.getMessageHandler(),
    }

    await consumer.run(consumerRunOptions)
  }

  private getMessageHandler() {
    return async (payload: {
      topic: string
      partition: number
      message: KafkaMessage
      //   heartbeat: () => Promise<void>
    }): Promise<void> => {
      await this.handleMessage(payload)
    }
  }

  private async handleMessage(payload: {
    topic: string
    partition: number
    message: KafkaMessage
    // heartbeat: () => Promise<void>
  }): Promise<void> {
    const { topic, message } = payload

    const rawMessage: ParsedMessage = {
      topic,
      partition: payload.partition,
      headers: message.headers,
      value: message.value?.toString() || null,
    }

    const ctx = {
      initInvoke: '' as string,
      detailLog: {} as DetailLog,
      summaryLog: {} as SummaryLog,
      scenario: '',
    }

    const processLogReq = () => {
      try {
        const msg = JSON.parse(rawMessage.value || '')
        return {
          headers: msg.header,
          body: msg.body,
        }
      } catch (error) {
        return {
          headers: rawMessage.headers,
          body: rawMessage.value,
        }
      }
    }

    const processLog = processLogReq()

    const kafkaContext: KafkaContext = {
      ...rawMessage,
      commonLog: (scenario: string, identity?: string) => {
        const session = processLog.headers.session || generateInternalTid(scenario, '-', 22)
        const initInvoke = generateInternalTid(scenario, '-', 20)
        const detailLog = new DetailLog(session, initInvoke, scenario, identity)
        const summaryLog = new SummaryLog(session, initInvoke, scenario)

        detailLog.addInputRequest(NODE_NAME.KAFKA_PRODUCER, scenario, initInvoke, {
          header: processLog.headers,
          body: processLog.body,
        })

        ctx.initInvoke = initInvoke
        ctx.detailLog = detailLog
        ctx.summaryLog = summaryLog
        ctx.scenario = scenario

        return {
          detailLog,
          summaryLog,
          initInvoke,
        }
      },
      sendMessage: async (topic: string, payload: any) => await this.sendMessage(topic, payload, ctx),
    }

    const handler = this.getHandlerByPattern(rawMessage.topic)

    if (!handler) {
      console.log('handler not found')
      return
    }

    const response = (await handler(rawMessage.value, kafkaContext)) as any
    if (!ctx.detailLog?.isEnd()) {
      ctx.detailLog.addOutputRequest(NODE_NAME.KAFKA_CONSUMER, ctx.scenario, ctx.initInvoke, '', response).end()
    }

    if (!ctx.summaryLog?.isEnd()) {
      ctx.summaryLog.end('', 'success')
    }
  }

  parse(message: KafkaMessage): { pattern: string; data: unknown } | {} {
    try {
      const value = message.value?.toString()
      return value ? JSON.parse(value) : {}
    } catch (err) {
      console.error('Failed to parse message', err)
      return {}
    }
  }

  private async sendMessage(
    topic: string,
    payload: any,
    ctx: {
      initInvoke: string
      detailLog: DetailLog
      summaryLog: SummaryLog
      scenario: string
    }
  ) {
    let producer = this.producer
    if (!producer) {
      producer = this.client?.producer(this.options.producer) ?? null
    }

    if (!producer) {
      return {
        err: true,
        result_desc: 'Failed to connect to Kafka',
        result_data: [],
      }
    }

    await producer.connect()

    const messages: Message[] = []
    const invokeKafka = generateInternalTid('kafka', '-', 20)

    if (typeof payload === 'object') {
      if (Array.isArray(payload)) {
        payload.forEach((msg) => {
          messages.push({ value: JSON.stringify(msg) })
        })
      } else {
        messages.push({ value: JSON.stringify(payload) })
      }
    } else {
      messages.push({ value: payload })
    }

    const processReqLog = {
      Body: {
        topic,
        messages: messages.map(({ value }) => {
          try {
            if (typeof value === 'string') {
              return JSON.parse(value)
            }
            return value
          } catch (error) {
            return value
          }
        }),
      },
      RawData: JSON.stringify(payload),
    }

    ctx.detailLog.addOutputRequest(NODE_NAME.KAFKA_PRODUCER, topic, invokeKafka, processReqLog.RawData, processReqLog.Body)
    ctx.detailLog.end()

    try {
      const recordMetadata = await producer.send({
        topic,
        messages: messages.map((msg) => {
          try {
            return JSON.parse(String(msg))
          } catch (error) {
            return msg
          }
        }),
      })

      const result = {
        err: false,
        result_desc: 'Success',
        result_data: recordMetadata,
      }

      ctx.summaryLog.addSuccessBlock(NODE_NAME.KAFKA_PRODUCER, topic, '200', 'success')
      ctx.detailLog.addInputResponse(NODE_NAME.KAFKA_PRODUCER, topic, invokeKafka, '', result)

      return result
    } catch (error) {
      console.error('Failed to send message to Kafka=============> ', error)
      let errMessage = 'Failed to send message to Kafka'

      const result = {
        err: true,
        result_desc: errMessage,
        result_data: [],
      }

      if (error instanceof Error) {
        errMessage = error.message
        result.result_desc = errMessage
      }

      ctx.summaryLog.addErrorBlock(NODE_NAME.KAFKA_PRODUCER, topic, '500', errMessage)
      ctx.detailLog.addInputResponse(NODE_NAME.KAFKA_PRODUCER, topic, invokeKafka, '', result)

      return result
    } finally {
      await producer.disconnect()
    }
  }

  private getHandlerByPattern(pattern: string): MessageHandler | undefined {
    return this.messageHandlers.get(pattern)
  }

  public addHandler(pattern: string, handler: MessageHandler): void {
    this.messageHandlers.set(pattern, handler)
  }
}
