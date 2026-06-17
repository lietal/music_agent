export interface Conversation {
  id: string
  title?: string
  createdAt?: string
  updatedAt?: string
  messages?: unknown[]
}

const API_BASE = ''

export function getToken(): string | null {
  return localStorage.getItem('jwt')
}

export function setToken(token: string) {
  localStorage.setItem('jwt', token)
}

export function clearToken() {
  localStorage.removeItem('jwt')
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken()
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers ?? {}),
    },
    ...init,
  })
  if (!response.ok) throw new Error(`HTTP ${response.status}`)
  return response.json() as T
}

export const api = {
  auth: {
    me: () => request<{ id: string; displayName: string; provider?: string }>('/api/auth/me'),
  },
  chat: {
    send: (message: string, conversationId?: string) =>
      request<{ runId: string; conversationId?: string }>('/api/chat', {
        method: 'POST',
        body: JSON.stringify({ message, conversationId }),
      }),
    eventsUrl: (runId: string) => `/api/chat/${runId}/events?token=${getToken()}`,
  },
  conversations: {
    list: () => request<Conversation[]>('/api/conversations'),
    get: (id: string) => request<Conversation>(`/api/conversations/${id}`),
    create: () =>
      request<Conversation>('/api/conversations', { method: 'POST', body: '{}' }),
    delete: (id: string) =>
      request<void>(`/api/conversations/${id}`, { method: 'DELETE' }),
  },
}
