import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Calculator } from './Calculator'

// Unlike Calculator.test.tsx, the real services/api runs here; only fetch is
// stubbed, so the hook → service → HTTP seam is exercised end to end.

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

async function clickAll(names: string[]) {
  for (const name of names) {
    await userEvent.click(screen.getByRole('button', { name }))
  }
}

describe('Calculator with the real API service', () => {
  it('posts the operation and shows the result', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { result: 5 }))
    vi.stubGlobal('fetch', fetchMock)
    render(<Calculator />)

    await clickAll(['1', '0', 'divided by', '2', 'equals'])

    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('5'))
    expect(fetchMock).toHaveBeenCalledExactlyOnceWith(
      '/api/v1/calculate',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ operation: 'divide', a: 10, b: 2 }),
      }),
    )
    const history = screen.getByRole('region', { name: 'History' })
    expect(within(history).getByRole('listitem')).toHaveTextContent('10 ÷ 2 = 5')
  })

  it('shows the error envelope from a 400', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse(400, { error: 'division by zero is not allowed' })),
    )
    render(<Calculator />)

    await clickAll(['1', '0', 'divided by', '0', 'equals'])

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('division by zero is not allowed'),
    )
    expect(screen.getByText('No calculations yet')).toBeInTheDocument()
  })

  it('shows a friendly error when the proxy returns a non-JSON page', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('<html>Bad Gateway</html>', { status: 502 })),
    )
    render(<Calculator />)

    await clickAll(['9', 'square root'])

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Unexpected response from server (status 502)',
      ),
    )
  })
})
