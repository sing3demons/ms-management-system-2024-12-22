import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { produceMessage, kafka, initKafka } from './kafka'
import DetailLog from './logger/detail.js'
import SummaryLog from './logger/summary.js'

vi.mock('./logger/detail.js', () => {
    return {
        __esModule: true,
        default: vi.fn().mockImplementation(() => ({
            addInputRequest: vi.fn(),
            addOutputResponse: vi.fn(),
            addErrorBlock: vi.fn(),
            addSuccessBlock: vi.fn(),
            addField: vi.fn(),
            endASync: vi.fn(),
            isEnd: vi.fn(),
            addOutputRequest: vi.fn(),
            addOutputRequestRetry: vi.fn(),
            addOutputResponseRetry: vi.fn(),
            addOutput: vi.fn(),
            addInput: vi.fn(),
            addInputResponse: vi.fn(),
            addInputResponseError: vi.fn(),
            end: vi.fn(),
        })),
    }
})

vi.mock('./logger/summary.js', () => {
    return {
        __esModule: true,
        default: vi.fn().mockImplementation(() => ({
            addInputRequest: vi.fn(),
            addOutputResponse: vi.fn(),
            addErrorBlock: vi.fn(),
            addSuccessBlock: vi.fn(),
            addField: vi.fn(),
            endASync: vi.fn(),
            isEnd: vi.fn(),
            addOutputRequest: vi.fn(),
            addOutputRequestRetry: vi.fn(),
            addOutputResponseRetry: vi.fn(),
            addOutput: vi.fn(),
            addInput: vi.fn(),
            addInputResponse: vi.fn(),
            addInputResponseError: vi.fn(),
        })),
    }
})

