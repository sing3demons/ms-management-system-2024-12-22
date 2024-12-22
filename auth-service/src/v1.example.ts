import NODE_NAME from './lib/constants/nodeName.js'
import { initKafka, produceMessage } from './lib/kafka.js'
import AppServer, { AppRouter } from './lib/v1/route.js'
import generateInternalTid from './lib/utils/generateInternalTid.js'
import { v4 as uuid } from 'uuid'

const app = new AppServer(() => initKafka())
const route = new AppRouter()

route.get('/x/:id', async function ({ params, res, req }) {
  const cmd = 'x'
  const detailLog = req.detailLog.New(cmd)
  const summaryLog = req.summaryLog.New(cmd)
  const internalTid = generateInternalTid(cmd, '-', 20)

  detailLog.addInputRequest(NODE_NAME.CLIENT, cmd, internalTid, req)
  summaryLog.addSuccessBlock(NODE_NAME.CLIENT, cmd, 'null', 'success')

  await produceMessage('service.x', { username: params.id }, detailLog, summaryLog)

  const data = {
    message: 'Hello World',
    x: params.id,
  }

  detailLog.addOutputRequest(NODE_NAME.CLIENT, cmd, internalTid, '', data).end()
  summaryLog.end('', 'success')
  res.json({ data })
})

route.get('/auth', async function ({ res, req }) {
  const cmd = 'auth'
  const detailLog = req.detailLog.New(cmd)
  const summaryLog = req.summaryLog.New(cmd)
  const internalTid = generateInternalTid(cmd, '-', 20)

  detailLog.addInputRequest(NODE_NAME.CLIENT, cmd, internalTid, req)
  summaryLog.addSuccessBlock(NODE_NAME.CLIENT, cmd, 'null', 'success')

  const data = []
  for (let i = 0; i < 1000; i++) {
    data.push({
      header: {
        session: `${req.session}:${uuid()}`,
      },
      body: {
        username: `test${i}`,
        password: '123456',
        email: `test${i}@dev.com`,
      },
    })
  }

  // const data = {
  //   header: {
  //     session: req.session,
  //   },
  //   body: {
  //     username: 'test2',
  //     password: 'test1',
  //     email: 'test2@dev.com',
  //   },
  // }

  const resp = await produceMessage('service.register', data, detailLog, summaryLog)
  detailLog.addOutputRequest(NODE_NAME.CLIENT, cmd, internalTid, '', resp).end()
  summaryLog.end('', 'success')

  res.json({ data: resp })
})

app.router(route).listen(3000)