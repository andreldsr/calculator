import type { BinaryOperation, CalculatorKey } from '../types/calculator'

interface KeyDef {
  label: string
  ariaLabel?: string
  key: CalculatorKey
  variant?: 'operator' | 'function' | 'equals'
  wide?: boolean
}

function digit(d: '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9'): KeyDef {
  return { label: d, key: { type: 'digit', digit: d } }
}

function operator(label: string, ariaLabel: string, op: BinaryOperation): KeyDef {
  return { label, ariaLabel, key: { type: 'operator', operator: op }, variant: 'operator' }
}

// Row by row, four columns.
const KEYS: readonly KeyDef[] = [
  { label: 'AC', ariaLabel: 'all clear', key: { type: 'clear' }, variant: 'function' },
  { label: '⌫', ariaLabel: 'backspace', key: { type: 'backspace' }, variant: 'function' },
  { label: '%', ariaLabel: 'percent', key: { type: 'percent' }, variant: 'function' },
  operator('÷', 'divided by', 'divide'),

  digit('7'),
  digit('8'),
  digit('9'),
  operator('×', 'times', 'multiply'),

  digit('4'),
  digit('5'),
  digit('6'),
  operator('−', 'minus', 'subtract'),

  digit('1'),
  digit('2'),
  digit('3'),
  operator('+', 'plus', 'add'),

  { label: '±', ariaLabel: 'toggle sign', key: { type: 'sign' } },
  digit('0'),
  { label: '.', ariaLabel: 'decimal point', key: { type: 'decimal' } },
  operator('xʸ', 'to the power of', 'power'),

  { label: '√', ariaLabel: 'square root', key: { type: 'sqrt' }, variant: 'function' },
  { label: '=', ariaLabel: 'equals', key: { type: 'equals' }, variant: 'equals', wide: true },
]

interface KeypadProps {
  onPress: (key: CalculatorKey) => void
  activeOperator: BinaryOperation | null
  // While a request is in flight only AC stays enabled. Keys use aria-disabled
  // rather than disabled, which would drop focus from the key just pressed.
  disabled: boolean
}

export function Keypad({ onPress, activeOperator, disabled }: KeypadProps) {
  return (
    <div className="keypad" role="group" aria-label="Keypad">
      {KEYS.map(({ label, ariaLabel, key, variant, wide }) => {
        const classes = ['key', variant && `key--${variant}`, wide && 'key--wide']
        const inactive = disabled && key.type !== 'clear'
        return (
          <button
            key={label}
            type="button"
            className={classes.filter(Boolean).join(' ')}
            aria-label={ariaLabel}
            aria-pressed={key.type === 'operator' ? key.operator === activeOperator : undefined}
            aria-disabled={inactive}
            // Mouse clicks don't take focus (Chrome and Firefox would), so a
            // following Enter means = rather than repeating the clicked key.
            // Tab focus is unaffected.
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => {
              if (!inactive) onPress(key)
            }}
          >
            {label}
          </button>
        )
      })}
    </div>
  )
}
