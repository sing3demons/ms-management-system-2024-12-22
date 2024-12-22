import NODE_NAME from './constants/nodeName.js'
import TOPICS from './constants/topics.js'
import { ServerKafka } from './kafka/kafka_server.js'
import { loadLogConfig } from './logger/logger.js'
import { Type } from '@sinclair/typebox'
import TokenModel from './schema/token.model.js'
import crypto from 'crypto'
import { DetailLog, SummaryLog } from './logger/index.js'
import generateInternalTid from './v2/generateInternalTid.js'
import { HttpService } from './v2/http-service.js'
import { initMongo, mongo } from './mongo/mongo.js'

loadLogConfig({
  detail: {
    console: false,
    file: true,
    rawData: true,
    path: './logs/detail/',
  },
  summary: {
    console: true,
    file: true,
    path: './logs/summary/',
  },
})

initMongo()

const app = new ServerKafka({
  client: {
    brokers: ['localhost:29092'],
    clientId: 'kafka-client-app',
    logLevel: 1,
    retry: {
      initialRetryTime: 100,
      retries: 8,
    },
  },
})

const verifySchema = Type.Object({
  email: Type.Optional(Type.String()),
  username: Type.Optional(Type.String()),
  id: Type.String(),
})

app.consume(
  TOPICS.SERVICE_VERIFY,
  async ({ commonLog, body }) => {
    const { detailLog, summaryLog } = commonLog(TOPICS.SERVICE_REGISTER)

    summaryLog.addSuccessBlock(NODE_NAME.KAFKA_CONSUMER, TOPICS.SERVICE_REGISTER, 'null', 'success')

    // insertToken(body, detailLog, summaryLog)
    const resultToken = await insertToken(body, detailLog, summaryLog)
    if (resultToken.err) {
      return {
        ...resultToken,
        success: false,
      }
    }
    const requestHttp = await HttpService.requestHttp(
      {
        _command: 'send_email',
        _invoke: generateInternalTid('send_email', '-', 20),
        _service: 'email_service',
        body: {
          from: 'dev@test.com',
          to: body.email || 'test@test.com',
          subject: 'Verify Email',
          body: `http://localhost:8080/verify/${resultToken.result_data.token}`,
        },
        headers: {
          'Content-Type': 'application/json',
        },
        method: 'POST',
        url: 'http://localhost:8080/mail',
      },
      detailLog,
      summaryLog
    )

    summaryLog.addSuccessBlock('email_service', 'send_email', '200', 'success')

    return { ...requestHttp, token: resultToken.result_data.token, success: true }
  },
  { body: verifySchema }
)

app.consume(
  TOPICS.SERVICE_VERIFY_CONFIRM,
  async ({ commonLog, value, body }) => {
    return { success: true }
  },
  { body: verifySchema }
)

app.listen((err) => {
  if (err) {
    console.error(err)
    process.exit(1)
  }

  console.log('Kafka server is running')
})

async function insertToken<T extends { id: string }>(body: T, detailLog: DetailLog, summaryLog: SummaryLog) {
  const cmd = 'insert_token',
    node = NODE_NAME.MONGO,
    invokeMongo = generateInternalTid('mongo', '-', 20),
    method = 'create'

  const document = {
    filter: { userId: body.id },
    update: { userId: body.id, token: crypto.randomBytes(32).toString('hex') },
    options: {},
  }

  const result = await mongo(TokenModel, method, {
    filter: document.filter,
    new: document.update,
  })

  detailLog.addOutputRequest(node, cmd, invokeMongo, result.outgoing_detail.RawData, result.outgoing_detail.Body)
  detailLog.end()
  if (result.err) {
    summaryLog.addErrorBlock(node, cmd, '500', result.result_desc)

    detailLog.addInputResponseError(node, cmd, invokeMongo, result.result_desc)
    return {
      err: result.err,
      result_desc: result.result_desc,
      result_data: result.result_data,
    }
  }

  const processResLog = {
    Body: {
      Return: result.result_data,
    },
    RawData: JSON.stringify(result.result_data),
  }

  summaryLog.addSuccessBlock(node, cmd, '201', result.result_desc)
  detailLog.addInputResponse(node, cmd, invokeMongo, processResLog.RawData, processResLog.Body)

  return {
    err: result.err,
    result_desc: result.result_desc,
    result_data: result.result_data,
  }
}
