import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { History } from './History'
import type { HistoryEntry } from '../types/calculator'

describe('History', () => {
  it('lives in a section labeled by its heading', () => {
    render(<History entries={[]} />)

    expect(screen.getByRole('region', { name: 'History' })).toBeInTheDocument()
  })

  it('says when there are no calculations', () => {
    render(<History entries={[]} />)

    expect(screen.getByText('No calculations yet')).toBeInTheDocument()
    expect(screen.queryByRole('list')).not.toBeInTheDocument()
  })

  it('lists entries in the order given, formatted', () => {
    const entries: HistoryEntry[] = [
      { id: 4, operation: 'percentage', a: 10, b: 200, result: 20 },
      { id: 3, operation: 'sqrt', a: 16, result: 4 },
      { id: 2, operation: 'divide', a: 1, b: 3, result: 1 / 3 },
      { id: 1, operation: 'add', a: 12, b: 3, result: 15 },
    ]
    render(<History entries={entries} />)

    const items = screen.getAllByRole('listitem').map((item) => item.textContent)
    expect(items).toEqual([
      '10% of 200 = 20',
      '√16 = 4',
      '1 ÷ 3 = 0.333333333333333',
      '12 + 3 = 15',
    ])
  })
})
