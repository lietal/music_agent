import '@testing-library/jest-dom/vitest'
import { vi } from 'vitest'

Element.prototype.scrollIntoView = vi.fn()

Object.defineProperty(window, 'localStorage', {
  value: (() => {
    let store: Record<string, string> = {}
    return {
      getItem: vi.fn((key: string) => store[key] ?? null),
      setItem: vi.fn((key: string, value: string) => { store[key] = value }),
      removeItem: vi.fn((key: string) => { delete store[key] }),
      clear: vi.fn(() => { store = {} }),
    }
  })(),
  writable: true,
})
