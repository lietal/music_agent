import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import AgentMessageList from './AgentMessageList'

describe('AgentMessageList', () => {
  it('shows empty state', () => {
    render(<AgentMessageList messages={[]} isStreaming={false} traceSteps={[]} />)
    expect(screen.getByText(/开始对话/)).toBeDefined()
  })

  it('renders messages', () => {
    render(<AgentMessageList messages={[
      { id: '1', role: 'user', content: 'hi', timestamp: 1 },
      { id: '2', role: 'agent', content: 'hello', timestamp: 2 },
    ]} isStreaming={false} traceSteps={[]} />)
    expect(screen.getByText('hi')).toBeDefined()
    expect(screen.getByText('hello')).toBeDefined()
  })

  it('renders songs', () => {
    render(<AgentMessageList messages={[
      { id: '1', role: 'agent', content: '', timestamp: 1, songs: [{ id: 's1', title: '晴天', artist: '周杰伦' }] },
    ]} isStreaming={false} traceSteps={[]} />)
    expect(screen.getByText('晴天')).toBeDefined()
  })

  it('shows trace steps when streaming', () => {
    render(<AgentMessageList messages={[{ id: '1', role: 'agent', content: 'loading', timestamp: 1 }]} isStreaming={true} traceSteps={[
      { id: 't1', type: 'tool_start', name: 'search', status: 'running' },
      { id: 't2', type: 'tool_done', name: 'search', status: 'done' },
    ]} />)
    expect(screen.getByText('loading')).toBeDefined()
  })
})
