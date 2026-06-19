import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import ChatPage from './ChatPage'

const startFn = vi.hoisted(() => vi.fn())
vi.mock('../hooks/useSSE', () => ({
  useSSE: () => ({ messages: [], setMessages: vi.fn(), isStreaming: false, traceSteps: [], start: startFn, reset: vi.fn() }),
}))

describe('ChatPage', () => {
  it('renders Music Agent title', () => {
    render(<MemoryRouter initialEntries={['/chat']}><ChatPage /></MemoryRouter>)
    expect(screen.getByText('Music Agent')).toBeDefined()
  })

  it('renders input and trace panel', () => {
    render(<MemoryRouter initialEntries={['/chat']}><ChatPage /></MemoryRouter>)
    expect(screen.getByPlaceholderText(/输入你想听的音乐/)).toBeDefined()
    expect(screen.getByText('Trace')).toBeDefined()
  })

  it('shows loading state with conversation id', () => {
    render(<MemoryRouter initialEntries={['/chat?id=conv-1']}><ChatPage /></MemoryRouter>)
    expect(screen.getByText('加载中...')).toBeDefined()
  })
})
