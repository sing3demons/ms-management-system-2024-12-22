import { generateXTid } from './index';
import { describe, test, expect, vi, afterEach } from 'vitest'

describe('generateXTid', () => {
    afterEach(() => {
        vi.restoreAllMocks()
    })

    test('should generate a string of the specified length', () => {
        const result = generateXTid('test')
        expect(result).toBeDefined()
        expect(result.length).toBe(22)
    })

    test('should generate a string with the correct prefix', () => {
        const result = generateXTid('test')
        expect(result.startsWith('test-')).toBe(true)
    })

    test('should handle long node names', () => {
        const result = generateXTid('longnodename')
        expect(result.startsWith('longn')).toBe(true)
    })

    test('should handle empty node name', () => {
        const result = generateXTid('')
        expect(result.startsWith('-')).toBe(true)
    })
});