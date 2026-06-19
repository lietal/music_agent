import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useSSE } from './useSSE'
import { renderHook, act } from '@testing-library/react'

class FakeEventSource {
  listeners: Record<string, ((e: { data: string }) => void)[]> = {}
  onerror: ((e: unknown) => void) | null = null
  closed = false
  static instance: FakeEventSource
  constructor(_url: string) { FakeEventSource.instance = this }
  addEventListener(type: string, h: (e: { data: string }) => void) {
    if (!this.listeners[type]) this.listeners[type] = []
    this.listeners[type].push(h)
  }
  close() { this.closed = true }
}

beforeEach(() => {
  localStorage.clear()
  vi.restoreAllMocks()
  vi.stubGlobal('crypto', { randomUUID: () => 'uuid-1' })
  ;(globalThis as any).EventSource = FakeEventSource
  localStorage.setItem('jwt', 'test-jwt')
})

const es = () => FakeEventSource.instance

describe('useSSE', () => {
  it('start sets streaming and adds agent msg', () => {
    const { result } = renderHook(() => useSSE())
    act(() => { result.current.start('run-1') })
    expect(result.current.isStreaming).toBe(true)
    expect(result.current.messages[0].role).toBe('agent')
  })

  it('plan event', () => {
    const { result } = renderHook(() => useSSE())
    act(() => { result.current.start('run-1') })
    act(() => { es().listeners['plan']?.[0]?.({ data: JSON.stringify({ plan: 'search' }) }) })
    expect(result.current.traceSteps[0].type).toBe('plan')
  })

  it('tool_start event', () => {
    const { result } = renderHook(() => useSSE())
    act(() => { result.current.start('run-1') })
    act(() => { es().listeners['tool_start']?.[0]?.({ data: JSON.stringify({ name: 's', input: 'q' }) }) })
    expect(result.current.traceSteps[0].status).toBe('running')
  })

  it('tool_done adds songs', () => {
    const { result } = renderHook(() => useSSE())
    act(() => { result.current.start('run-1') })
    act(() => { es().listeners['tool_start']?.[0]?.({ data: JSON.stringify({ name: 's' }) }) })
    act(() => { es().listeners['tool_done']?.[0]?.({ data: JSON.stringify({ songs: [{ title: '晴天' }] }) }) })
    expect(result.current.messages[0].songs).toEqual([{ title: '晴天' }])
  })

  it('delta appends', () => {
    const { result } = renderHook(() => useSSE())
    act(() => { result.current.start('run-1') })
    act(() => { es().listeners['delta']?.[0]?.({ data: JSON.stringify({ message: 'a' }) }) })
    act(() => { es().listeners['delta']?.[0]?.({ data: JSON.stringify({ message: 'b' }) }) })
    expect(result.current.messages[0].content).toBe('ab')
  })

  it('done stops', () => {
    const { result } = renderHook(() => useSSE())
    act(() => { result.current.start('run-1') })
    act(() => { es().listeners['done']?.[0]?.({ data: '' }) })
    expect(result.current.isStreaming).toBe(false)
  })

  it('error with message', () => {
    const { result } = renderHook(() => useSSE())
    act(() => { result.current.start('run-1') })
    act(() => { es().listeners['error']?.[0]?.({ data: JSON.stringify({ message: 'fail' }) }) })
    expect(result.current.messages[0].content).toBe('fail')
  })

  it('error non-JSON', () => {
    const { result } = renderHook(() => useSSE())
    act(() => { result.current.start('run-1') })
    act(() => { es().listeners['error']?.[0]?.({ data: 'bad' }) })
    expect(result.current.isStreaming).toBe(false)
  })

  it('onerror stops', () => {
    const { result } = renderHook(() => useSSE())
    act(() => { result.current.start('run-1') })
    act(() => { es().onerror?.({}) })
    expect(result.current.isStreaming).toBe(false)
  })

  it('reset clears', () => {
    const { result } = renderHook(() => useSSE())
    act(() => { result.current.start('run-1') })
    act(() => { result.current.reset() })
    expect(result.current.messages).toHaveLength(0)
  })
})
