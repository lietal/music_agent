import { useState, useRef, useCallback } from 'react'
import type { ChatMessage, TraceStep } from '../types'

export type { ChatMessage, TraceStep }

export function useSSE() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [isStreaming, setIsStreaming] = useState(false)
  const [traceSteps, setTraceSteps] = useState<TraceStep[]>([])
  const esRef = useRef<EventSource | null>(null)
  const agentMsgIdRef = useRef<string | null>(null)

  const start = useCallback((runId: string) => {
    esRef.current?.close()
    setIsStreaming(true)
    setTraceSteps([])

    const agentMsgId = crypto.randomUUID()
    agentMsgIdRef.current = agentMsgId
    const agentMsg: ChatMessage = {
      id: agentMsgId,
      role: 'agent',
      content: '',
      timestamp: Date.now(),
    }
    setMessages((prev: ChatMessage[]) => [...prev, agentMsg])

    const token = localStorage.getItem('jwt')
    const es = new EventSource(`/api/chat/${runId}/events?token=${token}`)
    esRef.current = es

    es.addEventListener('plan', (e) => {
      const data = JSON.parse((e as MessageEvent).data)
      setTraceSteps((prev: TraceStep[]) => [...prev, {
        id: crypto.randomUUID(),
        type: 'plan',
        name: 'Plan',
        status: 'done',
        details: data.plan,
      }])
    })

    es.addEventListener('tool_start', (e) => {
      const data = JSON.parse((e as MessageEvent).data)
      setTraceSteps((prev: TraceStep[]) => [...prev, {
        id: crypto.randomUUID(),
        type: 'tool_start',
        name: data.name,
        status: 'running',
        details: data.input ? JSON.stringify(data.input) : undefined,
      }])
    })

    es.addEventListener('tool_done', (e) => {
      const data = JSON.parse((e as MessageEvent).data)
      setTraceSteps((prev: TraceStep[]) => {
        const reversed = [...prev].reverse()
        const lastIdx = reversed.findIndex(
          (s: TraceStep) => s.type === 'tool_start' && s.status === 'running'
        )
        if (lastIdx === -1) return prev
        const idx = prev.length - 1 - lastIdx
        return prev.map((s: TraceStep, i: number) =>
          i === idx
            ? { ...s, status: 'done' as const, details: data.output ? JSON.stringify(data.output) : s.details }
            : s
        )
      })
      if (data.songs || data.output?.songs) {
        const songs = data.songs || data.output.songs
        setMessages((prev: ChatMessage[]) => prev.map((m: ChatMessage) =>
          m.id === agentMsgId ? { ...m, songs } : m
        ))
      }
      // Also extract songs from tool result data (backend nests songs in result.data as JSON string)
      const resultData = data.result?.data || data.Result?.data
      if (resultData) {
        try {
          const parsed = JSON.parse(resultData)
          let rawSongs = Array.isArray(parsed) ? parsed : parsed?.songs || parsed?.output?.songs
          if (Array.isArray(rawSongs) && rawSongs.length > 0) {
            const songs = rawSongs.map((s: any) => ({
              id: s.id || '',
              title: s.title || '',
              artist: s.artist || (Array.isArray(s.artists) ? s.artists[0] : '') || '',
              album: s.album || '',
              coverUrl: s.coverUrl || s.artwork_url || s.cover_url || '',
              durationSeconds: s.durationSeconds || s.duration_seconds,
            }))
            setMessages((prev: ChatMessage[]) => prev.map((m: ChatMessage) =>
              m.id === agentMsgId ? { ...m, songs } : m
            ))
          }
        } catch { /* not JSON, ignore */ }
      }
    })

    es.addEventListener('delta', (e) => {
      const data = JSON.parse((e as MessageEvent).data)
      setMessages((prev: ChatMessage[]) => prev.map((m: ChatMessage) =>
        m.id === agentMsgId ? { ...m, content: m.content + (data.text || data.message || '') } : m
      ))
    })

    es.addEventListener('done', () => {
      setIsStreaming(false)
      es.close()
    })

    es.addEventListener('error', (e) => {
      try {
        const data = JSON.parse((e as MessageEvent).data)
        setMessages((prev: ChatMessage[]) => prev.map((m: ChatMessage) =>
          m.id === agentMsgId
            ? { ...m, content: m.content || (data.message ?? 'Stream error') }
            : m
        ))
      } catch {
        void 0
      }
      setIsStreaming(false)
      es.close()
    })

    es.onerror = () => {
      setIsStreaming(false)
      es.close()
    }

    return () => es.close()
  }, [])

  const reset = useCallback(() => {
    esRef.current?.close()
    setMessages([])
    setIsStreaming(false)
    setTraceSteps([])
    agentMsgIdRef.current = null
  }, [])

  return { messages, setMessages, isStreaming, traceSteps, start, reset }
}
