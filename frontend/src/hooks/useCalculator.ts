import { useCallback, useEffect, useReducer } from 'react'
import { calculate } from '../services/api'
import { describeCalculation, formatResult, operatorSymbol } from '../utils/format'
import type {
  BinaryOperation,
  CalculateOutcome,
  CalculateRequest,
  CalculatorKey,
  HistoryEntry,
} from '../types/calculator'

const HISTORY_LIMIT = 10
// Doubles hold 15 significant digits exactly; longer entries would change when sent.
const MAX_ENTRY_DIGITS = 15
const UNEXPECTED_ERROR = 'Something went wrong. Please try again.'

// What the entry on the display currently is:
// - typing: digits the user is typing; new digits append.
// - result: a computed value; new digits replace it, but it counts as an operand.
// - awaiting: an operator was just pressed; the entry still shows the left
//   operand and the right operand hasn't been typed yet.
type EntryMode = 'typing' | 'result' | 'awaiting'

interface PendingOperation {
  operator: BinaryOperation
  operand: number
}

// What to do with the backend result once it arrives.
type Continuation =
  | { kind: 'chain'; operator: BinaryOperation } // binary op, then a new pending operator
  | { kind: 'equals' }
  | { kind: 'unary' } // sqrt or percent; the pending operation is kept

interface InFlightRequest {
  id: number
  request: CalculateRequest
  then: Continuation
}

export interface CalculatorState {
  entry: string
  mode: EntryMode
  pending: PendingOperation | null
  expression: string
  error: string | null
  history: HistoryEntry[]
  request: InFlightRequest | null
  nextId: number
}

export type CalculatorAction =
  | { type: 'press'; key: CalculatorKey }
  | { type: 'resolve'; id: number; outcome: CalculateOutcome }

export const initialState: CalculatorState = {
  entry: '0',
  mode: 'typing',
  pending: null,
  expression: '',
  error: null,
  history: [],
  request: null,
  nextId: 1,
}

// reset clears everything but the history.
function reset(state: CalculatorState): CalculatorState {
  return { ...initialState, history: state.history, nextId: state.nextId }
}

function pendingExpression(pending: PendingOperation): string {
  return `${formatResult(pending.operand)} ${operatorSymbol(pending.operator)}`
}

// startEntry begins a new number after an operator or result; a finished
// calculation's expression is dropped unless an operator is still pending.
function startEntry(state: CalculatorState, entry: string): CalculatorState {
  return {
    ...state,
    entry,
    mode: 'typing',
    expression: state.pending ? state.expression : '',
  }
}

function send(state: CalculatorState, request: CalculateRequest, then: Continuation): CalculatorState {
  return { ...state, request: { id: state.nextId, request, then }, nextId: state.nextId + 1 }
}

function applyKey(state: CalculatorState, key: CalculatorKey): CalculatorState {
  const { entry, mode, pending } = state
  switch (key.type) {
    case 'digit': {
      if (mode !== 'typing') return startEntry(state, key.digit)
      if (entry === '0') return { ...state, entry: key.digit }
      if (entry === '-0') return { ...state, entry: `-${key.digit}` }
      if (entry.replace(/[-.]/g, '').length >= MAX_ENTRY_DIGITS) return state
      return { ...state, entry: entry + key.digit }
    }
    case 'decimal': {
      if (mode !== 'typing') return startEntry(state, '0.')
      if (entry.includes('.')) return state
      return { ...state, entry: `${entry}.` }
    }
    case 'sign': {
      if (mode === 'awaiting') return startEntry(state, '-0')
      if (entry === '0') return state
      return { ...state, entry: entry.startsWith('-') ? entry.slice(1) : `-${entry}` }
    }
    case 'backspace': {
      if (mode !== 'typing') return state
      const shorter = entry.slice(0, -1)
      return { ...state, entry: shorter === '' || shorter === '-' ? '0' : shorter }
    }
    case 'clear':
      return reset(state)
    case 'operator': {
      if (pending === null) {
        const next = { operator: key.operator, operand: Number(entry) }
        return { ...state, pending: next, mode: 'awaiting', expression: pendingExpression(next) }
      }
      if (mode === 'awaiting') {
        const next = { ...pending, operator: key.operator }
        return { ...state, pending: next, expression: pendingExpression(next) }
      }
      return send(
        state,
        { operation: pending.operator, a: pending.operand, b: Number(entry) },
        { kind: 'chain', operator: key.operator },
      )
    }
    case 'equals': {
      if (pending === null || mode === 'awaiting') return state
      return send(
        state,
        { operation: pending.operator, a: pending.operand, b: Number(entry) },
        { kind: 'equals' },
      )
    }
    case 'sqrt':
      return send(state, { operation: 'sqrt', a: Number(entry) }, { kind: 'unary' })
    case 'percent':
      // With an operator pending, x% is x percent of the left operand
      // (200 + 10% → 200 + 20); otherwise it is x / 100.
      return send(
        state,
        { operation: 'percentage', a: Number(entry), b: pending ? pending.operand : 1 },
        { kind: 'unary' },
      )
  }
}

