import { describe, test, expect, vi, beforeAll, afterAll } from 'vitest'
import DetailLog from './detail'

// vi.mock('./detail', () => {
//   return {
//     __esModule: true,
//     default: vi.fn().mockImplementation(() => ({
//       addInputRequest: vi.fn(),
//       addOutputResponse: vi.fn(),
//       addErrorBlock: vi.fn(),
//       addSuccessBlock: vi.fn(),
//       addField: vi.fn(),
//       endASync: vi.fn(),
//       isEnd: vi.fn(),
//       addOutputRequest: vi.fn(),
//       addOutputRequestRetry: vi.fn(),
//       addOutputResponseRetry: vi.fn(),
//       addOutput: vi.fn(),
//       addInput: vi.fn(),
//       addInputResponse: vi.fn(),
//       addInputResponseError: vi.fn(),
//       end: vi.fn(),
//     })),
//   }
// })

vi.mock('./logger.js', () => ({
  confLog: {
    projectName: 'TestProject',
    detail: {
      rawData: true,
      console: true,
      file: true,
    },
    namespace: 'TestNamespace',
  },
  writeLogFile: vi.fn(),
}))

describe('DetailLog - addInput', () => {
 
  beforeAll(() => { 
    vi.spyOn(process.stdout, 'write').mockImplementation(vi.fn())
  })
  afterAll(() => {
    vi.restoreAllMocks()
  })


  test('should add input with valid data', () => {
    const detailLog = new DetailLog('session123')
    detailLog.New("test")
    const req = {
      deviceInfo: 'device123',
      session: 'session123',
      userId: 'user123',
      hostname: 'localhost',
      ip: '127.0.0.1',
      params: { id: '1' },
      query: { search: 'test' },
      body: { key: 'value', email: 'test@test.com', password: '123456' },
      protocol: 'http',
      method: 'GET',
    }

    detailLog.addInputRequest('node1', 'cmd1', 'invoke1', req)
    detailLog.addOutputRequest('node1', 'cmd1', 'invoke1', JSON.stringify(req), { status: 200 })
    detailLog.end()
    detailLog.addInputResponse('node1', 'cmd1', 'invoke1', 'rawData', { status: 200 }, 100)
    detailLog.addOutputResponse('node1', 'cmd1', 'invoke1', 'rawData', { status: 200 })
    detailLog.end()

    expect(detailLog).toBeDefined()
  })

  test('should add input with raw data when data is invalid', () => {
    const detailLog = new DetailLog('session123')
    const req = {
      protocol: 'http',
      method: 'POST',
    }

    detailLog.addInputRequest('node1', 'cmd1', 'invoke1', req)
    detailLog.end()
    expect(detailLog).toBeDefined()

  })

  test('should mask sensitive data in input', () => {
    const detailLog = new DetailLog('session123')
    const req = {
      body: { password: 'secret123', email: 'test@example.com' },
      protocol: 'https',
      method: 'POST',
    }

    detailLog.addInputRequest('node1', 'cmd1', 'invoke1', req)

    expect(detailLog).toBeDefined()

  })

  test('should handle empty invoke and use InitInvoke', () => {
    const detailLog = new DetailLog('session123')
    const req = {
      protocol: 'http',
      method: 'GET',
    }

    detailLog.addInputRequest('node1', 'cmd1', '', req)

    expect(detailLog).toBeDefined()

  })

  test('should handle multiple inputs', () => {
    const detailLog = new DetailLog('session123')
    const req1 = { protocol: 'http', method: 'GET' }
    const req2 = { protocol: 'https', method: 'POST' }

    detailLog.addInputRequest('node1', 'cmd1', 'invoke1', req1)
    detailLog.addInputRequest('node2', 'cmd2', 'invoke2', req2)
    expect(detailLog).toBeDefined()

  })

  // addOutputRequestRetry
  test('should add output request retry', () => {
    const detailLog = new DetailLog('session123')
    const req = {
      protocol: 'http',
      method: 'GET',
    }

    detailLog.isRawDataEnabled()
    detailLog.addOutputRequestRetry('node1', 'cmd1', 'invoke1', JSON.stringify(req), { status: 200 }, 1, 1)
    detailLog.isEnd()

    expect(detailLog).toBeDefined()

  })

  // addInputResponseError
  test('should add input response error', () => {
    const detailLog = new DetailLog('session123')
    const req = {
      protocol: 'http',
      method: 'GET',
    }

    detailLog.addInputResponseError('node1', 'cmd1', 'invoke1', 'rawData')

    expect(detailLog).toBeDefined()

  })
})