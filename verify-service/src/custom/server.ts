import express, { type Request, type Response, type NextFunction, type Express } from 'express'
import { CtxSchema, HttpMethod, InternalRoute, InlineHandler, MiddlewareRoute } from './context.js'
import { TObject, Type } from '@sinclair/typebox'
import useragent from 'useragent'
import { v4, v7 } from 'uuid'
import { TypeCompiler } from '@sinclair/typebox/compiler'

import Fastify, {
  FastifyRequest,
  FastifyReply,
  FastifyInstance,
  HookHandlerDoneFunction,
  FastifyListenOptions,
  FastifyHttpOptions,
} from 'fastify'

// const router = express.Router()

// Example Route Handler

class AppRouter {
  protected readonly _routes: InternalRoute[] = []

  private add(method: HttpMethod, path: string, handler: InlineHandler<any, any>, hook?: MiddlewareRoute<any>) {
    this._routes.push({ method, path, handler, hook })
    return this
  }

  public get = <const P extends string, const R extends CtxSchema>(
    path: P,
    handler: InlineHandler<R, P>,
    hook?: MiddlewareRoute<R>
  ) => this.add(HttpMethod.GET, path, handler, hook)

  public post = <const P extends string, const R extends CtxSchema>(
    path: P,
    handler: InlineHandler<R, P>,
    hook?: MiddlewareRoute<R>
  ) => this.add(HttpMethod.POST, path, handler, hook)

  public put = <const P extends string, const R extends CtxSchema>(
    path: P,
    handler: InlineHandler<R, P>,
    hook?: MiddlewareRoute<R>
  ) => this.add(HttpMethod.PUT, path, handler, hook)

  public delete = <const P extends string, const R extends CtxSchema>(
    path: P,
    handler: InlineHandler<R, P>,
    hook?: MiddlewareRoute<R>
  ) => this.add(HttpMethod.DELETE, path, handler, hook)

  public patch = <const P extends string, const R extends CtxSchema>(
    path: P,
    handler: InlineHandler<R, P>,
    hook?: MiddlewareRoute<R>
  ) => this.add(HttpMethod.PATCH, path, handler, hook)

  public getRoutes = () => this._routes
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

export default class AppServer extends AppRouter {
  constructor(private readonly framework: TFramework = 'express') {
    super()

    if (this.framework === 'fastify') {
      this.fastify = Fastify()
      this.fastify.addHook('onRequest', (req: FastifyRequest, _res: FastifyReply, done: HookHandlerDoneFunction) => {
        const agent = useragent.parse(req.headers['user-agent'] || '')
        req.deviceInfo = {
          browser: agent.family,
          version: agent.toVersion(),
          os: agent.os.toString(),
          device: agent.device.toString(),
        }

        if (!req.headers['x-session']) {
          req.headers['x-session'] = v4()
          req.session = req.headers['x-session'] as string
        }

        if (!req.headers['x-tid']) {
          req.headers['x-tid'] = v7()
          req.invoke = req.headers['x-tid'] as string
        }

        done()
      })
    } else {
      this.express = express()
      this.express.use((req: Request, _res: Response, next: NextFunction) => {
        const agent = useragent.parse(req.headers['user-agent'] || '')
        req.deviceInfo = {
          browser: agent.family,
          version: agent.toVersion(),
          os: agent.os.toString(),
          device: agent.device.toString(),
        }

        if (!req.headers['x-session']) {
          req.headers['x-session'] = v4()
          req.session = req.headers['x-session'] as string
        }

        if (!req.headers['x-tid']) {
          req.headers['x-tid'] = v7()
          req.invoke = req.headers['x-tid'] as string
        }

        next()
      })
    }
  }

  private readonly express: Express | null = null
  private readonly fastify: FastifyInstance | null = null

  public router(router: AppRouter) {
    router.getRoutes().forEach((e) => this._routes.push(e))
  }

  private validatorFactory(req: CtxSchema, schema: CtxSchema) {
    try {
      const errors = []
      if (schema.body) {
        const C = TypeCompiler.Compile(schema.body as TObject)
        const isValid = C.Check(req.body)
        if (!isValid) {
          errors.push(...[...C.Errors(req.body)].map((e) => ({ type: 'body', path: e.path, message: e.message })))
        }
      }

      if (schema.params) {
        const C = TypeCompiler.Compile(schema.params as TObject)
        const isValid = C.Check(req.params)
        if (!isValid) {
          errors.push(...[...C.Errors(req.params)].map((e) => ({ type: 'params', path: e.path, message: e.message })))
        }
      }

      if (schema.query) {
        const C = TypeCompiler.Compile(schema.query as TObject)
        const isValid = C.Check(req.query)
        if (!isValid) {
          errors.push(...[...C.Errors(req.query)].map((e) => ({ type: 'query', path: e.path, message: e.message })))
        }
      }

      if (schema.headers) {
        const C = TypeCompiler.Compile(schema.headers as TObject)
        const isValid = C.Check(req.headers)
        if (!isValid) {
          errors.push(...[...C.Errors(req.headers)].map((e) => ({ type: 'headers', path: e.path, message: e.message })))
        }
      }

      const isError = errors.length > 0 ? true : false
      return {
        err: isError,
        desc: isError ? 'invalid_request' : 'success',
        data: errors,
      }
    } catch (error) {
      return {
        err: true,
        desc: 'unknown_error',
        data: [error],
      }
    }
  }

  public listen(port: number, callback: (err?: Error) => void) {
    if (!this.express && !this.fastify) {
      throw new Error('App is not initialized')
    }

    // if add === FastifyInstance
    if (this.fastify) {
      const fastify = this.fastify as FastifyInstance

      this.getRoutes().forEach(({ method, path, handler, hook }) => {
        const schema = hook?.schema
        fastify[method](path, async (req: FastifyRequest, res: FastifyReply) => {
            
          try {
            const ctx: CtxSchema = {
              body: req.body,
              headers: req.headers,
              params: req.params,
              query: req.query,
              // session: req.session,
              // invoke: req.invoke,
              // deviceInfo: req.deviceInfo as any,
            //   cookie: req.cookies,
              response: (code: number, data: any) => {
                res.code(code).send(data)
              },
            }
          } catch (error) {}
        })
      })

      //   this.fastify.listen({ port })
      const _start = async () => {
        try {
          const opts: FastifyListenOptions = {
            port,
          }

          const s = await fastify.listen(opts)
          //   cb ? cb() : console.log(`Server is running on port ${s}`)
        } catch (err) {
          fastify.log.error(err)
          process.exit(1)
        }
      }

      _start()
    } else {
    }

    // this.routes().forEach(({ method, path, handler }) => {
    //   app[method.toLowerCase() as keyof express.Application](
    //     path,
    //     async (req: Request, res: Response, next: NextFunction) => {
    //       const result = await handler(req)
    //       res.json(result)
    //     }
    //   )
    // })

    // app.listen(port, callback)
  }
}

export function Router() {
  return new AppRouter()
}

const router = Router()
router
  .get(
    '/user/:id',
    async ({ params, body, headers, query }) => {
      return {}
    },
    {
      schema: {
        body: Type.Object({
          id: Type.String(),
        }),
        params: Type.Object({
          id: Type.String(),
        }),
        headers: Type.Object({
          'x-custom-header': Type.String(),
        }),
      },
    }
  )
  .post('/user/:id', async ({ params, body, headers, query }) => {
    return {}
  })

const app = new AppServer()
app.router(router)
app.listen(3000, (err) => {
  if (err) {
    console.error(err)
    process.exit(1)
  }

  console.log('Server is running')
})
