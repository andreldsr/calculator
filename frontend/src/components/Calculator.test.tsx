import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { Calculator } from './Calculator'
import { calculate } from '../services/api'
import type { CalculateOutcome, CalculateRequest } from '../types/calculator'

vi.mock('../services/api', () => ({ calculate: vi.fn() }))
const calculateMock = vi.mocked(calculate)

function fakeCalculate({ operation, a, b = 0 }: CalculateRequest): CalculateOutcome {
  if (operation === 'divide' && b === 0) {
    return { ok: false, error: 'division by zero is not allowed' }
  }
  const results: Record<CalculateRequest['operation'], number> = {
    add: a + b,
    subtract: a - b,
    multiply: a * b,
    divide: a / b,
    power: a ** b,
    sqrt: Math.sqrt(a),
    percentage: (a * b) / 100,
  }
  return { ok: true, result: results[operation] }
}

beforeEach(() => {
  calculateMock.mockReset()
  calculateMock.mockImplementation((req) => Promise.resolve(fakeCalculate(req)))
})

async function clickAll(names: string[]) {
  for (const name of names) {
    await userEvent.click(screen.getByRole('button', { name }))
  }
}

const display = () => screen.getByRole('status')

describe('Calculator', () => {
  it('calculates left to right through the keypad', async () => {
    render(<Calculator />)

    await clickAll(['2', 'plus', '3', 'times', '4', 'equals'])

    await waitFor(() => expect(display()).toHaveTextContent('20'))
    const displayRegion = screen.getByRole('region', { name: 'Display' })
    expect(within(displayRegion).getByText('5 × 4 =')).toBeInTheDocument()
  })

  it('highlights the pending operator', async () => {
    render(<Calculator />)

    await clickAll(['2', 'minus'])

    expect(screen.getByRole('button', { name: 'minus' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('shows a backend error and clears it on the next key', async () => {
    render(<Calculator />)

    await clickAll(['1', '0', 'divided by', '0', 'equals'])
    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('division by zero is not allowed'),
    )

    await clickAll(['7'])
    expect(screen.getByRole('alert')).toBeEmptyDOMElement()
    expect(display()).toHaveTextContent('7')
  })

  it('records history newest first, keeping the last 10', async () => {
    render(<Calculator />)

    await userEvent.keyboard(`0${'+1'.repeat(12)}{Enter}`)

    const history = screen.getByRole('region', { name: 'History' })
    await waitFor(() => expect(within(history).getAllByRole('listitem')).toHaveLength(10))
    const items = within(history).getAllByRole('listitem')
    expect(items[0]).toHaveTextContent('11 + 1 = 12')
    expect(items[9]).toHaveTextContent('2 + 1 = 3')
  })

  it('accepts keyboard input', async () => {
    render(<Calculator />)

    await userEvent.keyboard('12.5*2{Enter}')

    await waitFor(() => expect(display()).toHaveTextContent('25'))
  })

  it('lets Enter activate the focused key instead of pressing equals', async () => {
    render(<Calculator />)
    await clickAll(['2', 'plus', '3'])
    screen.getByRole('button', { name: 'all clear' }).focus()

    await userEvent.keyboard('{Enter}')

    expect(display()).toHaveTextContent('0')
    expect(calculateMock).not.toHaveBeenCalled()
  })

  it('activates a keyboard-focused digit once on Enter', async () => {
    render(<Calculator />)
    await clickAll(['2', 'plus', '3'])
    screen.getByRole('button', { name: '3' }).focus()

    await userEvent.keyboard('{Enter}')

    expect(display()).toHaveTextContent('33')
    expect(calculateMock).not.toHaveBeenCalled()
  })

  it('does not focus keys on mouse click, so Enter afterwards means equals', async () => {
    render(<Calculator />)

    await clickAll(['2', 'plus', '3'])
    expect(screen.getByRole('button', { name: '3' })).not.toHaveFocus()
    await userEvent.keyboard('{Enter}')

    await waitFor(() => expect(display()).toHaveTextContent('5'))
  })

  it('prevents the default action of handled keys, except Enter on a button', () => {
    render(<Calculator />)
    const button = screen.getByRole('button', { name: '7' })

    // fireEvent returns false when the default action was prevented.
    expect(fireEvent.keyDown(document.body, { key: '/' })).toBe(false)
    expect(fireEvent.keyDown(document.body, { key: 'Enter' })).toBe(false)
    expect(fireEvent.keyDown(button, { key: 'Enter' })).toBe(true)
    expect(fireEvent.keyDown(document.body, { key: 'a' })).toBe(true)
  })

  it('leaves browser shortcuts alone', async () => {
    render(<Calculator />)

    await userEvent.keyboard('{Control>}r{/Control}')

    expect(calculateMock).not.toHaveBeenCalled()
  })

  it('stops listening to the keyboard after unmount', async () => {
    const { unmount } = render(<Calculator />)
    unmount()

    await userEvent.keyboard('9r')

    expect(calculateMock).not.toHaveBeenCalled()
  })
})
