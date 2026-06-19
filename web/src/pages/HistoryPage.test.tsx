import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import HistoryPage from './HistoryPage'

describe('HistoryPage', () => {
  it('renders conversation list', async () => {
    render(<MemoryRouter><HistoryPage /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('周杰伦的歌')).toBeDefined(), { timeout: 2000 })
    expect(screen.getByText('推荐英文歌')).toBeDefined()
  })
})
