import type { TypeCheck, ValueError } from '@sinclair/typebox/compiler'

export interface RouteSchema {
  body?: unknown
  headers?: unknown
  query?: unknown
  params?: unknown
  cookie?: unknown
  response?: unknown
}

export type Prettify<T> = {
  [K in keyof T]: T[K]
} & {}

type IsPathParameter<Part extends string> = Part extends `:${infer Parameter}`
  ? Parameter
  : Part extends `*`
  ? '*'
  : never

export type GetPathParameter<Path extends string> = Path extends `${infer A}/${infer B}`
  ? IsPathParameter<A> | GetPathParameter<B>
  : IsPathParameter<Path>

export type ResolvePath<Path extends string> = Prettify<
  {
    [Param in GetPathParameter<Path> as Param extends `${string}?` ? never : Param]: string
  } & {
    [Param in GetPathParameter<Path> as Param extends `${infer OptionalParam}?` ? OptionalParam : never]?: string
  }
>

export type HigherOrderFunction<T extends (...arg: unknown[]) => Function = (...arg: unknown[]) => Function> = (
  fn: T,
  request: Request
) => ReturnType<T>

type SetContentType =
  | 'application/octet-stream'
  | 'application/vnd.ms-fontobject'
  | 'application/epub+zip'
  | 'application/gzip'
  | 'application/json'
  | 'application/ld+json'
  | 'application/ogg'
  | 'application/pdf'
  | 'application/rtf'
  | 'application/wasm'
  | 'application/xhtml+xml'
  | 'application/xml'
  | 'application/zip'
  | 'text/css'
  | 'text/csv'
  | 'text/html'
  | 'text/calendar'
  | 'text/javascript'
  | 'text/plain'
  | 'text/xml'
  | 'image/avif'
  | 'image/bmp'
  | 'image/gif'
  | 'image/x-icon'
  | 'image/jpeg'
  | 'image/png'
  | 'image/svg+xml'
  | 'image/tiff'
  | 'image/webp'
  | 'multipart/mixed'
  | 'multipart/alternative'
  | 'multipart/form-data'
  | 'audio/aac'
  | 'audio/x-midi'
  | 'audio/mpeg'
  | 'audio/ogg'
  | 'audio/opus'
  | 'audio/webm'
  | 'video/x-msvideo'
  | 'video/quicktime'
  | 'video/x-ms-wmv'
  | 'video/x-msvideo'
  | 'video/x-flv'
  | 'video/av1'
  | 'video/mp4'
  | 'video/mpeg'
  | 'video/ogg'
  | 'video/mp2t'
  | 'video/webm'
  | 'video/3gpp'
  | 'video/3gpp2'
  | 'font/otf'
  | 'font/ttf'
  | 'font/woff'
  | 'font/woff2'
  | 'model/gltf+json'
  | 'model/gltf-binary'

