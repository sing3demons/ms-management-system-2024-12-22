import pkg from 'pg' // For ES modules
const { Pool } = pkg

type EMethod = 'INSERT' | 'UPDATE' | 'DELETE' | 'FIND' | 'FIND_ONE'

type ResultSQL<T> = {
  err: boolean
  result_desc: string
  result_data: T
  outgoing_detail: {
    Query: string
    Parameters: any[]
  }
  ingoing_detail: {
    Body: any
    RawData: string
  }
}

type TDocument<T> = {
  table: string
  values?: Partial<T>
  condition?: string
  conditionParams?: any[]
  returning?: string[]
}

let client: pkg.Pool

interface DbConnection extends pkg.PoolConfig {
  host: string
  port: number
  user: string
  password: string
  database: string
}

async function initPostgres(conn: DbConnection) {
  try {
    client = new Pool({ ...conn })
    await client.connect()
  } catch (error) {
    console.error('Database connection failed:', error)
  }
}

async function sql<T extends object>(method: EMethod, document: TDocument<T[]>) {
  const result: ResultSQL<T[]> = {
    err: false,
    result_desc: 'success',
    result_data: [] as T[],
    outgoing_detail: {
      Query: '',
      Parameters: [],
    },
    ingoing_detail: {
      Body: {},
      RawData: '',
    },
  }

  try {
    let query = ''
    let parameters: any[] = []

    switch (method) {
      case 'INSERT':
        if (!document.values || !document.table) {
          throw new Error(`'values' and 'table' are required for ${method}`)
        }

        const columns = Object.keys(document.values).join(', ')
        const values = Object.values(document.values)
        const placeholders = values.map((_, idx) => `$${idx + 1}`).join(', ')

        query = `INSERT INTO ${document.table} (${columns}) VALUES (${placeholders})`
        if (document.returning && document.returning.length) {
          query += ` RETURNING ${document.returning.join(', ')}`
        }
        parameters = values
        break

      case 'UPDATE':
        if (!document.values || !document.table || !document.condition) {
          throw new Error(`'values', 'table', and 'condition' are required for ${method}`)
        }

        const setClause = Object.entries(document.values)
          .map(([key, _], idx) => `${key} = $${idx + 1}`)
          .join(', ')

        query = `UPDATE ${document.table} SET ${setClause} WHERE ${document.condition}`
        if (document.returning && document.returning.length) {
          query += ` RETURNING ${document.returning.join(', ')}`
        }
        parameters = [...Object.values(document.values), ...(document.conditionParams || [])]
        break

      case 'DELETE':
        if (!document.table || !document.condition) {
          throw new Error(`'table' and 'condition' are required for ${method}`)
        }

        query = `DELETE FROM ${document.table} WHERE ${document.condition}`
        if (document.returning && document.returning.length) {
          query += ` RETURNING ${document.returning.join(', ')}`
        }
        parameters = document.conditionParams || []
        break

      case 'FIND':
        if (!document.table) {
          throw new Error(`'table' is required for ${method}`)
        }

        const returning = document.returning?.length ? document.returning.join(', ') : '*'
        query = `SELECT ${returning} FROM ${document.table}`
        if (document.condition) {
          query += ` WHERE ${document.condition}`
        }
        parameters = document.conditionParams || []
        break

      case 'FIND_ONE':
        if (!document.table || !document.condition) {
          throw new Error(`'table' and 'condition' are required for ${method}`)
        }

        const returningFields = document.returning?.length ? document.returning.join(', ') : '*'
        query = `SELECT ${returningFields} FROM ${document.table} WHERE ${document.condition} LIMIT 1`
        parameters = document.conditionParams || []
        break

      default:
        throw new Error(`Unsupported method: ${method}`)
    }

    result.outgoing_detail = { Query: query, Parameters: parameters }

    const queryResult = await client.query(query, parameters)

    result.result_data = queryResult.rows as T[]

    result.ingoing_detail = {
      Body: { Return: queryResult.rows },
      RawData: JSON.stringify(queryResult.rows),
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

export { initPostgres, sql }
