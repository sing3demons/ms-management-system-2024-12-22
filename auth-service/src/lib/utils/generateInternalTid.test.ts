import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import generateInternalTid from './generateInternalTid'

describe('generateInternalTid', () => {
    beforeEach(() => { })
    afterEach(() => {
        vi.restoreAllMocks()
    })
    test('should generate a string of the specified length', () => {
        const result = generateInternalTid('test', '-', 20)
        expect(result).toBeDefined()
        expect(result.length).toBe(20)
    })
    test('should generate a string with the correct prefix', () => {
        const result = generateInternalTid('test', '-', 20)
        expect(result.startsWith('test-')).toBe(true)
    })

    test('should generate a string with the correct random part', () => {
        const result = generateInternalTid('test', '-', 20)
        const randomPart = result.substring(10)
        expect(randomPart).toMatch(/^[a-zA-Z0-9]+$/) // Alphanumeric
    })

    test('should handle long node names', () => {
        const result = generateInternalTid('longnodename', '-', 20)
        expect(result.startsWith('longn')).toBe(true)
    })

    test('should handle zero length', () => {
        const result = generateInternalTid('test', '-', 0);
        expect(result.length).toBe(0);
    } );

    test('should handle negative length', () => {
        const result = generateInternalTid('test', '-', -5);
        expect(result.length).toBe(0);
    });
    test('should handle empty node name', () => {
        const result = generateInternalTid('', '-', 20);
        expect(result.startsWith('-')).toBe(true);
    } );
    test('should handle empty symbol', () => {
        const result = generateInternalTid('test', '', 20)
        expect(result.startsWith('test')).toBe(true)
    })
})
