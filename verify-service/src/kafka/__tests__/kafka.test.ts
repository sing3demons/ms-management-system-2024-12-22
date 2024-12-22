import { Kafka, Consumer, Producer, KafkaMessage } from 'kafkajs'
import { ServerKafka, MessageHandler, ServerKafkaOptions } from '../kafka_server.js'

jest.mock('kafkajs')

describe('ServerKafka', () => {
  let serverKafka: ServerKafka
  let options: ServerKafkaOptions

  beforeEach(() => {
    options = {
      client: {
        clientId: 'test-client',
        brokers: ['localhost:9092'],
      },
      consumer: {
        groupId: 'test-group',
      },
      producer: {},
    }
    serverKafka = new ServerKafka(options)
  })

  afterEach(async () => {
    await serverKafka.close()
  })

  it('should create a Kafka client with the correct configuration', () => {
    const kafkaConfig = {
      clientId: 'test-client-server',
      brokers: ['localhost:9092'],
      logLevel: 1,
      retry: {
        initialRetryTime: 300,
        retries: 8,
      },
    }

    const kafkaInstance = serverKafka['createClient']()
    expect(Kafka).toHaveBeenCalledWith(kafkaConfig)
  })

  it('should connect and start the consumer and producer', async () => {
    const callback = jest.fn()
    await serverKafka.listen(callback)

    expect(serverKafka['consumer']?.connect).toHaveBeenCalled()
    expect(serverKafka['producer']?.connect).toHaveBeenCalled()
    expect(callback).toHaveBeenCalled()
  })

  it('should disconnect the consumer and producer on close', async () => {
    await serverKafka.listen(jest.fn())
    await serverKafka.close()

    expect(serverKafka['consumer']?.disconnect).toHaveBeenCalled()
    expect(serverKafka['producer']?.disconnect).toHaveBeenCalled()
  })

  it('should handle messages correctly', async () => {
    const handler: MessageHandler = jest.fn().mockResolvedValue({})
    serverKafka.consume('test-topic', handler)

    const message: KafkaMessage = {
      key: null,
      value: Buffer.from(JSON.stringify({ key: 'value' })),
      timestamp: Date.now().toString(),
      attributes: 0,
      offset: '0',
      headers: {},
    }

    await serverKafka['handleMessage']({
      topic: 'test-topic',
      partition: 0,
      message,
    })

    expect(handler).toHaveBeenCalled()
  })

  it('should send messages correctly', async () => {
    await serverKafka.listen(jest.fn())

    const result = await serverKafka['sendMessage'](
      'test-topic',
      { key: 'value' },
      {
        initInvoke: '',
        detailLog: { addOutputRequest: jest.fn(), end: jest.fn() } as any,
        summaryLog: { addSuccessBlock: jest.fn(), addErrorBlock: jest.fn(), end: jest.fn() } as any,
        scenario: '',
      }
    )

    expect(result.err).toBe(false)
    expect(result.result_desc).toBe('Success')
  })
})
