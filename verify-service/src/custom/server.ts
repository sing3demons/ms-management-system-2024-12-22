import express, { Request, Response, NextFunction } from 'express'
import {
  RouteHandler,
  Context,
  RouteSchema,
  HTTPHeaders,
  HTTPMethod,
  HigherOrderFunction,
  InternalRoute,
} from './context.js'
import { Static, Type } from '@sinclair/typebox'

// Example Route Schema
const userRouteSchema: RouteSchema = {
  params: { id: 'string' },
  query: { search: 'string' },
  headers: { 'x-custom-header': 'string' },
  response: { 200: { message: 'string', user: { id: 'string', name: 'string' } } },
}

// Example Route Handler
const getUserHandler: RouteHandler<typeof userRouteSchema, '/user/:id'> = async (
  context: Context<typeof userRouteSchema, '/user/:id'>
) => {
  const { params, query, headers } = context

  // Validate the data (mocked here for simplicity)
  if (!params?.id || typeof params.id !== 'string') {
    throw new Error('Invalid user ID')
  }

  return {
    status: 200,
    message: 'User fetched successfully',
    user: {
      id: params.id,
      name: query.search || 'Anonymous',
    },
  }
}

export type InlineHandler<Route extends RouteSchema = {}, Path extends string | undefined = undefined> =
  | RouteHandler<Route, Path>
  | RouteHandler<Route, Path>[]

class AppRouter {
  _routes: InternalRoute[] = []

  private add(
    method: HTTPMethod,
    path: string,
    handler: InlineHandler<any, any> | InlineHandler<any, any>[],
    hook?: {
      before?: HigherOrderFunction
      after?: HigherOrderFunction
      schema: RouteSchema
    }
  ) {
    this._routes.push({ method, path, handler, hook })
    return this
  }

  public get<const Path extends string, const Route extends RouteSchema>(
    path: Path,
    handler: InlineHandler<Route, Path> | InlineHandler<Route, Path>[],
    hook?: {
      before?: HigherOrderFunction
      after?: HigherOrderFunction
      schema: Route
    }
  ) {
    return this.add(HTTPMethod.GET, path, handler, hook)
  }
}

const app = new AppRouter()
app.get('/user/:id', async ({ params: { id }, body }) => {}, {
  schema: {
    body: Type.Object({
      id: Type.String(),
    }),
    params: Type.Object({
      id: Type.String(),
    }),
  },
})
