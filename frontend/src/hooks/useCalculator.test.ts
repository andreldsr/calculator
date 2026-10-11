import { act, renderHook, waitFor, type RenderHookResult } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { initialState, reducer, useCalculator, type UseCalculator } from './useCalculator'
import { calculate } from '../services/api'
import type {
  BinaryOperation,
  CalculateOutcome,
  CalculateRequest,
  CalculatorKey,
  Digit,
} from '../types/calculator'

vi.mock('../services/api', () => ({ calculate: vi.fn() }))
const calculateMock = vi.mocked(calculate)

// fakeCalculate mirrors the backend's math and its domain errors.
function fakeCalculate({ operation, a, b = 0 }: CalculateRequest): CalculateOutcome {
  switch (operation) {
    case 'add':
      return { ok: true, result: a + b }
    case 'subtract':
      return { ok: true, result: a - b }
    case 'multiply':
      return { ok: true, result: a * b }
    case 'divide':
      return b === 0
        ? { ok: false, error: 'division by zero is not allowed' }
        : { ok: true, result: a / b }
    case 'power':
      return { ok: true, result: a ** b }
    case 'sqrt':
      return a < 0
        ? { ok: false, error: 'square root of a negative number is not allowed' }
        : { ok: true, result: Math.sqrt(a) }
    case 'percentage':
      return { ok: true, result: (a * b) / 100 }
  }
}

beforeEach(() => {
  calculateMock.mockReset()
  calculateMock.mockImplementation((req) => Promise.resolve(fakeCalculate(req)))
})

const OPERATOR_TOKENS: Record<string, BinaryOperation> = {
  '+': 'add',
  '-': 'subtract',
  '*': 'multiply',
  '/': 'divide',
  '^': 'power',
}

const COMMAND_TOKENS: Record<string, CalculatorKey> = {
  '=': { type: 'equals' },
  '√': { type: 'sqrt' },
  '%': { type: 'percent' },
  '±': { type: 'sign' },
  AC: { type: 'clear' },
  '⌫': { type: 'backspace' },
}

function isDigit(char: string): char is Digit {
  return /^[0-9]$/.test(char)
}

// parseKeys turns "12.5 + 3 =" into key presses; number tokens become one
// key per character.
function parseKeys(sequence: string): CalculatorKey[] {
  return sequence
    .split(' ')
    .filter((token) => token !== '')
    .flatMap((token): CalculatorKey[] => {
      const operator = OPERATOR_TOKENS[token]
      if (operator) return [{ type: 'operator', operator }]
      const command = COMMAND_TOKENS[token]
      if (command) return [command]
      return [...token].map((char): CalculatorKey => {
        if (char === '.') return { type: 'decimal' }
        if (isDigit(char)) return { type: 'digit', digit: char }
        throw new Error(`unknown token ${token}`)
      })
    })
}

async function pressAll(hook: RenderHookResult<UseCalculator, unknown>, sequence: string) {
  for (const key of parseKeys(sequence)) {
    act(() => hook.result.current.press(key))
    await waitFor(() => expect(hook.result.current.isLoading).toBe(false))
  }
}

async function run(sequence: string) {
  const hook = renderHook(() => useCalculator())
  await pressAll(hook, sequence)
  return hook.result.current
}

