import { describe, test, expect, vi, beforeAll, afterAll } from 'vitest'
import SummaryLog from './summary'
import { confLog, writeLogFile } from './logger.js'
import commandName from '../constants/commandName.js'

vi.mock('./logger.js', () => ({
  confLog: {
    projectName: 'TestProject',
    detail: {
      rawData: true,
      console: true,
      file: true,
    },
    namespace: 'TestNamespace',
    summary: {
      console: true,
      file: true,
    },
  },
  writeLogFile: vi.fn(),
}))

describe('SummaryLog', () => {
  beforeAll(() => {
    vi.spyOn(process.stdout, 'write').mockImplementation(vi.fn())
  })

  afterAll(() => {
    vi.restoreAllMocks()
  })

  test('should initialize with default values', () => {
    const summaryLog = new SummaryLog('session123')
    expect(summaryLog).toBeDefined()
  })

  test('should create a new scenario', () => {
    const summaryLog = new SummaryLog('session123')
    summaryLog.New('testScenario')
    expect(summaryLog).toBeDefined()
  })

  test('should add a success block', () => {
    const summaryLog = new SummaryLog('session123')
    summaryLog.addSuccessBlock('node1', commandName.LOGIN, '200', 'Success')
    summaryLog.addSuccessBlock('node1', commandName.LOGIN, '200', 'Success')
    summaryLog.addSuccessBlock('node1', commandName.LOGIN, '200', 'Success')
    summaryLog.addField('customField', 'customValue')
    summaryLog.addErrorBlock('node1', commandName.GET_USER, '500', 'Error')
    summaryLog.addSuccessBlock('node1', commandName.GET_USER, '200', 'Success')
    summaryLog.addErrorBlock('node1', commandName.GET_USER, '500', 'Error')
    summaryLog.addSuccessBlock('node1', commandName.REGISTER, '200', 'Success')
    summaryLog.addErrorBlock('node1', commandName.REGISTER, '500', 'Error')
    summaryLog.end("", "")
    expect(summaryLog).toBeDefined()
  })

  test('should add an error block', () => {
    const summaryLog = new SummaryLog('session123')
    summaryLog.addErrorBlock('node1', 'cmd1', '500', 'Error')
    expect(summaryLog).toBeDefined()
  })

  test('should add a custom field', () => {
    const summaryLog = new SummaryLog('session123')
    summaryLog.addField('customField', 'customValue')
    expect(summaryLog).toBeDefined()
  })

  test('should end the log with success', () => {
    const summaryLog = new SummaryLog('session123')
    summaryLog.end('200', 'Success')
    expect(summaryLog.isEnd()).toBe(true)
  })

  test('should throw error when ending twice', () => {
    const summaryLog = new SummaryLog('session123')
    summaryLog.end('200', 'Success')
    expect(() => summaryLog.end('200', 'Success')).toThrowError('summaryLog is ended')
  })

  test('should process async end', () => {
    const summaryLog = new SummaryLog('session123')
    summaryLog.endASync('200', 'Success', 'TransactionSuccess', 'Transaction completed')
    expect(summaryLog.isEnd()).toBe(true)
  })

  test('should throw error when async end is called twice', () => {
    const summaryLog = new SummaryLog('session123')
    summaryLog.endASync('200', 'Success', 'TransactionSuccess', 'Transaction completed')
    expect(() => summaryLog.endASync('200', 'Success', 'TransactionSuccess', 'Transaction completed')).toThrowError('summaryLog is ended')
  })

  test('should write log to console and file', () => {
    const summaryLog = new SummaryLog('session123')
    summaryLog.end('200', 'Success')
    expect(process.stdout.write).toHaveBeenCalled()
    expect(writeLogFile).toHaveBeenCalledWith(expect.any(String), expect.any(String))
  })
})