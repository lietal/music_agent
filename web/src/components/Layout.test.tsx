import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import '@testing-library/jest-dom/vitest'
import Layout from './Layout'

describe('Layout', () => {
  beforeEach(() => { localStorage.setItem('jwt', 'test-token') })
  afterEach(() => { localStorage.clear() })

  it('renders three nav links', () => {
    render(<BrowserRouter><Layout /></BrowserRouter>)
    const links = screen.getAllByRole('link')
    expect(links).toHaveLength(3)
  })
})
