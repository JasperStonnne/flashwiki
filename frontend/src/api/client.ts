import { API_ENDPOINTS } from './endpoints'

export type Envelope<T> =
  | { success: true; data: T; error: null }
  | { success: false; data: null; error: { code: string; message: string } }

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

type AuthHandlers = {
  getAccessToken: () => string | null
  refresh: () => Promise<boolean>
  logout: () => void
}

const ACCESS_TOKEN_KEY = 'access_token'

let authHandlers: AuthHandlers | null = null
let refreshInFlight: Promise<boolean> | null = null

export class ApiError extends Error {
  code: string

  constructor(code: string, message: string) {
    super(message)
    this.code = code
    this.name = 'ApiError'
  }
}

export function setAuthHandlers(handlers: AuthHandlers | null) {
  authHandlers = handlers
}

function getAccessToken(): string | null {
  if (authHandlers) {
    return authHandlers.getAccessToken()
  }
  return localStorage.getItem(ACCESS_TOKEN_KEY)
}

async function refreshOnce(): Promise<boolean> {
  if (!authHandlers) {
    return false
  }

  if (!refreshInFlight) {
    refreshInFlight = authHandlers
      .refresh()
      .catch(() => false)
      .finally(() => {
        refreshInFlight = null
      })
  }

  return refreshInFlight
}

function triggerLogout() {
  if (authHandlers) {
    authHandlers.logout()
    return
  }

  localStorage.removeItem(ACCESS_TOKEN_KEY)
  window.location.href = '/login'
}

function buildHeaders(body: unknown): Headers {
  const headers = new Headers()
  const token = getAccessToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  if (body !== undefined) {
    headers.set('Content-Type', 'application/json')
  }
  return headers
}

async function parseEnvelope<T>(response: Response): Promise<Envelope<T>> {
  let json: unknown
  try {
    json = await response.json()
  } catch {
    throw new ApiError('invalid_response', 'invalid response envelope')
  }
  return json as Envelope<T>
}

export async function request<T>(
  method: HttpMethod,
  path: string,
  body?: unknown,
  retried = false,
): Promise<T> {
  const response = await fetch(path, {
    method,
    headers: buildHeaders(body),
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (response.status === 401 && path !== API_ENDPOINTS.authRefresh && !retried) {
    const refreshed = await refreshOnce()
    if (refreshed) {
      return request<T>(method, path, body, true)
    }

    triggerLogout()
    throw new ApiError('unauthorized', 'unauthorized')
  }

  const envelope = await parseEnvelope<T>(response)
  if (!envelope.success) {
    throw new ApiError(envelope.error.code, envelope.error.message)
  }
  return envelope.data
}
