import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useQQMusicLogin } from './useQQMusicLogin'

vi.mock('../api/player', () => ({
  getLoginQRCode: vi.fn(),
  checkQRStatus: vi.fn(),
}))

import { getLoginQRCode, checkQRStatus } from '../api/player'

describe('useQQMusicLogin', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('starts with idle status', () => {
    const { result } = renderHook(() => useQQMusicLogin())
    expect(result.current.loginStatus).toBe('idle')
    expect(result.current.qrcodeUrl).toBeNull()
    expect(result.current.isLoggedIn).toBe(false)
  })

  it('startLogin sets qrcodeUrl and pending_scan', async () => {
    vi.mocked(getLoginQRCode).mockResolvedValue({ qrcode_url: 'https://qr.example.com/test', key: 'test_key' })
    const { result } = renderHook(() => useQQMusicLogin())

    await act(() => result.current.startLogin())

    expect(result.current.qrcodeUrl).toBe('https://qr.example.com/test')
    expect(result.current.loginStatus).toBe('pending_scan')
    expect(result.current.isLoggedIn).toBe(false)
  })

  it('startLogin sets error on failure', async () => {
    vi.mocked(getLoginQRCode).mockRejectedValue(new Error('network error'))
    const { result } = renderHook(() => useQQMusicLogin())

    await act(() => result.current.startLogin())

    expect(result.current.loginStatus).toBe('error')
  })

  it('checkStatus confirms login', async () => {
    vi.mocked(getLoginQRCode).mockResolvedValue({ qrcode_url: 'https://qr.example.com/test', key: 'test_key' })
    vi.mocked(checkQRStatus).mockResolvedValue({ status: 'confirmed', user_name: 'TestUser' })
    const { result } = renderHook(() => useQQMusicLogin())

    await act(() => result.current.startLogin())
    await act(() => result.current.checkStatus())

    expect(result.current.loginStatus).toBe('confirmed')
    expect(result.current.userName).toBe('TestUser')
    expect(result.current.isLoggedIn).toBe(true)
  })

  it('checkStatus handles scanned then expired states', async () => {
    vi.mocked(getLoginQRCode).mockResolvedValue({ qrcode_url: 'qr', key: 'key' })
    vi.mocked(checkQRStatus).mockResolvedValueOnce({ status: 'scanned' })
    const { result } = renderHook(() => useQQMusicLogin())

    await act(() => result.current.startLogin())
    await act(() => result.current.checkStatus())
    expect(result.current.loginStatus).toBe('scanned')

    vi.mocked(checkQRStatus).mockResolvedValueOnce({ status: 'expired' })
    await act(() => result.current.checkStatus())
    expect(result.current.loginStatus).toBe('expired')
  })

  it('logout resets state', async () => {
    vi.mocked(getLoginQRCode).mockResolvedValue({ qrcode_url: 'qr', key: 'key' })
    vi.mocked(checkQRStatus).mockResolvedValue({ status: 'confirmed', user_name: 'TestUser' })
    const { result } = renderHook(() => useQQMusicLogin())

    await act(() => result.current.startLogin())
    await act(() => result.current.checkStatus())
    expect(result.current.isLoggedIn).toBe(true)

    act(() => result.current.logout())
    expect(result.current.loginStatus).toBe('idle')
    expect(result.current.isLoggedIn).toBe(false)
  })
})
