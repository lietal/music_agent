import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import Layout from './Layout'
describe('Layout', () => {
  it('renders nav when authenticated', () => {
    localStorage.setItem('jwt', 'test-jwt')
    const { container } = render(<MemoryRouter initialEntries={['/chat']}><Layout /></MemoryRouter>)
    expect(container.querySelector('nav')).toBeTruthy()
  })
})