export type HTTPHeaders = Record<string, string | number> & {
  // Authentication
  'www-authenticate'?: string
  authorization?: string
  'proxy-authenticate'?: string
  'proxy-authorization'?: string

  // Caching
  age?: string
  'cache-control'?: string
  'clear-site-data'?: string
  expires?: string
  'no-vary-search'?: string
  pragma?: string

  // Conditionals
  'last-modified'?: string
  etag?: string
  'if-match'?: string
  'if-none-match'?: string
  'if-modified-since'?: string
  'if-unmodified-since'?: string
  vary?: string

  // Connection management
  connection?: string
  'keep-alive'?: string

  // Content negotiation
  accept?: string
  'accept-encoding'?: string
  'accept-language'?: string

  // Controls
  expect?: string
  'max-forwards'?: string

  // Cokies
  cookie?: string
  'set-cookie'?: string | string[]

  // CORS
  'access-control-allow-origin'?: string
  'access-control-allow-credentials'?: string
  'access-control-allow-headers'?: string
  'access-control-allow-methods'?: string
  'access-control-expose-headers'?: string
  'access-control-max-age'?: string
  'access-control-request-headers'?: string
  'access-control-request-method'?: string
  origin?: string
  'timing-allow-origin'?: string

  // Downloads
  'content-disposition'?: string

  // Message body information
  'content-length'?: string | number
  'content-type'?: SetContentType | (string & {})
  'content-encoding'?: string
  'content-language'?: string
  'content-location'?: string

  // Proxies
  forwarded?: string
  via?: string

  // Redirects
  location?: string
  refresh?: string

  // Request context
  // from?: string
  // host?: string
  // referer?: string
  // 'user-agent'?: string

  // Response context
  allow?: string
  server?: 'Elysia' | (string & {})

  // Range requests
  'accept-ranges'?: string
  range?: string
  'if-range'?: string
  'content-range'?: string

  // Security
  'content-security-policy'?: string
  'content-security-policy-report-only'?: string
  'cross-origin-embedder-policy'?: string
  'cross-origin-opener-policy'?: string
  'cross-origin-resource-policy'?: string
  'expect-ct'?: string
  'permission-policy'?: string
  'strict-transport-security'?: string
  'upgrade-insecure-requests'?: string
  'x-content-type-options'?: string
  'x-frame-options'?: string
  'x-xss-protection'?: string

  // Server-sent events
  'last-event-id'?: string
  'ping-from'?: string
  'ping-to'?: string
  'report-to'?: string

  // Transfer coding
  te?: string
  trailer?: string
  'transfer-encoding'?: string

  // Other
  'alt-svg'?: string
  'alt-used'?: string
  date?: string
  dnt?: string
  'early-data'?: string
  'large-allocation'?: string
  link?: string
  'retry-after'?: string
  'service-worker-allowed'?: string
  'source-map'?: string
  upgrade?: string

  // Non-standard
  'x-dns-prefetch-control'?: string
  'x-forwarded-for'?: string
  'x-forwarded-host'?: string
  'x-forwarded-proto'?: string
  'x-powered-by'?: 'Elysia' | (string & {})
  'x-request-id'?: string
  'x-requested-with'?: string
  'x-robots-tag'?: string
  'x-ua-compatible'?: string
}

export const statusMap = {
  Continue: 100,
  'Switching Protocols': 101,
  Processing: 102,
  'Early Hints': 103,
  OK: 200,
  Created: 201,
  Accepted: 202,
  'Non-Authoritative Information': 203,
  'No Content': 204,
  'Reset Content': 205,
  'Partial Content': 206,
  'Multi-Status': 207,
  'Already Reported': 208,
  'Multiple Choices': 300,
  'Moved Permanently': 301,
  Found: 302,
  'See Other': 303,
  'Not Modified': 304,
  'Temporary Redirect': 307,
  'Permanent Redirect': 308,
  'Bad Request': 400,
  Unauthorized: 401,
  'Payment Required': 402,
  Forbidden: 403,
  'Not Found': 404,
  'Method Not Allowed': 405,
  'Not Acceptable': 406,
  'Proxy Authentication Required': 407,
  'Request Timeout': 408,
  Conflict: 409,
  Gone: 410,
  'Length Required': 411,
  'Precondition Failed': 412,
  'Payload Too Large': 413,
  'URI Too Long': 414,
  'Unsupported Media Type': 415,
  'Range Not Satisfiable': 416,
  'Expectation Failed': 417,
  "I'm a teapot": 418,
  'Misdirected Request': 421,
  'Unprocessable Content': 422,
  Locked: 423,
  'Failed Dependency': 424,
  'Too Early': 425,
  'Upgrade Required': 426,
  'Precondition Required': 428,
  'Too Many Requests': 429,
  'Request Header Fields Too Large': 431,
  'Unavailable For Legal Reasons': 451,
  'Internal Server Error': 500,
  'Not Implemented': 501,
  'Bad Gateway': 502,
  'Service Unavailable': 503,
  'Gateway Timeout': 504,
  'HTTP Version Not Supported': 505,
  'Variant Also Negotiates': 506,
  'Insufficient Storage': 507,
  'Loop Detected': 508,
  'Not Extended': 510,
  'Network Authentication Required': 511,
} as const

