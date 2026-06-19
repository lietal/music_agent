import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useAuth } from './useAuth'
import { renderHook, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <MemoryRouter initialEntries={['/chat']}>{children}</MemoryRouter>
)

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
})

describe('useAuth', () => {
  it('login stores token and returns user', async () => {
    const mockUser = { user_id: 'u1', display_name: 'Test' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ token: 'jwt-token', user: mockUser }) }))

    const { result } = renderHook(() => useAuth(), { wrapper })
    let user: unknown
    await act(async () => { user = await result.current.login('user', 'pass') })

    expect(localStorage.getItem('jwt')).toBe('jwt-token')
    expect(user).toEqual(mockUser)
  })

  it('register stores token and returns user', async () => {
    const mockUser = { user_id: 'u2', display_name: 'New' }
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({ token: 'reg-token', user: mockUser }) }))

    const { result } = renderHook(() => useAuth(), { wrapper })
    let user: unknown
    await act(async () => { user = await result.current.register('user', 'pass', 'Display') })

    expect(localStorage.getItem('jwt')).toBe('reg-token')
    expect(user).toEqual(mockUser)
  })

  it('logout clears token', () => {
    localStorage.setItem('jwt', 'some-token')
    const { result } = renderHook(() => useAuth(), { wrapper })
    act(() => { result.current.logout() })
    expect(localStorage.getItem('jwt')).toBeNull()
  })

  it('isAuthenticated returns false without token', () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isAuthenticated()).toBe(false)
  })

  it('isAuthenticated returns false for expired token', () => {
    const header = btoa(JSON.stringify({ alg: 'HS256' }))
    const payload = btoa(JSON.stringify({ exp: 1000 }))
    localStorage.setItem('jwt', `${header}.${payload}.sig`)
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isAuthenticated()).toBe(false)
  })

  it('isAuthenticated returns true for valid token', () => {
    const header = btoa(JSON.stringify({ alg: 'HS256' }))
    const payload = btoa(JSON.stringify({ exp: Math.floor(Date.now() / 1000) + 3600, user_id: 'u1' }))
    localStorage.setItem('jwt', `${header}.${payload}.sig`)
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isAuthenticated()).toBe(true)
  })

  it('isAuthenticated returns false for invalid format', () => {
    localStorage.setItem('jwt', 'not-a-jwt')
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isAuthenticated()).toBe(false)
  })

  it('isAuthenticated returns false for malformed token', () => {
    localStorage.setItem('jwt', 'header.payload')
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isAuthenticated()).toBe(false)
  })

  it('getToken/setToken round trip', () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    act(() => { result.current.setToken('test-token') })
    expect(result.current.getToken()).toBe('test-token')
  })
})
