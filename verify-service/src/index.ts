import NODE_NAME from './constants/nodeName.js'
import TOPICS from './constants/topics.js'
import { CtxConsumer, ServerKafka } from './kafka/kafka_server.js'
import { loadLogConfig } from './logger/logger.js'
import { Static, TSchema, Type } from '@sinclair/typebox'
import TokenModel from './mongo/models/token.model.js'
import crypto from 'crypto'
import { DetailLog, SummaryLog } from './logger/index.js'
import generateInternalTid from './v2/generateInternalTid.js'
import { HttpService } from './v2/http-service.js'
import { initMongo, mongo } from './mongo/mongo.js'
import { initPostgres, sql } from './sql/pg.js'
import { verifySchema, VerifySchemaType } from './schema/verify.schema.js'

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

initMongo({
  url: 'mongodb://localhost:27017/verify-service',
})
initPostgres({
  host: 'localhost',
  port: 5432,
  user: 'postgres',
  password: 'syspass',
  database: 'profile_service',
})

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

type User = {
  id: string
  email: string
  username: string
  password: string
  first_name?: string
  last_name?: string
  date_of_birth?: string
  phone_number?: string
  gender?: string
  created_at: string
  updated_at: string
  created_by?: string
  updated_by?: string
  display_name?: string
  profile_image?: string
}
async function verifyHandler({ commonLog, body }: CtxConsumer<VerifySchemaType>) {
  const { detailLog, summaryLog } = commonLog(TOPICS.SERVICE_REGISTER)

  summaryLog.addSuccessBlock(NODE_NAME.KAFKA_CONSUMER, TOPICS.SERVICE_REGISTER, 'null', 'success')

  const data = await sql<User>('FIND_ONE', { table: 'Profile', condition: 'id = $1', conditionParams: [body.id] })
  const invokePostgres = generateInternalTid('postgres', '-', 20)
  detailLog
    .addOutputRequest(NODE_NAME.POSTGRES, 'find_one', invokePostgres, data.outgoing_detail.Query, data.outgoing_detail)
    .end()

  if (data.err) {
    summaryLog.addErrorBlock(NODE_NAME.POSTGRES, 'find_one', '500', data.result_desc)
    detailLog.addInputResponseError(NODE_NAME.POSTGRES, 'find_one', invokePostgres, data.result_desc)
    return {
      err: data.err,
      result_desc: data.result_desc,
      result_data: data.result_data,
      success: false,
    }
  }

  summaryLog.addSuccessBlock(NODE_NAME.POSTGRES, 'find_one', '200', data.result_desc)
  detailLog.addInputResponse(
    NODE_NAME.POSTGRES,
    'find_one',
    invokePostgres,
    data.ingoing_detail.RawData,
    data.ingoing_detail.Body
  )

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
}

app.consume(TOPICS.SERVICE_VERIFY, async (c) => verifyHandler(c), {
  body: verifySchema,
  beforeHandle: async (ctx) => {
    console.log('beforeHandle', ctx.body)
  },
})

app.consume(
  TOPICS.SERVICE_VERIFY_CONFIRM,
  async ({ body, headers }) => {
    return { success: true }
  },
  {
    body: verifySchema,
    headers: Type.Object({
      Authorization: Type.String({
        pattern: '/^Bearer .+$/',
      }),
    }),
  }
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
