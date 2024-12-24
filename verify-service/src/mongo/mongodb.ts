import {
  MongoClient,
  Db,
  Collection,
  Document,
  OptionalUnlessRequiredId,
  Filter,
  FindOneAndUpdateOptions,
  MongoClientOptions,
} from 'mongodb'

type EMethod =
  | 'insertOne'
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
  ingoing_detail: {
    Body: any
    RawData: string
  }
}

type TDocument<T> = {
  filter?: Filter<T>
  new?: OptionalUnlessRequiredId<T>
  options?: FindOneAndUpdateOptions | FindOneAndUpdateOptions
  sort_items?: Record<string, any>
  projection?: Record<string, any>
}

let db: Db

interface DbConnection extends MongoClientOptions {
  url: string
}

async function initMongo(conn: DbConnection) {
  try {
    const { url, ...options } = conn
    const client = new MongoClient(url, options)
    await client.connect()
    db = client.db('verify-service')
    console.log('Database connected')
  } catch (error) {
    console.error('Database connection failed:', error)
  }
}

async function mongo<T extends Document>(
  collectionName: string,
  method: EMethod,
  document: TDocument<T>
): Promise<ResultMongo> {
  const result: ResultMongo = {
    err: false,
    result_desc: 'success',
    result_data: {},
    outgoing_detail: {
      Body: {},
      RawData: '',
    },
    ingoing_detail: {
      Body: {},
      RawData: '',
    },
  }

  const collection: Collection<T> = db.collection(collectionName)

  const processReqLog = {
    Body: {
      Collection: collectionName,
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
    let operationResult = null
    const _model = `${collectionName}.${method}`

    switch (method) {
      case 'insertOne':
      case 'insertMany':
        if (!document.new) {
          throw new Error(`'new' must be an object for ${method}`)
        }
        processReqLog.RawData = `${_model}(${JSON.stringify(document.new).replace(/"/g, "'")})`
        result.outgoing_detail = processReqLog
        operationResult = await collection[method](document.new as any)
        break
      case 'updateOne':
      case 'updateMany':
      case 'findOneAndUpdate':
      case 'findOneAndReplace':
        if (!document.filter || !document.new) {
          throw new Error(`'filter' and 'new' are required for ${method}`)
        }
        processReqLog.RawData = `${_model}(${JSON.stringify(document.filter).replace(/"/g, "'")},${JSON.stringify(
          document.new
        ).replace(/"/g, "'")}${document.options ? ',' + JSON.stringify(document.options).replace(/"/g, "'") : ''})`
        result.outgoing_detail = processReqLog

        operationResult = await (collection[method] as typeof collection.updateOne)(
          document.filter,
          document.new,
          document.options
        )
        break

      case 'deleteOne':
      case 'deleteMany':
        if (!document.filter) {
          throw new Error(`'filter' is required for ${method}`)
        }
        processReqLog.RawData = `${_model}(${JSON.stringify(document.filter).replace(/"/g, "'")})`
        result.outgoing_detail = processReqLog

        operationResult = await collection[method](document.filter, document.options)
        break

      case 'findOne':
        processReqLog.RawData = `${_model}(${JSON.stringify(document.filter).replace(/"/g, "'")}, ${JSON.stringify(
          document.options
        ).replace(/"/g, "'")})`
        result.outgoing_detail = processReqLog

        operationResult = await collection[method](document.filter || {}, document.options)
        break

      case 'find':
        if (document.sort_items) {
          document.options = { ...document.options, sort: document.sort_items }
        }

        if (document.projection) {
          document.options = { ...document.options, projection: document.projection }
        }
        processReqLog.RawData = `${_model}(${JSON.stringify(document.filter).replace(/"/g, "'")}, ${JSON.stringify(
          document.options
        ).replace(/"/g, "'")})`
        result.outgoing_detail = processReqLog

        operationResult = await collection[method](document.filter ?? {}, document.options).toArray()
        break

      case 'findOneAndDelete':
        if (!document.filter) {
          throw new Error(`'filter' is required for ${method}`)
        }
        processReqLog.RawData = `${_model}(${JSON.stringify(document.filter).replace(/"/g, "'")})`
        result.outgoing_detail = processReqLog

        operationResult = await collection[method](document.filter)
        break

      default:
        throw new Error(`Unsupported method: ${method}`)
    }

    if (!operationResult) {
      throw new Error('Operation failed')
    }

    result.result_data = operationResult
    result.ingoing_detail = {
      Body: { Return: operationResult },
      RawData: JSON.stringify(operationResult),
    }

    return result
  } catch (error) {
    result.err = true
    result.result_desc = error instanceof Error ? error.message : 'Unknown error'

    result.ingoing_detail = {
      Body: { Return: error },
      RawData: result.result_desc,
    }

    return result
  }
}

export { initMongo, mongo }
