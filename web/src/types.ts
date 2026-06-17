export interface Song {
  id: string
  title: string
  artist: string
  album?: string
  coverUrl?: string
}

export interface ChatMessage {
  id: string
  role: 'user' | 'agent'
  content: string
  songs?: Song[]
  timestamp: number
}

export interface TraceStep {
  id: string
  type: 'plan' | 'tool_start' | 'tool_done'
  name: string
  status: 'running' | 'done' | 'error'
  details?: string
}
