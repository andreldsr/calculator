import { describe, expect, it } from 'vitest'
import { describeCalculation, formatResult, operatorSymbol } from './format'
import type { BinaryOperation, Operation } from '../types/calculator'

describe('formatResult', () => {
  it.each([
    { n: 5, want: '5' },
    { n: -2.5, want: '-2.5' },
    { n: 0, want: '0' },
    { n: -0, want: '0' },
    { n: 0.1 + 0.2, want: '0.3' },
    { n: 1 / 3, want: '0.333333333333333' },
    { n: 1e21, want: '1e+21' },
    { n: 1.5e-7, want: '1.5e-7' },
    { n: 123456789012, want: '123456789012' },
  ])('formats $n as $want', ({ n, want }) => {
    expect(formatResult(n)).toBe(want)
  })
})

describe('operatorSymbol', () => {
  it.each<{ op: BinaryOperation; want: string }>([
    { op: 'add', want: '+' },
    { op: 'subtract', want: '−' },
    { op: 'multiply', want: '×' },
    { op: 'divide', want: '÷' },
    { op: 'power', want: '^' },
  ])('maps $op to $want', ({ op, want }) => {
    expect(operatorSymbol(op)).toBe(want)
  })
})

describe('describeCalculation', () => {
  it.each<{ op: Operation; a: number; b?: number; want: string }>([
    { op: 'add', a: 12, b: 3, want: '12 + 3' },
    { op: 'subtract', a: 5, b: -2, want: '5 − -2' },
    { op: 'divide', a: 1, b: 3, want: '1 ÷ 3' },
    { op: 'power', a: 2, b: 10, want: '2 ^ 10' },
    { op: 'sqrt', a: 16, want: '√16' },
    { op: 'percentage', a: 10, b: 200, want: '10% of 200' },
    { op: 'percentage', a: 10, b: 1, want: '10%' },
  ])('describes $op($a, $b) as "$want"', ({ op, a, b, want }) => {
    expect(describeCalculation(op, a, b)).toBe(want)
  })
})