describe('produceMessage', () => {
    const mockDetailLog = new DetailLog('mockSession')
    const mockSummaryLog = new SummaryLog('mockSession')

    const OLD_ENV = process.env

    beforeEach(() => {
        // mock env
        process.env = {
            ...OLD_ENV,
            KAFKA_BROKERS: 'localhost:9092',
            KAFKA_CLIENT_ID: 'test-client',
            KAFKA_REQUEST_TIMEOUT: '30000',
            KAFKA_RETRY: '8',
            KAFKA_INITIAL_RETRY_TIME: '100',
            KAFKAJS_NO_PARTITIONER_WARNING: '1',
            KAFKA_USERNAME: 'test-username',
            KAFKA_PASSWORD: 'test-password',
        }

        initKafka()
        kafka.producer = {
            connect: vi.fn(),
            send: vi.fn(),
            disconnect: vi.fn(),
        } as any
    })

    afterEach(() => {
        vi.clearAllMocks()

        // Reset the kafka object
        kafka.producer = undefined
        kafka.consumer = undefined

        // Reset the environment variables
        process.env = OLD_ENV
    })

    it('should return success when producer sends messages successfully', async () => {
        const mockRecordMetadata = [{ topicName: 'test_topic', partition: 0, offset: '0' }]
        kafka.producer!.send = vi.fn().mockResolvedValue(mockRecordMetadata)

        const result = await produceMessage('test.command', { key: 'value' }, mockDetailLog, mockSummaryLog)

        expect(result).toEqual({
            err: false,
            result_desc: 'Success',
            result_data: mockRecordMetadata,
        })
        expect(kafka.producer!.connect).toHaveBeenCalled()
        expect(kafka.producer!.send).toHaveBeenCalledWith({
            topic: 'test.command',
            messages: [{ value: '{"key":"value"}' }],
        })
        expect(kafka.producer!.disconnect).toHaveBeenCalled()
    })

    // payload is an array
    it('should return success when producer sends array of messages successfully', async () => {
        const mockRecordMetadata = [{ topicName: 'test_topic', partition: 0, offset: '0' }]
        kafka.producer!.send = vi.fn().mockResolvedValue(mockRecordMetadata)
        const data = [
            { key: 'value1' },
            { key: 'value2' },
        ]

        const result = await produceMessage('test.command', data, mockDetailLog, mockSummaryLog)
        expect(result).toEqual({
            err: false,
            result_desc: 'Success',
            result_data: mockRecordMetadata,
        })
        expect(kafka.producer!.connect).toHaveBeenCalled()
        expect(kafka.producer!.send).toHaveBeenCalledWith({
            topic: 'test.command',
            messages: [
                { value: '{"key":"value1"}' },
                { value: '{"key":"value2"}' },
            ],
        })
        expect(kafka.producer!.disconnect).toHaveBeenCalled()
    })

    it('should return success when producer sends string messages successfully', async () => {
        const mockRecordMetadata = [{ topicName: 'test_topic', partition: 0, offset: '0' }]
        kafka.producer!.send = vi.fn().mockResolvedValue(mockRecordMetadata)
        const data = 'test message'

        const result = await produceMessage('test.command', data, mockDetailLog, mockSummaryLog)

        expect(result).toEqual({
            err: false,
            result_desc: 'Success',
            result_data: mockRecordMetadata,
        })
        expect(kafka.producer!.connect).toHaveBeenCalled()
        expect(kafka.producer!.send).toHaveBeenCalledWith({
            topic: 'test.command',
            messages: [{ value: data }],
        })
        expect(kafka.producer!.disconnect).toHaveBeenCalled()
    })
    it('should return error when producer send fails', async () => {
        const error = new Error('Failed to send message')
        kafka.producer!.send = vi.fn().mockRejectedValue(error)

        await produceMessage('test.command', { key: 'value' }, mockDetailLog, mockSummaryLog)
        expect(mockSummaryLog.addErrorBlock).toHaveBeenCalledWith(
            expect.anything(),
            'test.command',
            '',
            'system error'
        )
        expect(mockDetailLog.addInputResponseError).toHaveBeenCalledWith(
            expect.anything(),
            'test.command',
            expect.anything(),
            error.message
        )
        expect(kafka.producer!.disconnect).toHaveBeenCalled()
        expect(mockSummaryLog.addField).toHaveBeenCalledWith('errorCause', "Failed to send message")

    })

    it('should return error when producer is not initialized', async () => {
        kafka.producer = undefined

        const result = await produceMessage('test.command', { key: 'value' }, mockDetailLog, mockSummaryLog)

        expect(result).toEqual({
            err: true,
            result_desc: 'Producer not initialized',
        })
    })

    it('should handle connection timeout error', async () => {
        const error = new Error('Connection timeout')
        kafka.producer!.send = vi.fn().mockRejectedValue(error)

        const result = await produceMessage('test.command', { key: 'value' }, mockDetailLog, mockSummaryLog)

        expect(result).toEqual({
            err: true,
            result_desc: 'Connection timeout',
        })
        expect(mockSummaryLog.addErrorBlock).toHaveBeenCalledWith(expect.anything(), 'test.command', 'ret=4', 'Connection timeout')
    })

    it('should handle connection error', async () => {
        const error = new Error('Connection error')
        kafka.producer!.send = vi.fn().mockRejectedValue(error)

        const result = await produceMessage('test.command', { key: 'value' }, mockDetailLog, mockSummaryLog)

        expect(result).toEqual({
            err: true,
            result_desc: 'Connection error',
        })
        expect(mockSummaryLog.addErrorBlock).toHaveBeenCalledWith(expect.anything(), 'test.command', 'ret=1', 'Connection error')
    })

    it('should handle generic system error', async () => {
        const error = new Error('Generic error')
        kafka.producer!.send = vi.fn().mockRejectedValue(error)

        const result = await produceMessage('test.command', { key: 'value' }, mockDetailLog, mockSummaryLog)

        expect(result).toEqual({
            err: true,
            result_desc: 'Generic error',
        })
        expect(mockSummaryLog.addErrorBlock).toHaveBeenCalledWith(expect.anything(), 'test.command', '', 'system error')
    })

    it('should handle error', async () => {
        kafka.producer!.send = vi.fn().mockImplementation(() => {
            throw  'Generic error'  as unknown as Error
        })

        const result = await produceMessage('test.command', { key: 'value' }, mockDetailLog, mockSummaryLog)

        expect(result).toEqual({
            err: true,
            result_desc: 'Generic error',
        })
        expect(mockSummaryLog.addErrorBlock).toHaveBeenCalledWith(expect.anything(), 'test.command', '', 'system error')
    })
})
