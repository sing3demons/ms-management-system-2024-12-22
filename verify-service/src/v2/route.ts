import type { TSchema, Static, TObject } from '@sinclair/typebox'
import { TypeCompiler } from '@sinclair/typebox/compiler'
import DetailLog from '../logger/detail.js'
import SummaryLog from '../logger/summary.js'
import NODE_NAME from '../constants/nodeName.js'
import generateInternalTid from './generateInternalTid.js'
import { CtxConsumer, TSchemaCtx } from '../kafka/kafka_server.js'

type ExtractParams<Path extends string> = Path extends `${string}:${infer ParamName}/${infer Rest}`
  ? ParamName | ExtractParams<Rest>
  : Path extends `${string}:${infer ParamName}`
  ? ParamName
  : never

type HandlerContext = any[] | Blob | string | number | Record<any, any>

type SchemaRoutes = {
  body?: TObject
  query?: TObject
  headers?: TObject
}

type IContext<Schema extends SchemaRoutes, Route extends string> = {
  body: Schema['body'] extends TObject ? Static<Schema['body']> : Record<string, any>
  params: Record<ExtractParams<Route>, string>
  query: Schema['query'] extends TObject ? Static<Schema['query']> : Record<string, string>
  headers: Record<string, string>
  setHeaders: (header: Record<string, string>) => void
  commonLog: (cmd: string, identity?: string) => CommonLog
  response: (code: number, data: any) => void
}

enum HttpMethod {
  GET = 'get',
  POST = 'post',
  PUT = 'put',
  PATCH = 'patch',
  DELETE = 'delete',
}

type Handler<Route extends string, Schema extends SchemaRoutes> = (
  context: IContext<Schema, Route>
) => Promise<HandlerContext> | HandlerContext

type CommonLog = {
  detailLog: DetailLog
  summaryLog: SummaryLog
}

type SchemaCtx = {
  body?: TSchema
  query?: TSchema
  headers?: TSchema
}

type IInComing = {
  body: Record<string, any>
  query: Record<string, any>
}

type Context = {
  body: unknown
  params: unknown
  query: unknown
  headers: unknown
  setHeaders: (headers: Record<string, string>) => void
  commonLog: (
    scenario: string,
    identity?: string
  ) => {
    detailLog: DetailLog
    summaryLog: SummaryLog
    initInvoke: string
  }
  response(code: number, data: any): void
  redirect(url: string, code?: number): void
}

type HandlerConsumer = (context: Context) => Promise<HandlerContext> | HandlerContext

type ICtx = {
  session: string
  invoke: string
  body: any
  query: any
}

type TRoutesMetadata = {
  method: HttpMethod
  path: string
  handler: Function
  schema?: SchemaCtx
}

class AppRoutes {
  protected readonly transaction = 'x-tid'
  protected readonly session = 'x-session'
  constructor() {}

  public init(cb: Function): this {
    cb()
    return this
  }

  protected readonly _RoutesMetadata: Array<TRoutesMetadata> = []

  private route<Method extends HttpMethod, PathName extends `/${string}`, Schema extends SchemaRoutes>(
    method: Method,
    route: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ): this {
    const path = route.startsWith('/') ? route : `/${route}`
    this._RoutesMetadata.push({
      method,
      path,
      handler,
      schema,
    })
    return this
  }

  // detailLog: DetailLog | undefined
  // summaryLog: SummaryLog | undefined

  routes() {
    return this._RoutesMetadata
  }

  protected initializeLogs(req: ICtx, scenario: string, identity?: string, schema?: SchemaCtx) {
    const initInvoke = generateInternalTid(scenario, '-', 20)
    const detailLog = new DetailLog(req.session, req.invoke, scenario, identity)
    const summaryLog = new SummaryLog(req.session, req.invoke, scenario)

    detailLog.addInputRequest(NODE_NAME.CLIENT, scenario, initInvoke, req)

    if (schema) {
      const validation = this.validateRequest(req, schema)
      if (validation.err) {
        detailLog.addOutputRequest(NODE_NAME.CLIENT, scenario, initInvoke, '', validation.result_data).end()

        summaryLog.addErrorBlock(NODE_NAME.CLIENT, scenario, 'null', validation.result_desc)
        summaryLog.end('400', validation.result_desc)

        throw new ErrorValidator({
          code: 400,
          message: validation.result_desc,
          response: validation.result_data,
        })
      }
    }

    return { detailLog, summaryLog, initInvoke }
  }

