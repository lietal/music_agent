import { http, HttpResponse } from 'msw'

export const handlers = [
  // Auth
  http.post('/api/auth/login', () =>
    HttpResponse.json({ token: 'msw-jwt-token', user: { user_id: 'u1', display_name: 'Test User' } })
  ),
  http.post('/api/auth/register', () =>
    HttpResponse.json({ token: 'msw-jwt-reg', user: { user_id: 'u2', display_name: 'New User' } })
  ),
  http.get('/api/auth/me', () =>
    HttpResponse.json({ id: 'u1', displayName: 'Test User', provider: 'password' })
  ),

  // Chat
  http.post('/api/chat', () =>
    HttpResponse.json({ runId: 'msw-run-1' })
  ),

  // Conversations
  http.get('/api/conversations', () =>
    HttpResponse.json([
      { id: 'conv-1', title: '周杰伦的歌', createdAt: '2024-01-01' },
      { id: 'conv-2', title: '推荐英文歌', createdAt: '2024-01-02' },
    ])
  ),
  http.get('/api/conversations/:id', ({ params }) =>
    HttpResponse.json({ id: params.id, title: 'Test', messages: [] })
  ),
  http.post('/api/conversations', () =>
    HttpResponse.json({ id: 'conv-new' }, { status: 201 })
  ),
  http.delete('/api/conversations/:id', () =>
    HttpResponse.json({}, { status: 200 })
  ),
]

// Error variants
export const errorHandlers = [
  http.post('/api/auth/login', () =>
    HttpResponse.json({ error: 'invalid credentials' }, { status: 401 })
  ),
  http.post('/api/auth/register', () =>
    HttpResponse.json({ error: 'username taken' }, { status: 409 })
  ),
  http.get('/api/conversations', () =>
    HttpResponse.json({ error: 'server error' }, { status: 500 })
  ),
]
