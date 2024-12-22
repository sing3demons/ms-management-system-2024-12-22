// import * as express from 'express'
// import Fastify, { FastifyRequest, FastifyReply, FastifyInstance } from 'fastify'
import 'fastify'

import type DetailLog from '../logger/detail.js'
import type SummaryLog from '../logger/summary.js'

export type RequestContext = {
  session: string
  invoke: string
  userId?: string
  detailLog: DetailLog
  summaryLog: SummaryLog
  deviceInfo: {
    browser: string
    version: string
    os: string
    device: string
  }
}

declare global {
  namespace Express {
    interface Request extends RequestContext {}
  }
}

declare module 'fastify' {
  interface FastifyRequest extends RequestContext {}
  // interface FastifyReply { json: (body: unknown) => void }
}
