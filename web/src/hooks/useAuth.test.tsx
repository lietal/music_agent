import { describe, it, expect, beforeEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { getToken, setToken, clearToken } from '../api/client'
import { useAuth } from './useAuth'

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <MemoryRouter initialEntries={['/chat']}>{children}</MemoryRouter>
)

describe('useAuth', () => {
  beforeEach(() => { localStorage.clear() })

  it('isAuthenticated returns false without token', () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isAuthenticated()).toBe(false)
  })

  it('getToken returns null initially', () => {
    renderHook(() => useAuth(), { wrapper })
    expect(getToken()).toBeNull()
  })

  it('setToken + getToken round trip', () => {
    setToken('my-token')
    expect(getToken()).toBe('my-token')
  })

  it('clearToken removes token', () => {
    setToken('x'); clearToken()
    expect(getToken()).toBeNull()
  })
})