  private funcGetParams(url: string, route: string): Record<string, string> {
    const routeParts = route.split('/').filter(Boolean)
    const urlParts = url.split('/').filter(Boolean)

    const params: Record<string, string> = {}

    routeParts.forEach((part, index) => {
      if (part.startsWith(':')) {
        params[part.substring(1)] = urlParts[index] || ''
      }
    })

    return params
  }

  public get<PathName extends `/${string}`, Schema extends SchemaRoutes>(
    path: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ) {
    return this.route(HttpMethod.GET, path, handler, schema)
  }

  public post<PathName extends `/${string}`, Schema extends SchemaRoutes>(
    path: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ) {
    return this.route(HttpMethod.POST, path, handler, schema)
  }

  public put<PathName extends `/${string}`, Schema extends SchemaRoutes>(
    path: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ) {
    return this.route(HttpMethod.PUT, path, handler, schema)
  }

  public delete<PathName extends `/${string}`, Schema extends SchemaRoutes>(
    path: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ) {
    return this.route(HttpMethod.DELETE, path, handler, schema)
  }

  public patch<PathName extends `/${string}`, Schema extends SchemaRoutes>(
    path: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ) {
    return this.route(HttpMethod.PATCH, path, handler, schema)
  }

  private validateRequest(ctx: IInComing, schemas: SchemaCtx) {
    try {
      if (schemas?.body) {
        const result = TypeCompiler.Compile(schemas.body)
        if (!result.Check(ctx.body)) {
          const first = result.Errors(ctx.body).First()
          if (first) {
            throw first
          }
        }
      }
      if (schemas?.query) {
        const result = TypeCompiler.Compile(schemas.query)
        if (!result.Check(ctx.query)) {
          const first = result.Errors(ctx.query).First()
          if (first) {
            throw first
          }
        }
      }

      return {
        err: false,
        result_desc: 'success',
        result_data: {},
      }
    } catch (error) {
      const err = error as { path: string; message: string }
      const details = {
        name: err?.path.startsWith('/') ? err.path.replace('/', '') : err.path,
        message: err?.message || 'Unknown error',
      }

      return {
        err: true,
        result_desc: 'invalid_request',
        result_data: details,
      }
    }
  }
}

class ErrorValidator extends Error {
  public readonly code: number
  public readonly response: Object
  constructor({ code, message, response }: { code: number; message: string; response: Object }) {
    super(message)
    this.code = code
    this.response = response
    this.name = 'CustomError'
  }
}

interface IServer {
  start(cb?: Function): void
  get<PathName extends `/${string}`, Schema extends SchemaRoutes>(
    path: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ): this
  post<PathName extends `/${string}`, Schema extends SchemaRoutes>(
    path: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ): this
  put<PathName extends `/${string}`, Schema extends SchemaRoutes>(
    path: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ): this
  delete<PathName extends `/${string}`, Schema extends SchemaRoutes>(
    path: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ): this
  patch<PathName extends `/${string}`, Schema extends SchemaRoutes>(
    path: PathName,
    handler: Handler<PathName, Schema>,
    schema?: Schema
  ): this
  init(cb: Function): this
  register(...routes: AppRoutes[]): this
  // consume<Schema extends Omit<SchemaCtx, 'query'>>(
  //   pattern: string,
  //   handler: MessageHandler<Schema>,
  //   schema?: Schema
  // ): void

  consume<BodySchema extends TSchema, HeaderSchema extends TSchema>(
    pattern: string,
    handler: (ctx: CtxConsumer<Static<BodySchema>, Static<HeaderSchema>>) => Promise<any>,
    schema?: TSchemaCtx<BodySchema, HeaderSchema>
  ): void
}

export {
  IServer,
  AppRoutes,
  SchemaCtx,
  Context,
  HandlerContext,
  CommonLog,
  ErrorValidator,
  HandlerConsumer,
  SchemaRoutes,
}
