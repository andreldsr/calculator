import { afterEach, describe, expect, it, vi } from 'vitest'
import { calculate } from './api'
import type { CalculateOutcome } from '../types/calculator'

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('calculate', () => {
  it('posts the request as JSON to /api/v1/calculate', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { result: 5 }))
    vi.stubGlobal('fetch', fetchMock)

    await calculate({ operation: 'divide', a: 10, b: 2 })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0] ?? []
    expect(url).toBe('/api/v1/calculate')
    expect(init).toMatchObject({
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ operation: 'divide', a: 10, b: 2 }),
    })
  })

  it('passes the abort signal to fetch', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { result: 1 }))
    vi.stubGlobal('fetch', fetchMock)
    const controller = new AbortController()

    await calculate({ operation: 'sqrt', a: 1 }, controller.signal)

    const [, init] = fetchMock.mock.calls[0] ?? []
    expect(init).toMatchObject({ signal: controller.signal })
  })

  const cases: { name: string; response: () => Response; want: CalculateOutcome }[] = [
    {
      name: '200 with a result',
      response: () => jsonResponse(200, { result: 5 }),
      want: { ok: true, result: 5 },
    },
    {
      name: '400 with an error envelope',
      response: () => jsonResponse(400, { error: 'division by zero is not allowed' }),
      want: { ok: false, error: 'division by zero is not allowed' },
    },
    {
      name: '405 with an error envelope',
      response: () => jsonResponse(405, { error: 'method not allowed' }),
      want: { ok: false, error: 'method not allowed' },
    },
    {
      name: '500 with an error envelope',
      response: () => jsonResponse(500, { error: 'internal server error' }),
      want: { ok: false, error: 'internal server error' },
    },
    {
      name: '502 with a non-JSON body',
      response: () => new Response('<html>Bad Gateway</html>', { status: 502 }),
      want: { ok: false, error: 'Unexpected response from server (status 502)' },
    },
    {
      name: '200 with a body that is not a result',
      response: () => jsonResponse(200, { value: 5 }),
      want: { ok: false, error: 'Unexpected response from server (status 200)' },
    },
    {
      name: '200 with a non-numeric result',
      response: () => jsonResponse(200, { result: '5' }),
      want: { ok: false, error: 'Unexpected response from server (status 200)' },
    },
    {
      name: '400 with a non-string error',
      response: () => jsonResponse(400, { error: 42 }),
      want: { ok: false, error: 'Unexpected response from server (status 400)' },
    },
    {
      name: '400 with a result instead of an error',
      response: () => jsonResponse(400, { result: 5 }),
      want: { ok: false, error: 'Unexpected response from server (status 400)' },
    },
    {
      name: '200 with null',
      response: () => jsonResponse(200, null),
      want: { ok: false, error: 'Unexpected response from server (status 200)' },
    },
  ]

  it.each(cases)('maps $name', async ({ response, want }) => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response()))

    await expect(calculate({ operation: 'add', a: 1, b: 2 })).resolves.toEqual(want)
  })

  it('maps a network failure to a friendly error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))

    await expect(calculate({ operation: 'add', a: 1, b: 2 })).resolves.toEqual({
      ok: false,
      error: 'Unable to reach the calculator service',
    })
  })

  it('rethrows an abort so the caller can ignore it', async () => {
    const abort = new DOMException('The operation was aborted.', 'AbortError')
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(abort))

    await expect(calculate({ operation: 'add', a: 1, b: 2 })).rejects.toBe(abort)
  })

  it('recognizes an abort that is not a DOMException', async () => {
    // e.g. from a fetch polyfill or another realm.
    const abort = Object.assign(new Error('aborted'), { name: 'AbortError' })
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(abort))

    await expect(calculate({ operation: 'add', a: 1, b: 2 })).rejects.toBe(abort)
  })
})
