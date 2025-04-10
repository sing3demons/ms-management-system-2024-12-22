import { describe, expect, it, vi } from 'vitest';
import axios, { AxiosError } from 'axios';
import { HttpService, RequestAttributes } from './http-service.js'
import DetailLog from './logger/detail.js';
import SummaryLog from './logger/summary.js';

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


describe('requestHttp', () => {
    const mockDetailLog = new DetailLog('mockSession')
    const mockSummaryLog = new SummaryLog('mockSession')
    it('should handle a single request with default status success', async () => {

        const requestAttributes: RequestAttributes = {
            headers: { 'Content-Type': 'application/json' },
            method: 'GET',
            url: 'https://example.com/api/test/:id',
            _service: 'testService',
            _command: 'testCommand',
            _invoke: 'testInvoke',
            params: { id: '1' },
        };

        vi.spyOn(axios, 'request').mockResolvedValueOnce({
            data: { success: true },
            headers: { 'x-test-header': 'test' },
            status: 200,
            statusText: 'OK',
        });


        const response = await HttpService.requestHttp(requestAttributes, mockDetailLog, mockSummaryLog);

        expect(response).toEqual({
            Body: { success: true },
            Header: { 'x-test-header': 'test' },
            Status: 200,
            StatusText: 'OK',
        });
        expect(mockDetailLog.addOutputRequest).toHaveBeenCalled();
        expect(mockDetailLog.addInputResponse).toHaveBeenCalled();
    });

    it('should handle multiple requests with custom status success', async () => {
        const requestAttributes: RequestAttributes[] = [
            {
                headers: { 'Content-Type': 'application/json' },
                method: 'GET',
                url: 'https://example.com/api/1',
                _service: 'testService1',
                _command: 'testCommand1',
                _invoke: 'testInvoke1',
                statusSuccess: [200, 201],
            },
            {
                headers: { 'Content-Type': 'application/json' },
                method: 'POST',
                url: 'https://example.com/api/2',
                _service: 'testService2',
                _command: 'testCommand2',
                _invoke: 'testInvoke2',
                statusSuccess: [201],
            },
        ];

        vi.spyOn(axios, 'request')
            .mockResolvedValueOnce({
                data: { id: 1 },
                headers: { 'x-test-header': 'test1' },
                status: 200,
                statusText: 'OK',
            })
            .mockResolvedValueOnce({
                data: { id: 2 },
                headers: { 'x-test-header': 'test2' },
                status: 201,
                statusText: 'Created',
            });

        const response = await HttpService.requestHttp(requestAttributes, mockDetailLog, mockSummaryLog);

        expect(response).toEqual([
            {
                Body: { id: 1 },
                Header: { 'x-test-header': 'test1' },
                Status: 200,
                StatusText: 'OK',
            },
            {
                Body: { id: 2 },
                Header: { 'x-test-header': 'test2' },
                Status: 201,
                StatusText: 'Created',
            },
        ]);
    });

    it('should handle request errors and log them', async () => {

        const requestAttributes: RequestAttributes = {
            headers: { 'Content-Type': 'application/json' },
            method: 'GET',
            url: 'https://example.com/api',
            _service: 'testService',
            _command: 'testCommand',
            _invoke: 'testInvoke',
        };

        vi.spyOn(axios, 'request').mockRejectedValueOnce(
            new AxiosError('Network Error', 'ENOTFOUND')
        );

        await HttpService.requestHttp(requestAttributes, mockDetailLog, mockSummaryLog);

        expect(mockDetailLog.addInputResponseError).toHaveBeenCalled();
        expect(mockSummaryLog.addErrorBlock).toHaveBeenCalledWith(
            'testService',
            'testCommand',
            'ret=1',
            'connection error'
        );
    });

    it('should retry on 429 status code with retry-after header', async () => {
        const requestAttributes: RequestAttributes = {
            headers: { 'Content-Type': 'application/json' },
            method: 'GET',
            url: 'https://example.com/api',
            _service: 'testService',
            _command: 'testCommand',
            _invoke: 'testInvoke',
        };

        vi.spyOn(axios, 'request')
            .mockRejectedValueOnce({
                response: {
                    status: 429,
                    headers: { 'retry-after': '1' },
                },
            })
            .mockResolvedValueOnce({
                data: { success: true },
                headers: { 'x-test-header': 'test' },
                status: 200,
                statusText: 'OK',
            });

        const response = await HttpService.requestHttp(requestAttributes);

        expect(response).toEqual({
            Status: 500,
        });
    });
});