export const InvertedStatusMap = Object.fromEntries(Object.entries(statusMap).map(([k, v]) => [v, k])) as {
  [K in keyof StatusMap as StatusMap[K]]: K
}

export type StatusMap = typeof statusMap
export type InvertedStatusMap = typeof InvertedStatusMap
export interface CookieOptions {
  domain?: string | undefined
  es?: Date | undefined
  httpOnly?: boolean | undefined
  maxAge?: number | undefined
  path?: string | undefined
  priority?: 'low' | 'medium' | 'high' | undefined
  partitioned?: boolean | undefined
  sameSite?: true | false | 'lax' | 'strict' | 'none' | undefined
  secure?: boolean | undefined
  secrets?: string | string[]
}

export type Cookie = Prettify<
  CookieOptions & {
    value?: unknown
  }
>

export type Redirect = (url: string, status?: number) => void

export type Context<Route extends RouteSchema = {}, Path extends string | undefined = undefined> = Prettify<{
  body: Route['body']
  query: undefined extends Route['query'] ? Record<string, string | undefined> : Route['query']
  params: undefined extends Route['params']
    ? undefined extends Path
      ? Record<string, string>
      : Path extends `${string}/${':' | '*'}${string}`
      ? ResolvePath<Path>
      : never
    : Route['params']
  headers: undefined extends Route['headers'] ? Record<string, string | undefined> : Route['headers']
  // cookie: undefined extends Route['cookie']
  //   ? Record<string, Cookie<string | undefined>>
  //   : Record<string, Cookie<string | undefined>> &
  //       Prettify<
  //         WithoutNullableKeys<{
  //           [key in keyof Route['cookie']]: Cookie<Route['cookie'][key]>
  //         }>
  //       >

  // server: Server | null
  redirect: Redirect

  set: {
    headers: HTTPHeaders
    status?: number | keyof StatusMap
    redirect?: string
    cookie?: Record<string, Cookie>
  }

  path: string
  route: string
  request: Request
  response?: undefined extends Route['response'] ? unknown : Route['response'][keyof Route['response']]
}>

export type RouteHandler<Route extends RouteSchema = {}, Path extends string | undefined = undefined> = (
  context: Context<Route, Path>
) => Promise<unknown>

function get<const Path extends string, Route extends RouteSchema>(
  path: Path,
  handler: RouteHandler<Route, Path>,
  hook?: {
    before?: HigherOrderFunction
    after?: HigherOrderFunction
    schema: Route
  }
) {}

get(
  '/user/:id',
  async (ctx) => {
    ctx.params
    ctx.body
    ctx.set.headers = {
      'accept-encoding': 'gzip',
    }

    ctx.response
  },
  {
    schema: {
      body: {
        id: 'string',
      },
    },
    before: (fn, req) => {
      return fn(req)
    },
  }
)

export enum HTTPMethod {
  GET = 'GET',
  POST = 'POST',
  PUT = 'PUT',
  DELETE = 'DELETE',
  PATCH = 'PATCH',
  OPTIONS = 'OPTIONS',
  HEAD = 'HEAD',
  CONNECT = 'CONNECT',
  TRACE = 'TRACE',
}

export type InternalRoute = {
  method: HTTPMethod
  path: string
  handler: RouteHandler<any, any> | RouteHandler<any, any>[] | any
  hook?: {
    before?: HigherOrderFunction
    after?: HigherOrderFunction
    schema: RouteSchema
  }
}
export type SchemaValidator = {
  createBody?(): TypeCheck<any>
  createHeaders?(): TypeCheck<any>
  createQuery?(): TypeCheck<any>
  createParams?(): TypeCheck<any>
  createCookie?(): TypeCheck<any>
  createResponse?(): Record<number, TypeCheck<any>>
  body?: TypeCheck<any>
  headers?: TypeCheck<any>
  query?: TypeCheck<any>
  params?: TypeCheck<any>
  cookie?: TypeCheck<any>
  response?: Record<number, TypeCheck<any>>
}