function deferred() {
  let resolve: (outcome: CalculateOutcome) => void = () => {}
  let reject: (err: unknown) => void = () => {}
  const promise = new Promise<CalculateOutcome>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useCalculator transitions', () => {
  it.each<{
    keys: string
    display: string
    expression?: string
    pendingOperator?: BinaryOperation | null
    error?: string | null
  }>([
    // entry
    { keys: '', display: '0', expression: '', pendingOperator: null, error: null },
    { keys: '123', display: '123' },
    { keys: '007', display: '7' },
    { keys: '1.5', display: '1.5' },
    { keys: '1..5', display: '1.5' },
    { keys: '1.2.3', display: '1.23' },
    { keys: '.', display: '0.' },
    { keys: '0.05', display: '0.05' },
    // Capped at 15 digits, the most a double holds exactly.
    { keys: '1234567890123456', display: '123456789012345' },
    { keys: '0.12345678901234567', display: '0.12345678901234' },
    { keys: '1.2345678901234 ±', display: '-1.2345678901234' },
    // sign
    { keys: '±', display: '0' },
    { keys: '5 ±', display: '-5' },
    { keys: '5 ± ±', display: '5' },
    { keys: '5 ± 3', display: '-53' },
    { keys: '5 + ± 3 =', display: '2' },
    { keys: '2 + 3 = ±', display: '-5' },
    { keys: '2 + 3 = ± * 2 =', display: '-10' },
    // backspace
    { keys: '123 ⌫', display: '12' },
    { keys: '5 ⌫', display: '0' },
    { keys: '5 ± ⌫', display: '0' },
    { keys: '1.5 ⌫ ⌫', display: '1' },
    { keys: '2 + 3 = ⌫', display: '5' },
    { keys: '2 + ⌫', display: '2', pendingOperator: 'add' },
    // binary operators, left to right
    { keys: '2 +', display: '2', expression: '2 +', pendingOperator: 'add' },
    { keys: '2 + 3', display: '3', expression: '2 +', pendingOperator: 'add' },
    { keys: '2 + 3 =', display: '5', expression: '2 + 3 =', pendingOperator: null },
    { keys: '2 + 3 * 4 =', display: '20' },
    { keys: '2 + 3 *', display: '5', expression: '5 ×', pendingOperator: 'multiply' },
    { keys: '2 + * 3 =', display: '6' },
    { keys: '9 - 12 =', display: '-3' },
    { keys: '10 / 4 =', display: '2.5' },
    { keys: '2 ^ 10 =', display: '1024' },
    { keys: '0.1 + 0.2 =', display: '0.3' },
    { keys: '5 + =', display: '5', pendingOperator: 'add' },
    { keys: '5 =', display: '5', pendingOperator: null },
    { keys: '2 + 3 = =', display: '5' },
    { keys: '2 + 3 = + 1 =', display: '6' },
    { keys: '2 + 3 = 4', display: '4', expression: '', pendingOperator: null },
    { keys: '2 + 3 = .', display: '0.', expression: '' },
    { keys: '2 + .5 =', display: '2.5' },
    // unary operations
    { keys: '16 √', display: '4', expression: '√16' },
    { keys: '16 √ 9', display: '9', expression: '' },
    { keys: '9 + 16 √', display: '4', expression: '9 + √16', pendingOperator: 'add' },
    { keys: '9 + 16 √ =', display: '13' },
    { keys: '50 %', display: '0.5', expression: '50%' },
    { keys: '200 + 10 %', display: '20', expression: '200 + 10% of 200', pendingOperator: 'add' },
    { keys: '200 + 10 % =', display: '220' },
    { keys: '200 + 10 % * 2 =', display: '440' },
    // Right after an operator, √ and % act on the left operand still displayed.
    { keys: '9 + √', display: '3', expression: '9 + √9', pendingOperator: 'add' },
    { keys: '9 + √ =', display: '12' },
    { keys: '200 + %', display: '400', expression: '200 + 200% of 200' },
    { keys: '200 + % =', display: '600' },
    // operator, then ±, starts a "-0" entry that backspace resets
    { keys: '5 + ±', display: '-0', pendingOperator: 'add' },
    { keys: '5 + ± ⌫', display: '0', pendingOperator: 'add' },
    { keys: '5 + ± ⌫ 3 =', display: '8' },
    // clear
    { keys: '2 + 3 AC', display: '0', expression: '', pendingOperator: null },
    // errors, and the next key clearing them
    { keys: '10 / 0 =', display: '0', error: 'division by zero is not allowed', pendingOperator: null },
    { keys: '4 ± √', error: 'square root of a negative number is not allowed', display: '-4' },
    { keys: '10 / 0 = 7', display: '7', error: null, expression: '' },
    { keys: '10 / 0 = AC', display: '0', error: null },
    { keys: '10 / 0 = ⌫', display: '0', error: null },
    { keys: '10 / 0 = + 1', display: '1', error: null, pendingOperator: 'add' },
    { keys: '10 / 0 = 9 √', display: '3', error: null },
  ])('"$keys" shows $display', async ({ keys, display, expression, pendingOperator, error }) => {
    const state = await run(keys)

    expect(state.display).toBe(display)
    if (expression !== undefined) expect(state.expression).toBe(expression)
    if (pendingOperator !== undefined) expect(state.pendingOperator).toBe(pendingOperator)
    if (error !== undefined) expect(state.error).toBe(error)
  })
})

describe('useCalculator backend calls', () => {
  it.each<{ keys: string; calls: CalculateRequest[] }>([
    { keys: '123 . 4 ± ⌫', calls: [] },
    { keys: '5 + * - ^ =', calls: [] },
    { keys: '2 + 3 =', calls: [{ operation: 'add', a: 2, b: 3 }] },
    {
      keys: '2 + 3 * 4 =',
      calls: [
        { operation: 'add', a: 2, b: 3 },
        { operation: 'multiply', a: 5, b: 4 },
      ],
    },
    {
      // The chained operand is the displayed 0.3, not 0.30000000000000004.
      keys: '0.1 + 0.2 + 1 =',
      calls: [
        { operation: 'add', a: 0.1, b: 0.2 },
        { operation: 'add', a: 0.3, b: 1 },
      ],
    },
    {
      // The longest entry is sent exactly as typed.
      keys: '999999999999999 + 1 =',
      calls: [{ operation: 'add', a: 999999999999999, b: 1 }],
    },
    { keys: '16 √', calls: [{ operation: 'sqrt', a: 16 }] },
    {
      keys: '9 + √ =',
      calls: [
        { operation: 'sqrt', a: 9 },
        { operation: 'add', a: 9, b: 3 },
      ],
    },
    {
      keys: '200 + % =',
      calls: [
        { operation: 'percentage', a: 200, b: 200 },
        { operation: 'add', a: 200, b: 400 },
      ],
    },
    { keys: '50 %', calls: [{ operation: 'percentage', a: 50, b: 1 }] },
    {
      keys: '200 + 10 % =',
      calls: [
        { operation: 'percentage', a: 10, b: 200 },
        { operation: 'add', a: 200, b: 20 },
      ],
    },
  ])('"$keys" makes one call per operation', async ({ keys, calls }) => {
    await run(keys)

    expect(calculateMock.mock.calls.map(([req]) => req)).toEqual(calls)
  })
})

