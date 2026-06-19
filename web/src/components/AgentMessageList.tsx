import { Loader, Check } from 'lucide-react'
import type { ChatMessage, TraceStep, Song } from '../types'
import SongCards from './SongCards'

interface AgentMessageListProps {
  messages: ChatMessage[]
  isStreaming: boolean
  traceSteps: TraceStep[]
  onPlaySong?: (song: Song) => void
  onAddToQueue?: (song: Song) => void
  currentSongId?: string | null
}

export default function AgentMessageList({
  messages,
  isStreaming,
  traceSteps,
  onPlaySong,
  onAddToQueue,
  currentSongId,
}: AgentMessageListProps) {
  if (messages.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-gray-500">开始对话，探索音乐世界</p>
      </div>
    )
  }

  return (
    <div className="space-y-4 pb-4">
      {messages.map((msg: ChatMessage, idx: number) => {
        const isLastAgent = msg.role === 'agent' && idx === messages.length - 1

        return (
          <div key={msg.id}>
            {msg.role === 'user' ? (
              <div className="flex justify-end">
                <div className="max-w-[80%] bg-indigo-600 text-white rounded-2xl rounded-br-md px-4 py-2.5">
                  <p className="text-sm whitespace-pre-wrap break-words">{msg.content}</p>
                </div>
              </div>
            ) : (
              <div className="flex justify-start">
                <div className="max-w-[80%] bg-gray-800 text-gray-100 rounded-2xl rounded-bl-md px-4 py-2.5">
                  <p className="text-sm whitespace-pre-wrap break-words">
                    {msg.content}
                    {isLastAgent && isStreaming && (
                      <span className="inline-block w-1.5 h-4 bg-indigo-400 ml-0.5 animate-pulse align-text-bottom" />
                    )}
                  </p>
                  {msg.songs && msg.songs.length > 0 && (
                    <SongCards
                      songs={msg.songs}
                      onPlay={onPlaySong}
                      onAddToQueue={onAddToQueue}
                      currentSongId={currentSongId}
                    />
                  )}
                </div>
              </div>
            )}
          </div>
        )
      })}

      {isStreaming && traceSteps.filter((s: TraceStep) => s.type !== 'plan').length > 0 && (
        <div className="flex flex-col gap-1 px-1">
          {traceSteps
            .filter((s: TraceStep) => s.type !== 'plan')
            .map((step: TraceStep) => (
              <div key={step.id} className="flex items-center gap-2 text-xs">
                {step.status === 'running' ? (
                  <Loader size={12} className="animate-spin text-indigo-400 flex-shrink-0" />
                ) : (
                  <Check size={12} className="text-green-400 flex-shrink-0" />
                )}
                <span className="text-gray-500 truncate">{step.name}</span>
              </div>
            ))}
        </div>
      )}
    </div>
  )
}
