import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import App from './App'

describe('App', () => {
  it('renders the app heading', () => {
    render(<App />)
    expect(screen.getByRole('heading', { name: /sezzle calculator/i })).toBeInTheDocument()
  })

  it('renders the calculator', () => {
    render(<App />)
    expect(screen.getByRole('group', { name: 'Keypad' })).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('0')
  })
})
