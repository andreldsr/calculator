// API contract for POST /api/v1/calculate. Mirrors backend/internal/handler.

export type BinaryOperation = 'add' | 'subtract' | 'multiply' | 'divide' | 'power'

// percentage(a, b) is a% of b; sqrt uses only a.
export type Operation = BinaryOperation | 'sqrt' | 'percentage'

export interface CalculateRequest {
  operation: Operation
  a: number
  b?: number
}

// 200 response body.
export interface CalculateSuccess {
  result: number
}

// Error envelope for 400, 405, 413 and 500 responses.
export interface CalculateFailure {
  error: string
}

// What the service hands back: it never throws for HTTP or network errors.
export type CalculateOutcome = { ok: true; result: number } | { ok: false; error: string }

// UI types.

export type Digit = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9'

export type CalculatorKey =
  | { type: 'digit'; digit: Digit }
  | { type: 'decimal' }
  | { type: 'sign' }
  | { type: 'operator'; operator: BinaryOperation }
  | { type: 'equals' }
  | { type: 'sqrt' }
  | { type: 'percent' }
  | { type: 'clear' }
  | { type: 'backspace' }

export interface HistoryEntry {
  id: number
  operation: Operation
  a: number
  b?: number
  result: number
}
