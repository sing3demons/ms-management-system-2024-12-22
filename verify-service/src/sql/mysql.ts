import mysql from 'mysql2/promise'

type EMethod =
  | 'insert'
  | 'update'
  | 'delete'
  | 'select'
  | 'findOne'

type ResultSQL = {
  err: boolean
  result_desc: string
  result_data: any
  outgoing_detail: {
    Query: string
    Params: any[]
  }
  ingoing_detail: {
    Result: any
    RawData: string
  }
}

type TDocument<T> = {
  filter?: Partial<T>
  new?: Partial<T>
  options?: {
    returnFields?: string[]
    sort?: Record<string, 'ASC' | 'DESC'>
  }
}

let connection: mysql.Connection

async function initSQL() {
  try {
    connection = await mysql.createConnection({
      host: 'localhost',
      user: 'root',
      password: 'password',
      database: 'verify_service',
    })
    console.log('Database connected')
  } catch (error) {
    console.error('Database connection failed:', error)
  }
}

async function sql<T>(
  tableName: string,
  method: EMethod,
  document: TDocument<T>
): Promise<ResultSQL> {
  const result: ResultSQL = {
    err: false,
    result_desc: 'success',
    result_data: null,
    outgoing_detail: {
      Query: '',
      Params: [],
    },
    ingoing_detail: {
      Result: null,
      RawData: '',
    },
  }

  try {
    let query = ''
    let params: any[] = []
    let operationResult: any

    switch (method) {
      case 'insert':
        if (!document.new) {
          throw new Error(`'new' data is required for ${method}`)
        }
        const fields = Object.keys(document.new)
        const placeholders = fields.map(() => '?').join(', ')
        query = `INSERT INTO ${tableName} (${fields.join(', ')}) VALUES (${placeholders})`
        params = Object.values(document.new)
        ;[operationResult] = await connection.execute(query, params)
        break

      case 'update':
        if (!document.filter || !document.new) {
          throw new Error(`'filter' and 'new' data are required for ${method}`)
        }
        const updateFields = Object.keys(document.new)
          .map((field) => `${field} = ?`)
          .join(', ')
        const whereConditions = Object.keys(document.filter)
          .map((field) => `${field} = ?`)
          .join(' AND ')
        query = `UPDATE ${tableName} SET ${updateFields} WHERE ${whereConditions}`
        params = [...Object.values(document.new), ...Object.values(document.filter)]
        ;[operationResult] = await connection.execute(query, params)
        break

      case 'delete':
        if (!document.filter) {
          throw new Error(`'filter' is required for ${method}`)
        }
        const deleteConditions = Object.keys(document.filter)
          .map((field) => `${field} = ?`)
          .join(' AND ')
        query = `DELETE FROM ${tableName} WHERE ${deleteConditions}`
        params = Object.values(document.filter)
        ;[operationResult] = await connection.execute(query, params)
        break

      case 'select':
        const selectFields = document.options?.returnFields?.join(', ') || '*'
        const selectConditions = document.filter
          ? 'WHERE ' +
            Object.keys(document.filter)
              .map((field) => `${field} = ?`)
              .join(' AND ')
          : ''
        const sortClause =
          document.options?.sort &&
          Object.entries(document.options.sort)
            .map(([field, order]) => `${field} ${order}`)
            .join(', ')
        query = `SELECT ${selectFields} FROM ${tableName} ${selectConditions} ${
          sortClause ? 'ORDER BY ' + sortClause : ''
        }`
        params = document.filter ? Object.values(document.filter) : []
        ;[operationResult] = await connection.execute(query, params)
        break

      case 'findOne':
        const findOneFields = document.options?.returnFields?.join(', ') || '*'
        const findOneConditions = document.filter
          ? 'WHERE ' +
            Object.keys(document.filter)
              .map((field) => `${field} = ?`)
              .join(' AND ')
          : ''
        query = `SELECT ${findOneFields} FROM ${tableName} ${findOneConditions} LIMIT 1`
        params = document.filter ? Object.values(document.filter) : []
        ;[operationResult] = await connection.execute(query, params)
        operationResult = operationResult[0] || null
        break

      default:
        throw new Error(`Unsupported method: ${method}`)
    }

    if (!operationResult) {
      throw new Error('Operation failed or no rows affected')
    }

    result.result_data = operationResult
    result.outgoing_detail = {
      Query: query,
      Params: params,
    }
    result.ingoing_detail = {
      Result: operationResult,
      RawData: JSON.stringify(operationResult),
    }

    return result
  } catch (error) {
    result.err = true
    result.result_desc = error instanceof Error ? error.message : 'Unknown error'
    result.ingoing_detail = {
      Result: error,
      RawData: result.result_desc,
    }
    return result
  }
}

export { initSQL, sql }
