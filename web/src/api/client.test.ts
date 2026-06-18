import { describe, it, expect } from 'vitest'
import { api } from './client'

describe('api', () => {
  it('exports auth.login', () => {
    expect(typeof api.auth.login).toBe('function')
  })

  it('exports auth.register', () => {
    expect(typeof api.auth.register).toBe('function')
  })

  it('exports chat.send', () => {
    expect(typeof api.chat.send).toBe('function')
  })

  it('exports conversations.list', () => {
    expect(typeof api.conversations.list).toBe('function')
  })
})
