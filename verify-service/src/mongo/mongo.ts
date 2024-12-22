import { connect, Model, Schema } from 'mongoose'

type EMethod =
  | 'create'
  | 'insertMany'
  | 'updateOne'
  | 'updateMany'
  | 'deleteOne'
  | 'deleteMany'
  | 'findOne'
  | 'find'
  | 'findOneAndUpdate'
  | 'findOneAndDelete'
  | 'findOneAndReplace'

type ResultMongo = {
  err: boolean
  result_desc: string
  result_data: any
  outgoing_detail: {
    Body: any
    RawData: string
  }
}

async function initMongo() {
  try {
    await connect('mongodb://localhost:27017/verify-service')
    console.log('Database connected')
  } catch (error) {
    console.error('Database connection failed')
  }
}

type TDocument<T> = {
  filter?: Record<string, any>
  new?: Partial<T>[] | Record<string, any>
  options?: Record<string, any>
  sort_items?: Record<string, any>
  projection?: Record<string, any>
}

async function mongo<T extends Schema = any>(
  model: Model<T>,
  method: EMethod,
  document: TDocument<T>
): Promise<ResultMongo> {
  const result = {
    err: false,
    result_desc: 'success',
    result_data: {},
    outgoing_detail: {
      Body: {},
      RawData: '',
    },
  }

  const _model = `${model.collection.name}.${method}`
  const processReqLog = {
    Body: {
      Collection: model.collection.name,
      Method: method,
      Query: document.filter ?? {},
      Document: document.new ?? null,
      options: document.options ?? null,
      sort_items: document.sort_items ?? null,
      projection: document.projection ?? null,
    },
    RawData: '',
  }

  try {
    switch (method) {
      case 'create':
      case 'insertMany':
        processReqLog.RawData = `${_model}(${JSON.stringify(document.new).replace(/"/g, "'")}${
          document.options ? ',' + JSON.stringify(document.options).replace(/"/g, "'") : ''
        })`
        result.outgoing_detail = processReqLog

        if (!document.new) {
          throw new Error(`'docs' is required for ${method}`)
        }

        // const insertResult = await model.create(document.new)
        const insertResult = await (model[method] as any)(document.new, document.options)

        if (!insertResult) {
          throw new Error('Insert failed')
        }

        result.result_data = insertResult

        return result

      case 'updateOne':
      case 'updateMany':
        processReqLog.RawData = `${_model}(${JSON.stringify(document.filter).replace(/"/g, "'")},${JSON.stringify(
          document.new
        ).replace(/"/g, "'")}${document.options ? ',' + JSON.stringify(document.options).replace(/"/g, "'") : ''})`
        result.outgoing_detail = processReqLog

        if (!document.filter || !document.new) {
          throw new Error(`'filter' and 'update' are required for ${method}`)
        }

        if (typeof document.filter === 'string') {
          throw new Error(`'filter' must be an object for ${method}`)
        }

        const updateResult = await (
          model[method] as (
            filter: Record<string, any>,
            update: Record<string, any>,
            options?: Record<string, any>
          ) => Promise<any>
        )(document.filter, document.new, document.options)

        if (!updateResult) {
          throw new Error('Update failed')
        }
        result.result_data = updateResult
        return result

      case 'deleteOne':
      case 'deleteMany':
        processReqLog.RawData = `${_model}(${JSON.stringify(document.filter).replace(/"/g, "'")}${
          document.options ? ',' + JSON.stringify(document.options).replace(/"/g, "'") : ''
        })`
        result.outgoing_detail = processReqLog

        if (!document.filter) {
          throw new Error(`'filter' is required for ${method}`)
        }

        result.result_data = await (model[method] as any)(document.filter, document.options)
        return result

      case 'findOne':
        processReqLog.RawData = `${_model}(${JSON.stringify(document.filter).replace(/"/g, "'")}${
          document.options ? ',' + JSON.stringify(document.options).replace(/"/g, "'") : ''
        })`
        result.outgoing_detail = processReqLog

        if (!document.filter) {
          throw new Error(`'filter' is required for ${method}`)
        }

        const findResult = await model[method](document.filter, document.options)
        if (!findResult) {
          throw new Error('Document not found')
        }

        result.result_data = findResult
        return result

      case 'find':
        processReqLog.RawData = `${_model}(${JSON.stringify(document.filter ?? {}).replace(/"/g, "'")}${
          document.options ? ',' + JSON.stringify(document.options).replace(/"/g, "'") : ''
        })`
        result.outgoing_detail = processReqLog

        result.result_data = await model[method](document.filter ?? {}, document.options)
        return result

      case 'findOneAndUpdate':
      case 'findOneAndDelete':
      case 'findOneAndReplace':
        processReqLog.RawData = `${_model}(${JSON.stringify(document.filter).replace(/"/g, "'")},${JSON.stringify(
          document.new
        ).replace(/"/g, "'")}${document.options ? ',' + JSON.stringify(document.options).replace(/"/g, "'") : ''})`
        result.outgoing_detail = processReqLog

        console.log(processReqLog.RawData)

        if (!document.filter || !document.new) {
          throw new Error(`'filter' and 'update' are required for ${method}`)
        }

        if (typeof document.filter === 'string') {
          throw new Error(`'filter' must be an object for ${method}`)
        }
        // await model.findOneAndUpdate(document.filter, document.update, document.options)
        // await model.findOneAndDelete(document.filter, document.options)
        // await model.findOneAndReplace(document.filter, document.update, document.options)
        const result_data = await (
          model[method] as (
            filter: Record<string, any>,
            update: Record<string, any>,
            options?: Record<string, any>
          ) => Promise<any>
        )(document.filter, document.new, document.options)

        if (!result_data) {
          throw new Error('Document not found')
        }

        result.result_data = result_data
        return result

      default:
        throw new Error(`Unsupported method: ${method}`)
    }
  } catch (error) {
    result.err = true
    result.result_desc = 'failed'

    if (error instanceof Error) {
      result.result_desc = error.message
    }

    return result
  }
}

export { mongo, initMongo, TDocument }