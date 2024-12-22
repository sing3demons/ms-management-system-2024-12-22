import { Kafka, KafkaMessage, Consumer, Producer, RecordMetadata, Message, KafkaConfig } from 'kafkajs'
import { DetailLog, SummaryLog } from '../logger/index.js'
import generateInternalTid from '../v2/generateInternalTid.js'
import NODE_NAME from '../constants/nodeName.js'
import { Static, TObject, TSchema } from '@sinclair/typebox'
import { TypeCompiler } from '@sinclair/typebox/compiler'
import { SchemaCtx } from '../v2/route.js'
import { v7 as uuid } from 'uuid'

export type ServerKafkaOptions = {
  client?: {
    clientId?: string
    brokers?: string[]
    logLevel?: number
    retry?: {
      initialRetryTime?: number
      retries?: number
    }
    auth?: {
      username: string
      password: string
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

export type KafkaContext<Schema extends SchemaCtx> = {
  topic: string
  partition: number
  headers: KafkaMessage['headers']
  value: string
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
  body: Schema['body'] extends TObject ? Static<Schema['body']> : Record<string, any>
}

type BaseResponse = {
  topic?: string
  status?: number
  data?: any
  success: boolean
  [key: string]: any
}

export type MessageHandler<Schema extends SchemaCtx> = (
  context: KafkaContext<Schema>
) => Promise<BaseResponse> | BaseResponse

type ParsedMessage = {
  topic: string
  partition: number
  headers: KafkaMessage['headers']
  value: string
  body: Record<string, TSchema>
}

class ServerKafkaError extends Error {
  topic?: string
  payload?: any

  constructor({ message, topic, payload }: { message: string; topic?: string; payload?: any }) {
    super(message)
    this.name = 'ServerKafkaError'
    this.topic = topic
    this.payload = payload
  }
}

type ICtxProducer = {
  initInvoke: string
  detailLog: DetailLog
  summaryLog: SummaryLog
  scenario: string
}

export class ServerKafka {
  private client: Kafka | null = null
  private consumer: Consumer | null = null
  private producer: Producer | null = null
  private brokers: string[]
  private clientId: string
  private groupId: string
  private options: ServerKafkaOptions
  private messageHandlers = new Map()
  private schemaHandler = new Map<string, Omit<SchemaCtx, 'query'>>() // topic -> schema

  constructor(options: ServerKafkaOptions) {
    this.options = options
    const clientOptions = options.client || {}
    const consumerOptions = options.consumer || {}
    const postfixId = options.postfixId ?? '-server'

    this.brokers = clientOptions.brokers || ['localhost:9092']
    this.clientId = (clientOptions.clientId || 'kafka-client') + postfixId
    this.groupId = (consumerOptions.groupId || 'kafka-group-2') + postfixId
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

    try {
      await this.consumer.connect()
      await this.producer.connect()

      await this.bindEvents(this.consumer)
    } catch (error) {
      console.error('Failed to connect to Kafka', error)
      await this.close()
    }

    callback()
  }

  private createClient(): Kafka {
    const config: KafkaConfig = {
      clientId: this.clientId,
      brokers: this.brokers,
      logLevel: this.options.client?.logLevel || 1,

      retry: {
        initialRetryTime: this.options.client?.retry?.initialRetryTime || 300,
        retries: this.options.client?.retry?.retries || 8,
      },
      ...this.options.client,
    }

    if (this.options.client?.auth) {
      config.sasl = {
        mechanism: 'plain',
        username: this.options.client.auth.username,
        password: this.options.client.auth.password,
      }
    }

    return new Kafka(config)
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
      heartbeat: () => Promise<void>
    }): Promise<void> => {
      await this.handleMessage(payload)
    }
  }

  private async handleMessage(payload: {
    topic: string
    partition: number
    message: KafkaMessage
    heartbeat: () => Promise<void>
  }): Promise<void> {
    const ctx = {
      initInvoke: '' as string,
      detailLog: {} as DetailLog,
      summaryLog: {} as SummaryLog,
      scenario: '',
    }

    try {
      const { topic, message } = payload
      let session = uuid()

      for (const key in message?.headers) {
        if (Object.prototype.hasOwnProperty.call(message.headers, key)) {
          const element = message.headers[key]
          if (element instanceof Buffer) {
            message.headers[key] = element.toString()
          }
        }
      }

      const rawMessage: ParsedMessage = {
        topic,
        partition: payload.partition,
        headers: message.headers,
        value: message.value?.toString() || '',
        body: JSON.parse(message.value?.toString() || ''),
      }

      const handler = this.getHandlerByPattern(rawMessage.topic)

      if (!handler) {
        return
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

      const schemaCtx = this.schemaHandler.get(topic) || {}

      const kafkaContext: KafkaContext<typeof schemaCtx> = {
        ...rawMessage,
        commonLog: (scenario: string, identity?: string) => {
          const sessionId = processLog?.headers?.session || scenario
          session = sessionId + '-' + session
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

          if (this.schemaHandler.has(topic)) {
            const schema = this.schemaHandler.get(topic)

            if (schema?.body) {
              const parsedBody = JSON.parse(message.value?.toString() || '')
              const typeCheck = TypeCompiler.Compile(schema.body as TSchema)
              if (!typeCheck.Check(parsedBody)) {
                const first = typeCheck.Errors(parsedBody)
                detailLog.addOutputRequest(NODE_NAME.KAFKA_CONSUMER, topic, initInvoke, '', first)
                summaryLog.addErrorBlock(NODE_NAME.KAFKA_CONSUMER, topic, 'null', 'Invalid_schema')

                throw new ServerKafkaError({
                  message: 'Invalid schema',
                  topic: topic,
                  payload: first,
                })
              }
            }

            if (schema?.headers) {
              const parsedQuery = rawMessage.headers
              const typeCheck = TypeCompiler.Compile(schema.headers as TSchema)
              if (!typeCheck.Check(parsedQuery)) {
                const first = typeCheck.Errors(parsedQuery)
                detailLog.addOutputRequest(NODE_NAME.KAFKA_CONSUMER, topic, initInvoke, '', first)
                summaryLog.addErrorBlock(NODE_NAME.KAFKA_CONSUMER, topic, 'null', 'Invalid_schema')
                throw new ServerKafkaError({
                  message: 'Invalid schema',
                  topic: topic,
                  payload: first,
                })
              }
            }
          }

          return {
            detailLog,
            summaryLog,
            initInvoke,
            session,
          }
        },
        sendMessage: async (topic: string, payload: any) => await this.sendMessage(topic, payload, ctx),
      }

      const response = await handler(kafkaContext)

      if (response.topic && response.data) {
        await this.sendMessage(response.topic, response.data, ctx)
      }
      if (!ctx.detailLog?.isEnd()) {
        ctx.detailLog.addOutputRequest(NODE_NAME.KAFKA_CONSUMER, ctx.scenario, ctx.initInvoke, '', response).end()
      }

      if (!ctx.summaryLog?.isEnd()) {
        ctx.summaryLog.end('', 'success')
      }
    } catch (error) {
      if (error instanceof Error) {
        if (ctx.summaryLog) {
          ctx.summaryLog?.addField('errorCause', error.message)
        }
      }

      if (!ctx.summaryLog?.isEnd()) {
        ctx.summaryLog.end('500', 'server_error')
      }
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

  private async sendMessage(topic: string, payload: any, ctx: ICtxProducer) {
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

    ctx.detailLog.addOutputRequest(
      NODE_NAME.KAFKA_PRODUCER,
      topic,
      invokeKafka,
      processReqLog.RawData,
      processReqLog.Body
    )
    ctx.detailLog.end()

    try {
      await producer.connect()
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
        result_desc: 'success',
        result_data: recordMetadata,
      }

      ctx.summaryLog.addSuccessBlock(NODE_NAME.KAFKA_PRODUCER, topic, '200', result.result_desc)
      ctx.detailLog.addInputResponse(NODE_NAME.KAFKA_PRODUCER, topic, invokeKafka, '', result)

      return result
    } catch (error) {
      const result = {
        err: true,
        result_desc: 'Failed to send message',
        result_data: [],
      }

      if (error instanceof Error) {
        result.result_desc = error.message
      }

      ctx.summaryLog.addErrorBlock(NODE_NAME.KAFKA_PRODUCER, topic, '500', result.result_desc)
      ctx.detailLog.addInputResponse(NODE_NAME.KAFKA_PRODUCER, topic, invokeKafka, '', result)

      return result
    } finally {
      await producer.disconnect()
    }
  }

  private getHandlerByPattern(pattern: string): MessageHandler<any> | undefined {
    return this.messageHandlers.get(pattern)
  }

  public consume<Schema extends Omit<SchemaCtx, 'query'>>(
    pattern: string,
    handler: MessageHandler<Schema>,
    schema?: Schema
  ): void {
    this.messageHandlers.set(pattern, handler)
    if (schema) {
      this.schemaHandler.set(pattern, schema)
    }
  }
}