describe('useCalculator history', () => {
  it('records successful calculations, newest first', async () => {
    const state = await run('2 + 3 * 4 = 16 √')

    expect(state.history.map(({ operation, a, b, result }) => ({ operation, a, b, result }))).toEqual([
      { operation: 'sqrt', a: 16, b: undefined, result: 4 },
      { operation: 'multiply', a: 5, b: 4, result: 20 },
      { operation: 'add', a: 2, b: 3, result: 5 },
    ])
    expect(new Set(state.history.map((entry) => entry.id)).size).toBe(3)
  })

  it('does not record failures', async () => {
    const state = await run('1 + 1 = 10 / 0 =')

    expect(state.history).toHaveLength(1)
  })

  it('keeps only the last 10', async () => {
    const state = await run(`0${' + 1'.repeat(12)} =`)

    expect(state.history).toHaveLength(10)
    expect(state.history[0]).toMatchObject({ operation: 'add', a: 11, b: 1, result: 12 })
    expect(state.history[9]).toMatchObject({ a: 2, b: 1, result: 3 })
  })

  it('survives AC and errors', async () => {
    const state = await run('1 + 1 = AC 4 ± √ 7')

    expect(state.history).toHaveLength(1)
  })
})

describe('useCalculator while a request is in flight', () => {
  it('is loading and ignores keys other than AC', async () => {
    const pending = deferred()
    calculateMock.mockReturnValueOnce(pending.promise)
    const hook = renderHook(() => useCalculator())
    await pressAll(hook, '2 + 3')

    act(() => hook.result.current.press({ type: 'equals' }))
    expect(hook.result.current.isLoading).toBe(true)
    act(() => hook.result.current.press({ type: 'digit', digit: '9' }))
    act(() => hook.result.current.press({ type: 'operator', operator: 'multiply' }))
    expect(hook.result.current.display).toBe('3')

    await act(async () => pending.resolve({ ok: true, result: 5 }))
    expect(hook.result.current.isLoading).toBe(false)
    expect(hook.result.current.display).toBe('5')
    expect(calculateMock).toHaveBeenCalledTimes(1)
  })

  it('aborts the request on AC and ignores its late result', async () => {
    const pending = deferred()
    calculateMock.mockReturnValueOnce(pending.promise)
    const hook = renderHook(() => useCalculator())
    await pressAll(hook, '2 + 3')
    act(() => hook.result.current.press({ type: 'equals' }))
    const signal = calculateMock.mock.calls[0]?.[1]

    act(() => hook.result.current.press({ type: 'clear' }))
    expect(signal?.aborted).toBe(true)
    expect(hook.result.current.isLoading).toBe(false)

    await act(async () => pending.resolve({ ok: true, result: 5 }))
    expect(hook.result.current.display).toBe('0')
    expect(hook.result.current.history).toHaveLength(0)
  })

  it('ignores the abort rejection of a request cancelled by AC', async () => {
    const pending = deferred()
    calculateMock.mockReturnValueOnce(pending.promise)
    const hook = renderHook(() => useCalculator())
    await pressAll(hook, '16')
    act(() => hook.result.current.press({ type: 'sqrt' }))
    act(() => hook.result.current.press({ type: 'clear' }))

    await act(async () => pending.reject(new DOMException('aborted', 'AbortError')))

    expect(hook.result.current.error).toBeNull()
    expect(hook.result.current.display).toBe('0')
  })

  it('aborts the request on unmount', async () => {
    calculateMock.mockReturnValueOnce(deferred().promise)
    const hook = renderHook(() => useCalculator())
    await pressAll(hook, '16')
    act(() => hook.result.current.press({ type: 'sqrt' }))
    const signal = calculateMock.mock.calls[0]?.[1]

    hook.unmount()

    expect(signal?.aborted).toBe(true)
  })

  it('reports an unexpected rejection as an error', async () => {
    calculateMock.mockRejectedValueOnce(new Error('boom'))
    const state = await run('16 √')

    expect(state.error).toBe('Something went wrong. Please try again.')
  })
})

describe('reducer', () => {
  it('ignores a result for a request that is no longer current', () => {
    const waiting = reducer(initialState, { type: 'press', key: { type: 'sqrt' } })
    expect(waiting.request).not.toBeNull()

    const stale = reducer(waiting, {
      type: 'resolve',
      id: (waiting.request?.id ?? 0) + 1,
      outcome: { ok: true, result: 9 },
    })

    expect(stale).toBe(waiting)
  })

  it('ignores a result when no request is in flight', () => {
    const next = reducer(initialState, { type: 'resolve', id: 1, outcome: { ok: true, result: 9 } })

    expect(next).toBe(initialState)
  })
})
