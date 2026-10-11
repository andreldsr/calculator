import { describe, expect, it } from 'vitest'
import { keyToCalculatorKey } from './keyboard'
import type { CalculatorKey } from '../types/calculator'

const noModifiers = { ctrlKey: false, metaKey: false, altKey: false }

describe('keyToCalculatorKey', () => {
  it.each<{ key: string; want: CalculatorKey }>([
    { key: '0', want: { type: 'digit', digit: '0' } },
    { key: '7', want: { type: 'digit', digit: '7' } },
    { key: '9', want: { type: 'digit', digit: '9' } },
    { key: '.', want: { type: 'decimal' } },
    { key: ',', want: { type: 'decimal' } },
    { key: '+', want: { type: 'operator', operator: 'add' } },
    { key: '-', want: { type: 'operator', operator: 'subtract' } },
    { key: '*', want: { type: 'operator', operator: 'multiply' } },
    { key: 'x', want: { type: 'operator', operator: 'multiply' } },
    { key: '/', want: { type: 'operator', operator: 'divide' } },
    { key: '^', want: { type: 'operator', operator: 'power' } },
    { key: 'Enter', want: { type: 'equals' } },
    { key: '=', want: { type: 'equals' } },
    { key: 'Backspace', want: { type: 'backspace' } },
    { key: 'Escape', want: { type: 'clear' } },
    { key: 'Delete', want: { type: 'clear' } },
    { key: '%', want: { type: 'percent' } },
    { key: 'r', want: { type: 'sqrt' } },
    { key: 'R', want: { type: 'sqrt' } },
    { key: 'n', want: { type: 'sign' } },
  ])('maps "$key"', ({ key, want }) => {
    expect(keyToCalculatorKey({ key, ...noModifiers })).toEqual(want)
  })

  it.each(['a', ' ', 'Tab', 'ArrowUp', 'Shift', 'F5', '10'])('ignores "%s"', (key) => {
    expect(keyToCalculatorKey({ key, ...noModifiers })).toBeNull()
  })

  it.each([
    { ctrlKey: true, metaKey: false, altKey: false },
    { ctrlKey: false, metaKey: true, altKey: false },
    { ctrlKey: false, metaKey: false, altKey: true },
  ])('ignores keys with modifiers %o', (modifiers) => {
    expect(keyToCalculatorKey({ key: 'r', ...modifiers })).toBeNull()
  })
})
