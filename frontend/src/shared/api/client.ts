// Small fetch wrapper shared by all features. Requests go to /api, which the
// Vite dev server and nginx proxy to the backend (docs/ARCHITECTURE.md).

const TOKEN_KEY = 'token'

/** Error body returned by the backend for every failed request. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export function getToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

/**
 * Sends a JSON request with the Bearer token and returns the parsed body.
 * Throws ApiError with the backend's code for non-2xx responses, and an
 * ApiError with code "network_error" when the backend cannot be reached.
 */
export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body !== undefined) headers.set('Content-Type', 'application/json')
  const token = getToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)

  let res: Response
  try {
    res = await fetch(`/api${path}`, { ...init, headers })
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError(0, 'network_error', 'Cannot reach the server. Check your connection.')
  }

  const body: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    const e = (body ?? {}) as { code?: unknown; message?: unknown }
    throw new ApiError(
      res.status,
      typeof e.code === 'string' ? e.code : 'unknown_error',
      typeof e.message === 'string' ? e.message : `Request failed (${res.status})`,
    )
  }
  return body as T
}
