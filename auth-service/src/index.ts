import NODE_NAME from './lib/constants/nodeName.js'
import { initKafka, produceMessage } from './lib/kafka.js'
import { v4 as uuid } from 'uuid'
import AppServer, { AppRoutes, t } from './lib/v2/index.js'
import { HttpService } from './lib/http-service.js'
import generateInternalTid from './lib/utils/generateInternalTid.js'

const router = new AppRoutes()

router.get(
  '/auth',
  async function ({ commonLog, response }) {
    const cmd = 'auth',
      { detailLog, summaryLog } = commonLog(cmd)

    summaryLog.addSuccessBlock(NODE_NAME.CLIENT, cmd, 'null', 'success')

    const data = []
    for (let i = 0; i < 5; i++) {
      data.push({
        header: {
          session: `${detailLog.detailLog.Session}:${uuid()}`,
        },
        body: {
          username: `test${i}`,
          password: '123456',
          email: `test${i}@dev.com`,
        },
      })
    }

    const resp = await produceMessage('service.register', data, detailLog, summaryLog)
    summaryLog.end('', 'success')

    return response(201, {
      success: true,
    })
  },
  {
    body: t.Object({
      session: t.String(),
    }),
  }
)

router.post(
  '/register',
  async function (ctx) {
    const cmd = 'register'
    const { detailLog, summaryLog } = ctx.commonLog(cmd)
    summaryLog.addSuccessBlock(NODE_NAME.CLIENT, cmd, 'null', 'success')

    const resp = await produceMessage(
      'service.register',
      {
        header: {
          session: `${detailLog.detailLog.Session}`,
        },
        body: ctx.body,
      },
      detailLog,
      summaryLog
    )

    summaryLog.end('', 'success')

    return ctx.response(201, {
      success: true,
    })
  },
  {
    body: t.Object({
      email: t.String(),
      password: t.String(),
      username: t.String(),
    }),
  }
)

router.post(
  '/login',
  async function (ctx) {
    const cmd = 'login'
    const { detailLog, summaryLog } = ctx.commonLog(cmd)
    summaryLog.addSuccessBlock(NODE_NAME.CLIENT, cmd, 'null', 'success')

    const loginInvoke = generateInternalTid(cmd, '-', 20)

    const requestHttp = await HttpService.requestHttp(
      {
        _service: 'profile_service',
        _command: 'get_user',
        _invoke: loginInvoke,
        url: 'http://localhost:8080/users/{publicId}'.replace('{publicId}', ctx.body.email),
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      },
      detailLog,
      summaryLog
    )

    if (requestHttp.Status != 200) {
      summaryLog.end('', 'failed')

      return ctx.response(400, {
        success: false,
      })
    }

    summaryLog.addSuccessBlock('profile_service', cmd, 'get_user', 'success')

    summaryLog.end('', 'success')

    return ctx.response(201, {
      success: true,
      data: requestHttp,
    })
  },
  {
    body: t.Object({
      email: t.String(),
      password: t.String(),
    }),
  }
)

const app = new AppServer('fastify')
  .load({
    Addr: '3001',
    Env: 'dev',
    LogConfig: {
      detail: {
        console: false,
        file: true,
        path: './logs/detail/',
      },
      summary: {
        console: true,
        file: true,
        path: './logs/summary/',
      },
    },
  })
  .init(() => initKafka())
  .register(router)

// app.consume('service.register', async function (data, ctx) {
//   const { detailLog, summaryLog } = ctx.commonLog(ctx.topic)
//   summaryLog.addSuccessBlock(NODE_NAME.KAFKA_CONSUMER, 'x', 'null', 'success')

//   await ctx.sendMessage('service-otp', data)

//   return { success: true }
// })

app.start()
