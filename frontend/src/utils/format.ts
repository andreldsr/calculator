import type { BinaryOperation, Operation } from '../types/calculator'

// Significant digits kept when displaying a result; drops float noise such as
// 0.1 + 0.2 = 0.30000000000000004.
const DISPLAY_PRECISION = 15

export function formatResult(n: number): string {
  return String(Number(n.toPrecision(DISPLAY_PRECISION)))
}

const OPERATOR_SYMBOLS: Record<BinaryOperation, string> = {
  add: '+',
  subtract: '−',
  multiply: '×',
  divide: '÷',
  power: '^',
}

export function operatorSymbol(op: BinaryOperation): string {
  return OPERATOR_SYMBOLS[op]
}

// describeCalculation renders an operation for the expression line and history,
// e.g. "12 + 3", "√16", "10% of 200". A percentage of 1 is a plain x%.
export function describeCalculation(op: Operation, a: number, b?: number): string {
  const left = formatResult(a)
  const right = b === undefined ? '' : formatResult(b)
  switch (op) {
    case 'sqrt':
      return `√${left}`
    case 'percentage':
      return b === undefined || b === 1 ? `${left}%` : `${left}% of ${right}`
    default:
      return `${left} ${operatorSymbol(op)} ${right}`
  }
}
