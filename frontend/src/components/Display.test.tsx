import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Display } from './Display'

describe('Display', () => {
  it('shows the value in a polite, atomic live region', () => {
    render(<Display value="12.5" expression="" error={null} isLoading={false} />)

    const output = screen.getByRole('status')
    expect(output).toHaveTextContent('12.5')
    expect(output).toHaveAttribute('aria-live', 'polite')
    expect(output).toHaveAttribute('aria-atomic', 'true')
  })

  it('shows the expression line', () => {
    render(<Display value="3" expression="2 +" error={null} isLoading={false} />)

    expect(screen.getByText('2 +')).toBeInTheDocument()
  })

  it('announces an error as an alert', () => {
    render(
      <Display value="0" expression="" error="division by zero is not allowed" isLoading={false} />,
    )

    expect(screen.getByRole('alert')).toHaveTextContent('division by zero is not allowed')
  })

  it('keeps an empty alert region when there is no error', () => {
    render(<Display value="0" expression="" error={null} isLoading={false} />)

    expect(screen.getByRole('alert')).toBeEmptyDOMElement()
  })

  it.each([true, false])('sets aria-busy to %s', (isLoading) => {
    render(<Display value="0" expression="" error={null} isLoading={isLoading} />)

    expect(screen.getByRole('region', { name: 'Display' })).toHaveAttribute(
      'aria-busy',
      String(isLoading),
    )
  })
})
