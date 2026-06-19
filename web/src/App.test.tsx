import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import App from './App'

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
})

describe('App', () => {
  it('renders login at /', () => {
    render(<MemoryRouter initialEntries={['/']}><App /></MemoryRouter>)
    expect(screen.getByText(/登录/)).toBeDefined()
  })

  it('renders login at /login', () => {
    render(<MemoryRouter initialEntries={['/login']}><App /></MemoryRouter>)
    expect(screen.getByText(/登录/)).toBeDefined()
  })
})
