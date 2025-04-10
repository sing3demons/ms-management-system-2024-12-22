import AppServer, { AppRoutes, t } from './index'
import { describe, test, expect, vi, afterEach } from 'vitest'

describe('AppRoutes', () => {
    afterEach(() => {
        vi.restoreAllMocks()
    })

    test('should have a defined routes property', () => {
        const appRoutes = new AppRoutes()
        expect(appRoutes.routes).toBeDefined()
        expect(appRoutes.routes.length).toBeGreaterThanOrEqual(0)
    })

    // /api/v1/health
    test('should have a health route', () => {
        const appRoutes = new AppRoutes()
        appRoutes.get(
            '/health',
            async (ctx) => {
                const cmd = 'auth'
                const logger = ctx.commonLog(cmd, 'test')
                logger.detailLog.addInputRequest('client', cmd, 'test', ctx)

                logger.summaryLog.addSuccessBlock('client', cmd, 'null', 'success')

                return ctx.response(200, { status: 'ok' })
            },
            {
                headers: t.Object({
                    'Content-Type': t.String(),
                }),
            }
        )

        const routes = appRoutes.routes()
        const healthRoute = routes.find((route) => route.path === '/health' && route.method === 'get')
        expect(healthRoute).toBeDefined()
    })

    test('should have a health route with schema', () => {
        const app = new AppServer('express').load({ Addr: '8080' })

        app.get(
            '/health',
            async (ctx) => {
                const cmd = 'auth'
                const logger = ctx.commonLog(cmd, 'test')
                logger.detailLog.addInputRequest('client', cmd, 'test', ctx)

                logger.summaryLog.addSuccessBlock('client', cmd, 'null', 'success')

                return ctx.response(200, { status: 'ok' })
            },
            {
                headers: t.Object({
                    'Content-Type': t.String(),
                }),
            }
        )

        expect(app).toBeDefined()
        expect(app.routes).toBeDefined()
    })
})
