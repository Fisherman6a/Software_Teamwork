import { describe, expect, it, vi } from 'vitest'

import { apiClient, ApiError, gatewayRequest } from './client'

function setGatewayBaseUrl() {
  vi.stubEnv('VITE_API_BASE_URL', 'http://gateway.test/api/v1')
}

describe('gatewayRequest', () => {
  it('maps gateway error envelopes to ApiError details', async () => {
    setGatewayBaseUrl()
    apiClient.setToken('token-123')
    const fetchMock = vi.fn(
      async (_request: Request) =>
        new Response(
          JSON.stringify({
            error: {
              code: 'validation_error',
              message: 'name is required',
              requestId: 'req-123',
              fields: { name: 'is required' },
            },
          }),
          {
            headers: { 'Content-Type': 'application/json' },
            status: 400,
          },
        ),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(
      gatewayRequest('/reports', { method: 'POST', body: { name: '' } }),
    ).rejects.toEqual(
      expect.objectContaining({
        code: 'validation_error',
        fields: { name: 'is required' },
        message: 'name is required',
        requestId: 'req-123',
        status: 400,
      }),
    )

    const request = fetchMock.mock.calls[0]?.[0]
    expect(request).toBeInstanceOf(Request)
    if (!(request instanceof Request)) throw new Error('expected fetch to receive a Request')
    expect(request.headers.get('Authorization')).toBe('Bearer token-123')
    expect(request.headers.get('Content-Type')).toBe('application/json')
  })

  it('clears the stored token when gateway returns unauthorized', async () => {
    setGatewayBaseUrl()
    apiClient.setToken('expired-token')
    const fetchMock = vi.fn(
      async (_request: Request) =>
        new Response(
          JSON.stringify({
            error: {
              code: 'unauthorized',
              message: 'session expired',
              requestId: 'req-auth',
            },
          }),
          {
            headers: { 'Content-Type': 'application/json' },
            status: 401,
          },
        ),
    )
    vi.stubGlobal('fetch', fetchMock)

    await expect(gatewayRequest('/users/me')).rejects.toBeInstanceOf(ApiError)
    expect(apiClient.getToken()).toBeNull()
    expect(localStorage.getItem('auth_token')).toBeNull()
  })
})
