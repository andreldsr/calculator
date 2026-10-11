import type {
  CalculateFailure,
  CalculateOutcome,
  CalculateRequest,
  CalculateSuccess,
} from '../types/calculator'

const CALCULATE_URL = '/api/v1/calculate'

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isSuccess(body: unknown): body is CalculateSuccess {
  return isRecord(body) && typeof body.result === 'number'
}

function isFailure(body: unknown): body is CalculateFailure {
  return isRecord(body) && typeof body.error === 'string'
}

// Checks the name rather than instanceof DOMException, which fails for aborts
// from polyfills or other realms.
function isAbort(err: unknown): boolean {
  return isRecord(err) && err.name === 'AbortError'
}

// calculate sends one operation to the backend. HTTP and network failures
// come back as { ok: false }; only an abort is thrown, so callers can ignore it.
export async function calculate(
  request: CalculateRequest,
  signal?: AbortSignal,
): Promise<CalculateOutcome> {
  let response: Response
  try {
    response = await fetch(CALCULATE_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
      ...(signal ? { signal } : {}),
    })
  } catch (err) {
    if (isAbort(err)) throw err
    return { ok: false, error: 'Unable to reach the calculator service' }
  }

  let body: unknown = null
  try {
    body = await response.json()
  } catch (err) {
    if (isAbort(err)) throw err
    // Not JSON, e.g. a proxy error page; handled below.
  }

  if (response.ok && isSuccess(body)) return { ok: true, result: body.result }
  if (!response.ok && isFailure(body)) return { ok: false, error: body.error }
  return { ok: false, error: `Unexpected response from server (status ${response.status})` }
}
