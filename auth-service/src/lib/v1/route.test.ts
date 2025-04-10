import { describe, test, expect, vi, beforeAll, afterAll } from 'vitest'
import { AppRouter, } from './route';
import { Static, TSchema, Type } from '@sinclair/typebox'
import AppServer from './route'

describe('AppRouter', () => {
    const app = new AppServer()

    beforeAll(() => {
        vi.spyOn(process.stdout, 'write').mockImplementation(vi.fn())
    })

    afterAll(() => {
        vi.restoreAllMocks()
    })

    test('should initialize with default values', () => {
        const appRouter = new AppRouter()
        appRouter.get('/test', async (c) => {
            c.res.json({ data: 'test' })
        })
        appRouter.post('/test', async (c) => {
        
            c.res.json({ data: 'test' })
        },
            {
            body: Type.Object({
                name: Type.String(),
                age: Type.Number(),
            }),
        })
        appRouter.put('/test', async (c) => {
            c.res.json({ data: 'test' })
        })
        appRouter.delete('/test', async (c) => {
            c.res.json({ data: 'test' })
        })
        appRouter.patch('/test/:id', async (c) => {
            c.res.json({ id: c.params.id })
        })
        app.use(appRouter.register())
        expect(appRouter.register()).toBeDefined()
    })

    // test('should create a new scenario', () => {
    //     const appRouter = new AppRouter('session123')
    //     appRouter.New('testScenario')
    //     expect(appRouter).toBeDefined()
    // })
})