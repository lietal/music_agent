import { useState, useRef, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Send } from 'lucide-react'
import { api } from '../api/client'
import { useSSE } from '../hooks/useSSE'
import { usePlayerStore } from '../hooks/usePlayerStore'
import type { ChatMessage } from '../types'
import AgentMessageList from '../components/AgentMessageList'
import TracePanel from '../components/TracePanel'
import PlayerBar from '../components/PlayerBar'
import PlayerPanel from '../components/PlayerPanel'

export default function ChatPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const conversationId = searchParams.get('id')
  const [input, setInput] = useState('')
  const [loadingHistory, setLoadingHistory] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const {
    messages,
    setMessages,
    isStreaming,
    traceSteps,
    start,
  } = useSSE()

  const {
    state,
    play,
    togglePlay,
    next,
    prev,
    seek,
    setVolume,
    addToQueue,
    removeFromQueue,
    togglePanel,
  } = usePlayerStore()

  useEffect(() => {
    if (!conversationId) return

    let cancelled = false
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoadingHistory(true)

    api.conversations
      .get(conversationId)
      .then((data) => {
        if (cancelled) return
        if (Array.isArray(data.messages)) {
          setMessages(data.messages as ChatMessage[])
        }
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoadingHistory(false)
      })

    return () => { cancelled = true }
  }, [conversationId, setMessages])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, traceSteps])

  const handleSend = useCallback(async () => {
    const text = input.trim()
    if (!text || isStreaming) return

    setInput('')

    const userMsg: ChatMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content: text,
      timestamp: Date.now(),
    }
    setMessages((prev: ChatMessage[]) => [...prev, userMsg])

    try {
      const result = await api.chat.send(text, conversationId ?? undefined)
      if (result.conversationId && !conversationId) {
        setSearchParams({ id: result.conversationId })
      }
      start(result.runId)
    } catch {
      setMessages((prev: ChatMessage[]) => [...prev, {
        id: crypto.randomUUID(),
        role: 'agent',
        content: '发送失败，请重试',
        timestamp: Date.now(),
      }])
    }
  }, [input, isStreaming, conversationId, setMessages, setSearchParams, start])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSend()
      }
    },
    [handleSend],
  )

  return (
    <div className="flex h-full">
      <div className="flex-1 flex flex-col min-w-0">
        <div className="flex-1 overflow-auto p-4 pb-14">
          {loadingHistory ? (
            <div className="flex items-center justify-center h-full">
              <p className="text-gray-500">加载中...</p>
            </div>
          ) : messages.length === 0 && !isStreaming ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-center">
                <h2 className="text-2xl font-bold text-gray-300 mb-2">
                  Music Agent
                </h2>
                <p className="text-gray-500">输入你想听的音乐，开始探索</p>
              </div>
            </div>
          ) : (
            <>
              <AgentMessageList
                messages={messages}
                isStreaming={isStreaming}
                traceSteps={traceSteps}
                onPlaySong={play}
                onAddToQueue={addToQueue}
                currentSongId={state.currentSong?.id ?? null}
              />
              <div ref={messagesEndRef} />
            </>
          )}
        </div>

        <div className="border-t border-gray-800 p-4 pb-14">
          <div className="flex gap-2 max-w-3xl mx-auto">
            <input
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="输入你想听的音乐..."
              disabled={isStreaming}
              className="flex-1 bg-gray-800 text-gray-100 rounded-xl px-4 py-3 outline-none focus:ring-2 focus:ring-indigo-500 placeholder-gray-500 disabled:opacity-50 text-sm"
            />
            <button
              onClick={handleSend}
              disabled={!input.trim() || isStreaming}
              className="px-5 py-3 bg-indigo-600 text-white rounded-xl hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              <Send size={18} />
            </button>
          </div>
        </div>
      </div>

      <TracePanel steps={traceSteps} />

      {state.panelOpen && (
        <PlayerPanel
          currentSong={state.currentSong}
          isPlaying={state.isPlaying}
          currentTime={state.currentTime}
          duration={state.duration}
          volume={state.volume}
          lyrics={state.lyrics}
          activeLyricIndex={state.activeLyricIndex}
          queue={state.queue}
          queueIndex={state.queueIndex}
          onTogglePlay={togglePlay}
          onPrev={prev}
          onNext={next}
          onSeek={seek}
          onVolumeChange={setVolume}
          onPlaySong={play}
          onRemoveFromQueue={removeFromQueue}
          onClose={togglePanel}
        />
      )}

      <PlayerBar
        currentSong={state.currentSong}
        isPlaying={state.isPlaying}
        currentTime={state.currentTime}
        duration={state.duration}
        onTogglePlay={togglePlay}
        onTogglePanel={togglePanel}
        onNext={next}
      />
    </div>
  )
}
