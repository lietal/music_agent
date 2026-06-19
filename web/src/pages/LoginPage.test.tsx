import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

vi.mock('../api/client', () => ({
  api: {
    auth: {
      login: vi.fn(),
      register: vi.fn(),
    },
  },
  setToken: vi.fn(),
}))

vi.mock('../hooks/useQQMusicLogin', () => ({
  useQQMusicLogin: vi.fn(),
}))

import { api } from '../api/client'
import { useQQMusicLogin } from '../hooks/useQQMusicLogin'
import LoginPage from './LoginPage'

function mockQQLogin(overrides: Record<string, unknown> = {}) {
  vi.mocked(useQQMusicLogin).mockReturnValue({
    loginStatus: 'idle',
    qrcodeUrl: null,
    userName: null,
    isLoggedIn: false,
    startLogin: vi.fn(),
    checkStatus: vi.fn(),
    logout: vi.fn(),
    ...overrides,
  } as ReturnType<typeof useQQMusicLogin>)
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockQQLogin()
  })

  function renderLoginPage() {
    return render(
      <MemoryRouter initialEntries={['/login']}>
        <LoginPage />
      </MemoryRouter>
    )
  }

  it('renders login form with username/password fields', () => {
    renderLoginPage()

    expect(screen.getByPlaceholderText('用户名')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('密码')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument()
  })

  it('toggle between login and register mode', () => {
    renderLoginPage()

    expect(screen.getByText(/没有账号/)).toBeInTheDocument()

    fireEvent.click(screen.getByText('注册'))

    expect(screen.getByText(/已有账号/)).toBeInTheDocument()
    expect(screen.getByPlaceholderText('显示名称（可选）')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '注册' })).toBeInTheDocument()

    fireEvent.click(screen.getByText('登录'))

    expect(screen.getByText(/没有账号/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument()
  })

  it('shows error on empty submit', async () => {
    vi.mocked(api.auth.login).mockRejectedValue(new Error('Login failed'))
    renderLoginPage()

    const form = document.querySelector('form')!
    fireEvent.submit(form)

    expect(api.auth.login).toHaveBeenCalledWith('', '')

    await waitFor(() => {
      expect(screen.getByText('用户名或密码错误')).toBeInTheDocument()
    })
  })

  describe('QQ Music login', () => {
    it('renders QQ 音乐登录 title and login button when idle', () => {
      renderLoginPage()

      expect(screen.getByText('QQ 音乐登录')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: '登录 QQ 音乐' })).toBeInTheDocument()
    })

    it('calls startLogin when button is clicked', () => {
      const startLogin = vi.fn()
      mockQQLogin({ startLogin })
      renderLoginPage()

      fireEvent.click(screen.getByRole('button', { name: '登录 QQ 音乐' }))
      expect(startLogin).toHaveBeenCalled()
    })

    it('shows loading text when status is loading', () => {
      mockQQLogin({ loginStatus: 'loading' })
      renderLoginPage()

      expect(screen.getByText('加载中...')).toBeInTheDocument()
    })

    it('shows QR code and pending_scan status text', () => {
      mockQQLogin({ loginStatus: 'pending_scan', qrcodeUrl: 'https://qr.example.com/img' })
      renderLoginPage()

      const img = screen.getByAltText('QQ 音乐登录二维码')
      expect(img).toBeInTheDocument()
      expect(img).toHaveAttribute('src', 'https://qr.example.com/img')
      expect(screen.getByText('请用QQ音乐App扫描二维码')).toBeInTheDocument()
    })

    it('shows QR code and scanned status text', () => {
      mockQQLogin({ loginStatus: 'scanned', qrcodeUrl: 'https://qr.example.com/img' })
      renderLoginPage()

      expect(screen.getByText('已扫码，确认中...')).toBeInTheDocument()
    })

    it('shows confirmed with username and logout button', () => {
      mockQQLogin({ loginStatus: 'confirmed', userName: 'QQUser', isLoggedIn: true })
      renderLoginPage()

      expect(screen.getByText(/已登录: QQUser/)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: '登出' })).toBeInTheDocument()
    })

    it('calls logout when 登出 button is clicked', () => {
      const logout = vi.fn()
      mockQQLogin({ loginStatus: 'confirmed', userName: 'QQUser', isLoggedIn: true, logout })
      renderLoginPage()

      fireEvent.click(screen.getByRole('button', { name: '登出' }))
      expect(logout).toHaveBeenCalled()
    })

    it('shows expired with retry button', () => {
      const startLogin = vi.fn()
      mockQQLogin({ loginStatus: 'expired', startLogin })
      renderLoginPage()

      expect(screen.getByText('二维码已过期')).toBeInTheDocument()
      fireEvent.click(screen.getByRole('button', { name: '重新获取' }))
      expect(startLogin).toHaveBeenCalled()
    })

    it('shows error with retry button', () => {
      const startLogin = vi.fn()
      mockQQLogin({ loginStatus: 'error', startLogin })
      renderLoginPage()

      expect(screen.getByText('获取二维码失败')).toBeInTheDocument()
      fireEvent.click(screen.getByRole('button', { name: '重试' }))
      expect(startLogin).toHaveBeenCalled()
    })

    it('does not show QR code when qrcodeUrl is null even in pending_scan', () => {
      mockQQLogin({ loginStatus: 'pending_scan', qrcodeUrl: null })
      renderLoginPage()

      expect(screen.queryByAltText('QQ 音乐登录二维码')).not.toBeInTheDocument()
    })
  })
})
