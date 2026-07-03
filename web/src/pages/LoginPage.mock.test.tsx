import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

import LoginPage from './LoginPage'

describe('LoginPage - mock login flow', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    ;(window as any).__MOCK_LOGIN__ = true
    const location = { href: '' }
    Object.defineProperty(window, 'location', {
      value: location,
      writable: true,
      configurable: true,
    })
  })

  afterEach(() => {
    delete (window as any).__MOCK_LOGIN__
  })

  function renderLoginPage() {
    return render(
      <MemoryRouter initialEntries={['/login']}>
        <LoginPage />
      </MemoryRouter>
    )
  }

  it('mock login shows QR and scanning text', () => {
    vi.useFakeTimers()
    renderLoginPage()

    fireEvent.click(screen.getByRole('button', { name: '登录 QQ 音乐' }))

    expect(screen.getByText('请用 QQ App 扫描二维码')).toBeInTheDocument()
    expect(screen.getByAltText('QQ 登录二维码')).toBeInTheDocument()

    vi.useRealTimers()
  })

  it('mock login sets JWT and redirects to /chat', async () => {
    vi.useFakeTimers()
    renderLoginPage()

    fireEvent.click(screen.getByRole('button', { name: '登录 QQ 音乐' }))
    expect(screen.getByAltText('QQ 登录二维码')).toBeInTheDocument()

    await act(() => vi.advanceTimersByTimeAsync(2100))

    expect(localStorage.getItem('jwt')).toBe('mock-jwt-token')
    expect(window.location.href).toBe('/chat')

    vi.useRealTimers()
  })
})
