import type { BinaryOperation, CalculatorKey, Digit } from '../types/calculator'

type KeyInput = Pick<KeyboardEvent, 'key' | 'ctrlKey' | 'metaKey' | 'altKey'>

const DIGITS: readonly string[] = ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9']

function isDigit(key: string): key is Digit {
  return DIGITS.includes(key)
}

const OPERATOR_KEYS: Record<string, BinaryOperation> = {
  '+': 'add',
  '-': 'subtract',
  '*': 'multiply',
  x: 'multiply',
  '/': 'divide',
  '^': 'power',
}

const COMMAND_KEYS: Record<string, CalculatorKey> = {
  '.': { type: 'decimal' },
  ',': { type: 'decimal' },
  Enter: { type: 'equals' },
  '=': { type: 'equals' },
  Backspace: { type: 'backspace' },
  Escape: { type: 'clear' },
  Delete: { type: 'clear' },
  '%': { type: 'percent' },
  r: { type: 'sqrt' },
  R: { type: 'sqrt' },
  n: { type: 'sign' },
}

// keyToCalculatorKey maps a keyboard event to a calculator key, or null when
// the key isn't handled. Shortcuts with Ctrl, Meta or Alt are left alone.
export function keyToCalculatorKey(event: KeyInput): CalculatorKey | null {
  if (event.ctrlKey || event.metaKey || event.altKey) return null
  const { key } = event
  if (isDigit(key)) return { type: 'digit', digit: key }
  const operator = OPERATOR_KEYS[key]
  if (operator) return { type: 'operator', operator }
  return COMMAND_KEYS[key] ?? null
}
