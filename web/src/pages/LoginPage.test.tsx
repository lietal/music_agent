import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

vi.mock('../hooks/useQQMusicLogin', () => ({
  useQQMusicLogin: vi.fn(),
}))

import { useQQMusicLogin } from '../hooks/useQQMusicLogin'
import LoginPage from './LoginPage'

function mockQQLogin(overrides: Record<string, unknown> = {}) {
  vi.mocked(useQQMusicLogin).mockReturnValue({
    loginStatus: 'idle',
    qrcodeUrl: null,
    userName: null,
    errorMsg: null,
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

  it('renders heading and subtitle', () => {
    renderLoginPage()
    expect(screen.getByText('Music Agent')).toBeInTheDocument()
    expect(screen.getByText('用 QQ 音乐扫码登录')).toBeInTheDocument()
  })

  it('shows login button when idle', () => {
    renderLoginPage()
    expect(screen.getByRole('button', { name: '登录 QQ 音乐' })).toBeInTheDocument()
  })

  it('calls startLogin when button is clicked', () => {
    const startLogin = vi.fn()
    mockQQLogin({ startLogin })
    renderLoginPage()
    fireEvent.click(screen.getByRole('button', { name: '登录 QQ 音乐' }))
    expect(startLogin).toHaveBeenCalled()
  })

  it('shows loading text', () => {
    mockQQLogin({ loginStatus: 'loading' })
    renderLoginPage()
    expect(screen.getByText('加载中...')).toBeInTheDocument()
  })

  it('shows QR code and pending_scan text', () => {
    mockQQLogin({ loginStatus: 'pending_scan', qrcodeUrl: 'https://qr.example.com/img' })
    renderLoginPage()
    const img = screen.getByAltText('QQ 音乐登录二维码')
    expect(img).toBeInTheDocument()
    expect(img).toHaveAttribute('src', 'https://qr.example.com/img')
    expect(screen.getByText('请用QQ音乐App扫描二维码')).toBeInTheDocument()
  })

  it('shows scanned status text', () => {
    mockQQLogin({ loginStatus: 'scanned', qrcodeUrl: 'https://qr.example.com/img' })
    renderLoginPage()
    expect(screen.getByText('已扫码，确认中...')).toBeInTheDocument()
  })

  it('shows confirmed with logout button', () => {
    mockQQLogin({ loginStatus: 'confirmed', userName: 'QQUser', isLoggedIn: true })
    renderLoginPage()
    expect(screen.getByText(/已登录: QQUser/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '登出' })).toBeInTheDocument()
  })

  it('calls logout when 登出 clicked', () => {
    const logout = vi.fn()
    mockQQLogin({ loginStatus: 'confirmed', userName: 'QQUser', logout })
    renderLoginPage()
    fireEvent.click(screen.getByRole('button', { name: '登出' }))
    expect(logout).toHaveBeenCalled()
  })

  it('shows expired with retry', () => {
    const startLogin = vi.fn()
    mockQQLogin({ loginStatus: 'expired', startLogin })
    renderLoginPage()
    expect(screen.getByText('二维码已过期')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '重新获取' }))
    expect(startLogin).toHaveBeenCalled()
  })

  it('shows error with retry', () => {
    const startLogin = vi.fn()
    mockQQLogin({ loginStatus: 'error', startLogin })
    renderLoginPage()
    expect(screen.getByText('获取二维码失败')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '重试' }))
    expect(startLogin).toHaveBeenCalled()
  })

  it('shows custom error message', () => {
    mockQQLogin({ loginStatus: 'error', errorMsg: '网络错误' })
    renderLoginPage()
    expect(screen.getByText('网络错误')).toBeInTheDocument()
  })
})
