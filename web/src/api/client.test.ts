import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getToken, setToken, clearToken, api } from './client'

beforeEach(() => { localStorage.clear(); vi.restoreAllMocks() })

describe('client', () => {
  it('getToken returns null initially', () => { expect(getToken()).toBeNull() })
  it('setToken/getToken round trip', () => { setToken('j'); expect(getToken()).toBe('j') })
  it('clearToken removes', () => { setToken('j'); clearToken(); expect(getToken()).toBeNull() })

  it('auth.login', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ token: 't', user: { user_id: 'u', display_name: 'D' } }) }))
    expect((await api.auth.login('u','p')).token).toBe('t')
  })
  it('auth.register', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ token: 't', user: { user_id: 'u', display_name: 'D' } }) }))
    expect((await api.auth.register('u','p')).token).toBe('t')
  })
  it('auth.me', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ id: 'u1' }) }))
    expect((await api.auth.me()).id).toBe('u1')
  })
  it('chat.send', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ runId: 'r1' }) }))
    expect((await api.chat.send('hi')).runId).toBe('r1')
  })
  it('chat.eventsUrl', () => { setToken('jwt'); expect(api.chat.eventsUrl('r1')).toContain('token=jwt') })
  it('conversations.list', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve([]) }))
    expect(await api.conversations.list()).toEqual([])
  })
  it('conversations.get', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ id: 'c1' }) }))
    expect((await api.conversations.get('c1')).id).toBe('c1')
  })
  it('conversations.create', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ id: 'c2' }) }))
    expect((await api.conversations.create()).id).toBe('c2')
  })
  it('error on non-ok', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 401 }))
    await expect(api.auth.me()).rejects.toThrow('HTTP 401')
  })
})
