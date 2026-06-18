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

import { api } from '../api/client'
import LoginPage from './LoginPage'

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
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
})