function resolve(state: CalculatorState, { request, then, id }: InFlightRequest, outcome: CalculateOutcome): CalculatorState {
  const done = { ...state, request: null }
  if (!outcome.ok) {
    return { ...done, error: outcome.error, pending: null, mode: 'result', expression: '' }
  }

  const { result } = outcome
  const entry: HistoryEntry = {
    id,
    operation: request.operation,
    a: request.a,
    ...(request.b === undefined ? {} : { b: request.b }),
    result,
  }
  // Later operations use the value the user sees, as `=` does via the entry.
  const shown = formatResult(result)
  const succeeded: CalculatorState = {
    ...done,
    entry: shown,
    mode: 'result',
    history: [entry, ...state.history].slice(0, HISTORY_LIMIT),
  }
  const described = describeCalculation(request.operation, request.a, request.b)

  switch (then.kind) {
    case 'chain': {
      const next = { operator: then.operator, operand: Number(shown) }
      return { ...succeeded, pending: next, mode: 'awaiting', expression: pendingExpression(next) }
    }
    case 'equals':
      return { ...succeeded, pending: null, expression: `${described} =` }
    case 'unary':
      return {
        ...succeeded,
        expression: state.pending ? `${pendingExpression(state.pending)} ${described}` : described,
      }
  }
}

export function reducer(state: CalculatorState, action: CalculatorAction): CalculatorState {
  switch (action.type) {
    case 'press': {
      // While a request is in flight only AC is accepted; it cancels the request.
      if (state.request) return action.key.type === 'clear' ? reset(state) : state
      // Any key dismisses an error, then applies to a fresh calculator.
      return applyKey(state.error ? reset(state) : state, action.key)
    }
    case 'resolve':
      if (state.request?.id !== action.id) return state
      return resolve(state, state.request, action.outcome)
  }
}

export interface UseCalculator {
  display: string
  expression: string
  error: string | null
  isLoading: boolean
  pendingOperator: BinaryOperation | null
  history: readonly HistoryEntry[]
  press: (key: CalculatorKey) => void
}

export function useCalculator(): UseCalculator {
  const [state, dispatch] = useReducer(reducer, initialState)
  const { request } = state

  // Sends the in-flight request; clearing it (AC) or unmounting aborts it.
  useEffect(() => {
    if (!request) return
    const controller = new AbortController()
    calculate(request.request, controller.signal).then(
      (outcome) => dispatch({ type: 'resolve', id: request.id, outcome }),
      // After AC the reducer drops this as stale, and after unmount dispatch is
      // a no-op, so an abort rejection needs no special case here.
      () => dispatch({ type: 'resolve', id: request.id, outcome: { ok: false, error: UNEXPECTED_ERROR } }),
    )
    return () => controller.abort()
  }, [request])

  const press = useCallback((key: CalculatorKey) => dispatch({ type: 'press', key }), [])

  return {
    display: state.entry,
    expression: state.expression,
    error: state.error,
    isLoading: request !== null,
    pendingOperator: state.pending?.operator ?? null,
    history: state.history,
    press,
  }
}
