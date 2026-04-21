import { afterEach, describe, expect, it, vi } from 'vitest'

import { ApiError, request, setAuthHandlers } from './client'
import { API_ENDPOINTS } from './endpoints'

describe('request', () => {
  afterEach(() => {
    setAuthHandlers(null)
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('triggers refresh on 401 and logs out when refresh fails', async () => {
    const refresh = vi.fn().mockResolvedValue(false)
    const logout = vi.fn()
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 401 }))

    setAuthHandlers({
      getAccessToken: () => 'dummy-token',
      refresh,
      logout,
    })
    vi.stubGlobal('fetch', fetchMock)

    await expect(request('GET', API_ENDPOINTS.me)).rejects.toMatchObject({
      name: 'ApiError',
      code: 'unauthorized',
    })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(logout).toHaveBeenCalledTimes(1)
  })

  it('throws ApiError when envelope.success is false', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          success: false,
          data: null,
          error: { code: 'bad_request', message: 'invalid payload' },
        }),
        {
          status: 400,
          headers: { 'Content-Type': 'application/json' },
        },
      ),
    )

    setAuthHandlers({
      getAccessToken: () => 'dummy-token',
      refresh: vi.fn().mockResolvedValue(false),
      logout: vi.fn(),
    })
    vi.stubGlobal('fetch', fetchMock)

    try {
      await request('POST', API_ENDPOINTS.authLogin, { email: 'a@b.c' })
      throw new Error('expected ApiError')
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError)
      expect(error).toMatchObject({
        code: 'bad_request',
        message: 'invalid payload',
      })
    }
  })
})
