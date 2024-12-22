import { Kafka, Message, Producer, Consumer, RecordMetadata, KafkaConfig } from 'kafkajs'
import DetailLog from './logger/detail.js'
import SummaryLog from './logger/summary.js'
import generateInternalTid from './utils/generateInternalTid.js'
import NODE_NAME from './constants/nodeName.js'
import { projectName } from './utils/index.js'

const kafka: {
  producer?: Producer
  consumer?: Consumer
} = {}

function initKafka() {
  const brokers = process.env.KAFKA_BROKERS?.split(',') ?? ['localhost:29092']
  const clientId = process.env.KAFKA_CLIENT_ID ?? projectName
  const requestTimeout = process.env?.KAFKA_REQUEST_TIMEOUT ?? 30000
  const retry = process.env?.KAFKA_RETRY ?? 8
  const initialRetryTime = process.env?.KAFKA_INITIAL_RETRY_TIME ?? 100

  if (!process.env.KAFKAJS_NO_PARTITIONER_WARNING) process.env.KAFKAJS_NO_PARTITIONER_WARNING = '1'

  const config: KafkaConfig = {
    brokers,
    clientId,
    requestTimeout: Number(requestTimeout),
    retry: {
      initialRetryTime: Number(initialRetryTime),
      retries: Number(retry),
    },
  }

  if (process.env.KAFKA_USERNAME && process.env.KAFKA_PASSWORD) {
    config.sasl = {
      mechanism: 'plain',
      username: process.env.KAFKA_USERNAME,
      password: process.env.KAFKA_PASSWORD,
    }
  }

  const k = new Kafka(config)

  kafka.producer = k.producer()
  kafka.consumer = k.consumer({ groupId: clientId })
}

type ProducerResponseSuccess = {
  err: false
  result_desc: string
  result_data: RecordMetadata[]
}

type ProducerResponseError = {
  err: true
  result_desc: string
}

async function produceMessage(
  command: string,
  data: Array<object> | object | string,
  detailLog: DetailLog,
  summaryLog: SummaryLog
): Promise<ProducerResponseSuccess | ProducerResponseError> {
  const producer = kafka.producer
  if (!producer) {
    return {
      err: true,
      result_desc: 'Producer not initialized',
    }
  }
  let messages: Message[] = []
  if (typeof data === 'object') {
    if (Array.isArray(data)) {
      data.forEach((msg) => {
        messages.push({ value: JSON.stringify(msg) })
      })
    } else {
      messages.push({ value: JSON.stringify(data) })
    }
  } else {
    messages.push({ value: data })
  }

  const invokeKafka = generateInternalTid('kafka', '-', 20)

  try {
    const payload = {
      topic: command.replace('.', '_'),
      messages: messages.map((msg) => {
        try {
          return JSON.parse(String(msg.value))
        } catch (error) {
          return msg.value
        }
      }),
    }

    const processReqLog = {
      Body: payload,
      RawData: JSON.stringify(payload),
    }
    detailLog.addOutputRequest(
      NODE_NAME.KAFKA_PRODUCER,
      command,
      invokeKafka,
      processReqLog.RawData,
      processReqLog.Body,
      NODE_NAME.KAFKA_PRODUCER
    )
    detailLog.end()

    await kafka.producer?.connect()
    const recordMetadata = await producer.send({ topic: command, messages })

    const processResLog = {
      Body: { Return: recordMetadata },
      RawData: JSON.stringify(recordMetadata),
    }
    detailLog.addInputResponse(NODE_NAME.KAFKA_PRODUCER, command, invokeKafka, processResLog.RawData, processResLog.Body)
    summaryLog.addSuccessBlock(NODE_NAME.KAFKA_PRODUCER, command, '20000', 'success')
    return {
      err: false,
      result_desc: 'Success',
      result_data: recordMetadata,
    }
  } catch (error) {
    if (error instanceof Error) {
      if (error?.message.includes('Connection timeout')) {
        summaryLog.addField('errorCause', error.message)
        summaryLog.addErrorBlock(NODE_NAME.KAFKA_PRODUCER, command, 'ret=4', 'Connection timeout')
        detailLog.addInputResponseError(NODE_NAME.KAFKA_PRODUCER, command, invokeKafka, error.message)
        return {
          err: true,
          result_desc: error.message,
        }
      } else if (error.message.includes('Connection error')) {
        summaryLog.addField('errorCause', error.message)
        summaryLog.addErrorBlock(NODE_NAME.KAFKA_PRODUCER, command, 'ret=1', 'Connection error')
        detailLog.addInputResponseError(NODE_NAME.KAFKA_PRODUCER, command, invokeKafka, error.message)
        return {
          err: true,
          result_desc: `${error.message}`,
        }
      }
      summaryLog.addField('errorCause', error.message)
      summaryLog.addErrorBlock(NODE_NAME.KAFKA_PRODUCER, command, '', 'system error')
      detailLog.addInputResponseError(NODE_NAME.KAFKA_PRODUCER, command, invokeKafka, error.message)

      return {
        err: true,
        result_desc: error.message,
      }
    }

    summaryLog.addField('errorCause', error)
    summaryLog.addErrorBlock(NODE_NAME.KAFKA_PRODUCER, command, '', 'system error')
    detailLog.addInputResponseError(NODE_NAME.KAFKA_PRODUCER, command, invokeKafka, String(error))
    return {
      err: true,
      result_desc: String(error),
    }
  } finally {
    await kafka.producer?.disconnect()
  }
}

export { kafka, initKafka, produceMessage }
