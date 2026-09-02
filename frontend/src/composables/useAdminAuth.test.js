import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

// Mock element-plus to silence ElMessage calls and let us assert on them.
const mockElMessage = {
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  info: vi.fn(),
}
vi.mock('element-plus', () => ({
  ElMessage: mockElMessage,
}))

// Import after mock so the composable sees the mocked module.
const { useAdminAuth } = await import('./useAdminAuth')

// Stub fetch for every test.
const mockFetch = vi.fn()

function jsonResponse(data, { ok = true, status = 200 } = {}) {
  return {
    ok,
    status,
    json: async () => data,
  }
}

function makeSut(options) {
  const result = useAdminAuth(options)
  // Convenience: wrap the returned refs into a plain object for ergonomic assertions.
  return result
}

describe('useAdminAuth', () => {
  beforeEach(() => {
    mockFetch.mockReset()
    mockElMessage.success.mockReset()
    mockElMessage.error.mockReset()
    mockElMessage.warning.mockReset()
    mockElMessage.info.mockReset()
    global.fetch = mockFetch
    localStorage.clear()
    sessionStorage.clear()
  })

  afterEach(() => {
    delete global.fetch
  })

  describe('credentialMode: body (default)', () => {
    it('POSTs JSON body with the credentialField and saves on success', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))
      const sut = makeSut({
        storageKey: 'k_body',
        verifyEndpoint: '/api/x/verify',
      })

      sut.passwordInput.value = 'secret123'
      await sut.login()

      expect(mockFetch).toHaveBeenCalledTimes(1)
      const [url, opts] = mockFetch.mock.calls[0]
      expect(url).toBe('/api/x/verify')
      expect(opts.method).toBe('POST')
      expect(opts.headers['Content-Type']).toBe('application/json')
      expect(JSON.parse(opts.body)).toEqual({ password: 'secret123' })
      expect(localStorage.getItem('k_body')).toBe('secret123')
      expect(sut.isAuthenticated.value).toBe(true)
      expect(mockElMessage.success).toHaveBeenCalledWith('登录成功')
      expect(sut.loggingIn.value).toBe(false)
    })

    it('removes stored key and shows error on failed verify', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'bad password' }, { ok: false, status: 401 }))
      const sut = makeSut({
        storageKey: 'k_body_fail',
        verifyEndpoint: '/api/x/verify',
      })

      sut.passwordInput.value = 'wrong'
      await sut.login()

      expect(localStorage.getItem('k_body_fail')).toBeNull()
      expect(sut.isAuthenticated.value).toBe(false)
      expect(mockElMessage.error).toHaveBeenCalledWith('密码错误或已失效')
    })

    it('warns and skips fetch when passwordInput is empty', async () => {
      const sut = makeSut({
        storageKey: 'k_body_empty',
        verifyEndpoint: '/api/x/verify',
      })
      await sut.login()
      expect(mockFetch).not.toHaveBeenCalled()
      expect(mockElMessage.warning).toHaveBeenCalledWith('请输入管理员密码')
      expect(sut.isAuthenticated.value).toBe(false)
    })

    it('honors a custom credentialField when sending body', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))
      const sut = makeSut({
        storageKey: 'k_body_custom',
        verifyEndpoint: '/api/x/verify',
        credentialField: 'admin_password',
      })
      sut.passwordInput.value = 'pw'
      await sut.login()
      expect(JSON.parse(mockFetch.mock.calls[0][1].body)).toEqual({ admin_password: 'pw' })
      expect(sut.authBody()).toEqual({ admin_password: 'pw' })
    })

    it('uses custom verifyOk predicate for non-ok-shaped responses', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ status: 'ok', nodes: [1, 2] }))
      const sut = makeSut({
        storageKey: 'k_body_verify_ok',
        verifyEndpoint: '/api/proxy/status',
        verifyOk: (d) => d.status === 'ok',
      })
      sut.passwordInput.value = 'pw'
      await sut.login()
      expect(sut.isAuthenticated.value).toBe(true)
      expect(localStorage.getItem('k_body_verify_ok')).toBe('pw')
    })
  })

  describe('credentialMode: query', () => {
    it('PUTs the credential into URL search params (URL-encoded) and saves on success', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))
      const sut = makeSut({
        storageKey: 'k_query',
        verifyEndpoint: '/api/proxy/status',
        credentialMode: 'query',
        credentialField: 'admin_password',
      })

      sut.passwordInput.value = 'pw with space & special=chars'
      await sut.login()

      const [calledUrl, calledOpts] = mockFetch.mock.calls[0]
      expect(calledOpts.method).toBe('POST')
      // URL object normalizes to absolute URL
      expect(calledUrl.startsWith('http')).toBe(true)
      const parsed = new URL(calledUrl)
      expect(parsed.pathname).toBe('/api/proxy/status')
      expect(parsed.searchParams.get('admin_password')).toBe('pw with space & special=chars')
      // No body leaks when in query mode
      expect(calledOpts.body).toBeUndefined()
      expect(localStorage.getItem('k_query')).toBe('pw with space & special=chars')
      expect(sut.isAuthenticated.value).toBe(true)
    })

    it('returns false and clears storage on a query-mode 401', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'wrong' }, { ok: false, status: 401 }))
      const sut = makeSut({
        storageKey: 'k_query_fail',
        verifyEndpoint: '/api/nps/status',
        credentialMode: 'query',
        credentialField: 'admin_password',
      })
      sut.passwordInput.value = 'wrong'
      await sut.login()
      expect(localStorage.getItem('k_query_fail')).toBeNull()
      expect(sut.isAuthenticated.value).toBe(false)
    })

    it('authQuery() returns the credentialField keyed by current stored value', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))
      const sut = makeSut({
        storageKey: 'k_query_authq',
        verifyEndpoint: '/api/proxy/status',
        credentialMode: 'query',
        credentialField: 'admin_password',
      })
      sut.passwordInput.value = 'mypass'
      await sut.login()
      expect(sut.authQuery()).toEqual({ admin_password: 'mypass' })
    })
  })

  describe('credentialMode: header', () => {
    it('sends the password in a custom header and saves on success', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))
      const sut = makeSut({
        storageKey: 'k_header',
        verifyEndpoint: '/api/monitor/verify',
        credentialMode: 'header',
        headerName: 'X-Super-Admin-Password',
      })

      sut.passwordInput.value = 'topsecret'
      await sut.login()

      const [url, opts] = mockFetch.mock.calls[0]
      expect(url).toBe('/api/monitor/verify')
      expect(opts.method).toBe('POST')
      expect(opts.headers).toEqual({ 'X-Super-Admin-Password': 'topsecret' })
      // No body should be set in header mode
      expect(opts.body).toBeUndefined()
      expect(localStorage.getItem('k_header')).toBe('topsecret')
      expect(sut.isAuthenticated.value).toBe(true)
    })

    it('authHeader() returns the header name + current stored value', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))
      const sut = makeSut({
        storageKey: 'k_header_helper',
        verifyEndpoint: '/api/monitor/verify',
        credentialMode: 'header',
      })
      sut.passwordInput.value = 'pw'
      await sut.login()
      expect(sut.authHeader()).toEqual({ 'X-Super-Admin-Password': 'pw' })
    })

    it('treats res.ok=false as failure', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({}, { ok: false, status: 401 }))
      const sut = makeSut({
        storageKey: 'k_header_fail',
        verifyEndpoint: '/api/monitor/verify',
        credentialMode: 'header',
      })
      sut.passwordInput.value = 'bad'
      await sut.login()
      expect(sut.isAuthenticated.value).toBe(false)
      expect(localStorage.getItem('k_header_fail')).toBeNull()
      expect(mockElMessage.error).toHaveBeenCalledWith('密码错误或已失效')
    })
  })

  describe('tryStored + trustStored', () => {
    it('trustStored=true: stored creds make isAuthenticated=true without calling fetch', async () => {
      localStorage.setItem('k_trust', 'stored_pw')
      const sut = makeSut({
        storageKey: 'k_trust',
        verifyEndpoint: '/api/x/verify',
        trustStored: true,
      })
      const ok = await sut.tryStored()
      expect(ok).toBe(true)
      expect(sut.isAuthenticated.value).toBe(true)
      expect(mockFetch).not.toHaveBeenCalled()
    })

    it('trustStored=false: stored creds trigger a silent verify and stay in if it passes', async () => {
      localStorage.setItem('k_verify_stored', 'pw')
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))
      const sut = makeSut({
        storageKey: 'k_verify_stored',
        verifyEndpoint: '/api/x/verify',
        trustStored: false,
      })
      const ok = await sut.tryStored()
      expect(ok).toBe(true)
      expect(mockFetch).toHaveBeenCalledTimes(1)
      expect(sut.isAuthenticated.value).toBe(true)
      expect(mockElMessage.success).not.toHaveBeenCalled() // silent
    })

    it('trustStored=false: stored creds are removed on failed verify', async () => {
      localStorage.setItem('k_verify_stored_fail', 'old')
      mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'expired' }, { ok: false, status: 401 }))
      const sut = makeSut({
        storageKey: 'k_verify_stored_fail',
        verifyEndpoint: '/api/x/verify',
        trustStored: false,
      })
      const ok = await sut.tryStored()
      expect(ok).toBe(false)
      expect(localStorage.getItem('k_verify_stored_fail')).toBeNull()
      expect(sut.isAuthenticated.value).toBe(false)
      expect(mockElMessage.error).not.toHaveBeenCalled() // silent
    })

    it('tryStored returns false when nothing is stored', async () => {
      const sut = makeSut({
        storageKey: 'k_empty',
        verifyEndpoint: '/api/x/verify',
        trustStored: true,
      })
      const ok = await sut.tryStored()
      expect(ok).toBe(false)
      expect(sut.isAuthenticated.value).toBe(false)
      expect(mockFetch).not.toHaveBeenCalled()
    })
  })

  describe('storage selection', () => {
    it('localStorage is used by default', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))
      const sut = makeSut({
        storageKey: 'k_default_local',
        verifyEndpoint: '/api/x/verify',
      })
      sut.passwordInput.value = 'pw'
      await sut.login()
      expect(localStorage.getItem('k_default_local')).toBe('pw')
      expect(sessionStorage.getItem('k_default_local')).toBeNull()
    })

    it('sessionStorage is used when storage=session', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))
      const sut = makeSut({
        storageKey: 'k_session',
        verifyEndpoint: '/api/x/verify',
        storage: 'session',
      })
      sut.passwordInput.value = 'pw'
      await sut.login()
      expect(sessionStorage.getItem('k_session')).toBe('pw')
      expect(localStorage.getItem('k_session')).toBeNull()
    })

    it('logout clears the matching storage and resets state', async () => {
      mockFetch.mockResolvedValueOnce(jsonResponse({ ok: true }))
      const sut = makeSut({
        storageKey: 'k_logout',
        verifyEndpoint: '/api/x/verify',
      })
      sut.passwordInput.value = 'pw'
      await sut.login()
      expect(sut.isAuthenticated.value).toBe(true)
      expect(localStorage.getItem('k_logout')).toBe('pw')

      sut.logout()
      expect(localStorage.getItem('k_logout')).toBeNull()
      expect(sut.isAuthenticated.value).toBe(false)
      expect(sut.passwordInput.value).toBe('')
      expect(mockElMessage.info).toHaveBeenCalledWith('已退出登录')
    })
  })
})
