import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import '@testing-library/jest-dom/vitest'
import QueuePanel from './QueuePanel'
import type { Song } from '../types'

const song1: Song = { id: '1', title: 'Song One', artist: 'Artist A' }
const song2: Song = { id: '2', title: 'Song Two', artist: 'Artist B' }
const song3: Song = { id: '3', title: 'Song Three', artist: 'Artist C' }

describe('QueuePanel', () => {
  it('renders empty state when queue is empty', () => {
    render(
      <QueuePanel queue={[]} currentIndex={-1} onPlaySong={vi.fn()} onRemove={vi.fn()} />
    )
    expect(screen.getByText('队列为空')).toBeInTheDocument()
  })

  it('renders queue items with index, title, and artist', () => {
    render(
      <QueuePanel
        queue={[song1, song2]}
        currentIndex={-1}
        onPlaySong={vi.fn()}
        onRemove={vi.fn()}
      />
    )
    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
    expect(screen.getByText('Song One')).toBeInTheDocument()
    expect(screen.getByText('Artist A')).toBeInTheDocument()
    expect(screen.getByText('Song Two')).toBeInTheDocument()
    expect(screen.getByText('Artist B')).toBeInTheDocument()
  })

  it('highlights current song with green text', () => {
    render(
      <QueuePanel
        queue={[song1, song2, song3]}
        currentIndex={1}
        onPlaySong={vi.fn()}
        onRemove={vi.fn()}
      />
    )
    const rows = screen.getAllByRole('listitem')
    expect(rows[0]).not.toHaveClass('text-green-400')
    expect(rows[1]).toHaveClass('text-green-400')
    expect(rows[2]).not.toHaveClass('text-green-400')
  })

  it('calls onPlaySong when clicking a non-current song', async () => {
    const onPlaySong = vi.fn()
    render(
      <QueuePanel
        queue={[song1, song2]}
        currentIndex={-1}
        onPlaySong={onPlaySong}
        onRemove={vi.fn()}
      />
    )
    await userEvent.click(screen.getByText('Song One'))
    expect(onPlaySong).toHaveBeenCalledWith(song1)
  })

  it('does not call onPlaySong when clicking the current song', async () => {
    const onPlaySong = vi.fn()
    render(
      <QueuePanel
        queue={[song1, song2]}
        currentIndex={0}
        onPlaySong={onPlaySong}
        onRemove={vi.fn()}
      />
    )
    await userEvent.click(screen.getByText('Song One'))
    expect(onPlaySong).not.toHaveBeenCalled()
  })

  it('calls onRemove when clicking X button', async () => {
    const onRemove = vi.fn()
    render(
      <QueuePanel
        queue={[song1, song2]}
        currentIndex={-1}
        onPlaySong={vi.fn()}
        onRemove={onRemove}
      />
    )
    const removeButtons = screen.getAllByRole('button', { name: /remove/i })
    await userEvent.click(removeButtons[1])
    expect(onRemove).toHaveBeenCalledWith(1)
  })
})
