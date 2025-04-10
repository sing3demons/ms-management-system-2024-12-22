import { IConfig } from '../config.js'
import { Context, AppRoutes, ErrorValidator, IServer, SchemaCtx } from './route.js'
import { Type as t } from '@sinclair/typebox'

import express, { Response, Request, NextFunction, Express } from 'express'
import Fastify, { FastifyRequest, FastifyReply, FastifyInstance, HookHandlerDoneFunction, FastifyListenOptions } from 'fastify'

import NODE_NAME from '../constants/nodeName.js'
import { generateXTid } from '../v1/route.js'
import useragent from 'useragent'
import { v4 as uuid } from 'uuid'
import { MessageHandler, ServerKafka } from '../../kafka_server.js'
import { loadLogConfig } from '../logger/logger.js'

class ExpressServer extends AppRoutes implements IServer {
  private readonly express: Express
  private readonly serverKafka: ServerKafka

  constructor() {
    super()

    this.express = express()
    this.express.use(express.json())
    this.express.use(express.urlencoded({ extended: true }))
    this.express.use((req: Request, _res: Response, next: NextFunction) => {
      const agent = useragent.parse(req.headers['user-agent'])
      req.deviceInfo = {
        browser: agent.family,
        version: agent.toVersion(),
        os: agent.os.toString(),
        device: agent.device.toString(),
      }

      if (!req.headers[this.session]) {
        req.headers[this.session] = uuid()
        req.session = req.headers[this.session] as string
      }

      if (!req.headers[this.transaction]) {
        req.headers[this.transaction] = generateXTid()
        req.invoke = req.headers[this.transaction] as string
      }

      next()
    })

    this.serverKafka = new ServerKafka({
      client: {
        brokers: ['localhost:29092'],
        clientId: 'kafka-client',
      },
      consumer: {
        groupId: 'kafka-group',
      },
    })
  }

  private createContext(req: Request, res: Response, schema?: SchemaCtx): Context {
    let initInvoke = ''
    let firstCmd = ''

    const context = {
      body: req.body,
      params: req.params,
      query: req.query,
      headers: req.headers,
      setHeaders: (headers: Record<string, string>) => {
        Object.entries(headers).forEach(([key, value]) => {
          res.setHeader(key, value)
        })
      },
      commonLog: (scenario: string, identity?: string) => {
        const logs = this.initializeLogs(req, scenario, identity, schema)
        initInvoke = logs.initInvoke
        firstCmd = scenario
        req.detailLog = logs.detailLog
        req.summaryLog = logs.summaryLog
        return logs
      },
      response: (code: number, data: any) => {
        this.logResponse(req, firstCmd, initInvoke, data, code)
        res.status(code).json(data)
      },
      redirect: (url: string, code: number = 302) => {
        this.logResponse(req, firstCmd, initInvoke, url, code)
        res.status(code)
        res.redirect(url)
      },
    }

    return context
  }

  private logResponse(req: { detailLog?: any; summaryLog?: any }, scenario: string, initInvoke: string, data: any, code: number) {
    const logFramework = 'Fastify';
    req.detailLog?.addOutputRequest(`${NODE_NAME.CLIENT} (${logFramework})`, scenario, initInvoke, '', data)?.end()

    if (!req.summaryLog?.isEnd) {
      req.summaryLog.end(`${code} (${logFramework})`, '')
    }
  }

  private sendErrorResponse(res: Response, error: any): void {
    res.status(400)
    res.json(JSON.stringify({ error: error.message }))
  }

  private handleError(res: Response, error: unknown, req: Request) {
    if (error instanceof ErrorValidator) {
      res.status(error.code).json(error.response)
      return
    }
    this.sendErrorResponse(res, error)
  }

  public register(...routes: AppRoutes[]): this {
    this._RoutesMetadata.push(...routes.flatMap((r) => r.routes()))

    return this
  }

  public start(cb?: Function) {
    this._RoutesMetadata.forEach(({ path, method, handler, schema }) => {
      this.express[method](path, async (req: Request, res: Response) => {
        try {
          const context = this.createContext(req, res, schema)
          return await handler(context)
        } catch (error) {
          this.handleError(res, error, req)
        } finally {
          this.cleanupResources(req)
        }
      })
    })

    const port = process.env?.PORT?.split(':').pop() ?? '3000'

    this.express.listen(port, () => {
      cb ? cb() : console.log(`Server is running on port ${port}`)
    })
  }
  private cleanupResources(req: Request) {
    if (req.detailLog) {
      try {
        ; (req.detailLog as any)?.finalize?.()
      } catch (error) {
        console.error('Error finalizing detail log:', error)
      }
    }

    if (req.summaryLog) {
      try {
        ; (req.summaryLog as any)?.finalize?.()
      } catch (error) {
        console.error('Error finalizing summary log:', error)
      }
    }
  }

