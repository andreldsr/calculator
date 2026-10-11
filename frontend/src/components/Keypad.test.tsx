import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { Keypad } from './Keypad'
import type { BinaryOperation, CalculatorKey } from '../types/calculator'

function renderKeypad(activeOperator: BinaryOperation | null = null, disabled = false) {
  const onPress = vi.fn<(key: CalculatorKey) => void>()
  render(<Keypad onPress={onPress} activeOperator={activeOperator} disabled={disabled} />)
  return onPress
}

const BUTTONS: { name: string; key: CalculatorKey }[] = [
  ...(['0', '1', '2', '3', '4', '5', '6', '7', '8', '9'] as const).map((digit) => ({
    name: digit,
    key: { type: 'digit', digit } as const,
  })),
  { name: 'decimal point', key: { type: 'decimal' } },
  { name: 'toggle sign', key: { type: 'sign' } },
  { name: 'plus', key: { type: 'operator', operator: 'add' } },
  { name: 'minus', key: { type: 'operator', operator: 'subtract' } },
  { name: 'times', key: { type: 'operator', operator: 'multiply' } },
  { name: 'divided by', key: { type: 'operator', operator: 'divide' } },
  { name: 'to the power of', key: { type: 'operator', operator: 'power' } },
  { name: 'square root', key: { type: 'sqrt' } },
  { name: 'percent', key: { type: 'percent' } },
  { name: 'equals', key: { type: 'equals' } },
  { name: 'all clear', key: { type: 'clear' } },
  { name: 'backspace', key: { type: 'backspace' } },
]

describe('Keypad', () => {
  it('groups its buttons under a labeled group', () => {
    renderKeypad()

    const group = screen.getByRole('group', { name: 'Keypad' })
    expect(within(group).getAllByRole('button')).toHaveLength(BUTTONS.length)
  })

  it.each(BUTTONS)('"$name" sends its key', async ({ name, key }) => {
    const onPress = renderKeypad()

    await userEvent.click(screen.getByRole('button', { name }))

    expect(onPress).toHaveBeenCalledExactlyOnceWith(key)
  })

  it('uses type="button" everywhere', () => {
    renderKeypad()

    for (const button of screen.getAllByRole('button')) {
      expect(button).toHaveAttribute('type', 'button')
    }
  })

  it('marks only the active operator as pressed', () => {
    renderKeypad('multiply')

    expect(screen.getByRole('button', { name: 'times' })).toHaveAttribute('aria-pressed', 'true')
    for (const name of ['plus', 'minus', 'divided by', 'to the power of']) {
      expect(screen.getByRole('button', { name })).toHaveAttribute('aria-pressed', 'false')
    }
    expect(screen.getByRole('button', { name: '7' })).not.toHaveAttribute('aria-pressed')
  })

  it('disables every key except all clear', async () => {
    const onPress = renderKeypad(null, true)

    for (const button of screen.getAllByRole('button')) {
      const isClear = button.getAttribute('aria-label') === 'all clear'
      expect(button).toHaveProperty('disabled', !isClear)
    }
    await userEvent.click(screen.getByRole('button', { name: 'all clear' }))
    expect(onPress).toHaveBeenCalledExactlyOnceWith({ type: 'clear' })
  })
})
