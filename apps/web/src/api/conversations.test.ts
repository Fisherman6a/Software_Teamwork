import { describe, expect, it, vi } from 'vitest'

import { deleteSession } from './conversations'

describe('conversations gateway API', () => {
  it('treats a 204 delete session response as success', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'http://gateway.test/api/v1')
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValue(new Response(null, { status: 204, statusText: 'No Content' }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(deleteSession('session-1')).resolves.toBeUndefined()

    const request = fetchMock.mock.calls[0]?.[0]
    expect(request).toBeInstanceOf(Request)
    if (!(request instanceof Request)) throw new Error('expected fetch to receive a Request')
    expect(request.method).toBe('DELETE')
    expect(request.url).toBe('http://gateway.test/api/v1/qa-sessions/session-1')
  })
})