  public consume(topic: string, handler: MessageHandler): void {
    this.serverKafka.addHandler(topic, handler)
  }
}

class FastifyServer extends AppRoutes implements IServer {
  private readonly fastify: FastifyInstance = Fastify()
  private readonly serverKafka: ServerKafka
  constructor() {
    super()

    this.fastify.addHook('onRequest', (req: FastifyRequest, _res: FastifyReply, done: HookHandlerDoneFunction) => {
      const agent = useragent.parse(req.headers['user-agent'])
      req.deviceInfo = {
        browser: agent.family,
        version: agent.toVersion(),
        os: agent.os.toString(),
        device: agent.device.toString(),
      }

      if (!req.headers[this.session]) {
        req.headers[this.session] = uuid()
        req.session = req.headers[this.session] as string
      }

      if (!req.headers[this.transaction]) {
        req.headers[this.transaction] = generateXTid('clnt')
        req.invoke = req.headers[this.transaction] as string
      }

      done()
    })

    this.serverKafka = new ServerKafka({
      client: {
        brokers: ['localhost:29092'],
        clientId: 'kafka-client',
      },
      consumer: {
        groupId: 'kafka-group',
      },
    })
  }

  private createContext(req: FastifyRequest, res: FastifyReply, schema?: SchemaCtx): Context {
    let initInvoke = ''
    let firstCmd = ''

    const context = {
      body: req.body,
      params: req.params,
      query: req.query,
      headers: req.headers,
      setHeaders: (headers: Record<string, string>) => {
        Object.entries(headers).forEach(([key, value]) => {
          res.header(key, value)
        })
      },
      commonLog: (scenario: string, identity?: string) => {
        const logs = this.initializeLogs(req, scenario, identity, schema)
        initInvoke = logs.initInvoke
        firstCmd = scenario
        req.detailLog = logs.detailLog
        req.summaryLog = logs.summaryLog
        return logs
      },
      response: (code: number, data: any) => {
        this.logResponse(req, firstCmd, initInvoke, data, code)
        res.status(code).send(data)
      },
      redirect: (url: string, code: number = 302) => {
        res.status(code)
        res.redirect(url)
      },
    }

    return context
  }

  private logResponse(req: { detailLog?: any; summaryLog?: any }, scenario: string, initInvoke: string, data: any, code: number) {
    req.detailLog?.addOutputRequest(NODE_NAME.CLIENT, scenario, initInvoke, '', data)?.end()

    if (!req.summaryLog?.isEnd) {
      req.summaryLog.end(`${code}`, '')
    }
  }

  public register(...routes: AppRoutes[]): this {
    for (const route of routes) {
      this._RoutesMetadata.push(...route.routes());
    }

    return this;
  }

  public consume(topic: string, handler: MessageHandler): void {
    this.serverKafka.addHandler(topic, handler)
  }

  public start(cb?: Function): void {
    this._RoutesMetadata.forEach(({ path, method, handler, schema }) => {
      this.fastify[method](path, async (req: FastifyRequest, res: FastifyReply) => {
        try {
          const context = this.createContext(req, res, schema)
          return await handler(context)
        } catch (error) {
          if (error instanceof ErrorValidator) {
            res.status(error.code).send(error.response)
            return
          }

          if (error instanceof Error) {
            res.status(500).send(error.message)
            return
          }

          res.status(500).send(error)
        }
      })
    })

    const port = process.env?.PORT?.split(':').pop() ?? '3000'
    const _start = async () => {
      try {
        const opts: FastifyListenOptions = {
          port: parseInt(port),
        }
        const s = await this.fastify.listen(opts)
        cb ? cb() : console.log(`Server is running on port ${s}`)
      } catch (err) {
        this.fastify.log.error(err)
        process.exit(1)
      }
    }

    this.serverKafka.listen((err) => {
      if (err) {
        console.log('Error:', err)
        return
      }

      console.log('Kafka server is running')
    })

    _start()

  }
}

enum Framework {
  EXPRESS = 'express',
  FASTIFY = 'fastify',
}

const framework = {
  express: Framework.EXPRESS,
  fastify: Framework.FASTIFY,
} as const

type TFramework = keyof typeof framework

class AppServer {
  constructor(private readonly framework: TFramework) { }

  load = (config: IConfig) => {
    if (config) {
      loadLogConfig(config)
    }

    switch (this.framework) {
      case framework.express:
        return new ExpressServer()
      case framework.fastify:
        return new FastifyServer()
      default:
        throw new Error('Invalid framework')
    }
  }
}

export { t, AppRoutes }
export default AppServer
