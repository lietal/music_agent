import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import LyricsPanel from './LyricsPanel'
import type { LyricLine } from '../types'

const sampleLyrics: LyricLine[] = [
  { time: 0, text: 'First line' },
  { time: 5, text: 'Second line' },
  { time: 10, text: 'Third line' },
]

describe('LyricsPanel', () => {
  it('shows empty state when lyrics is null', () => {
    render(<LyricsPanel lyrics={null} activeIndex={0} />)
    expect(screen.getByText('暂无歌词')).toBeInTheDocument()
  })

  it('renders all lyric lines as <p> elements', () => {
    render(<LyricsPanel lyrics={sampleLyrics} activeIndex={0} />)
    const lines = screen.getAllByRole('paragraph')
    expect(lines).toHaveLength(3)
  })

  it('highlights active line with green and large text', () => {
    render(<LyricsPanel lyrics={sampleLyrics} activeIndex={1} />)
    const activeLine = screen.getByText('Second line')
    expect(activeLine.className).toContain('text-green-400')
    expect(activeLine.className).toContain('text-lg')
  })

  it('renders inactive lines with gray and small text', () => {
    render(<LyricsPanel lyrics={sampleLyrics} activeIndex={1} />)
    const inactiveLine = screen.getByText('First line')
    expect(inactiveLine.className).toContain('text-gray-500')
    expect(inactiveLine.className).toContain('text-sm')
  })

  it('calls onLyricClick with the line time when clicked', () => {
    const onLyricClick = vi.fn()
    render(<LyricsPanel lyrics={sampleLyrics} activeIndex={0} onLyricClick={onLyricClick} />)
    fireEvent.click(screen.getByText('Second line'))
    expect(onLyricClick).toHaveBeenCalledWith(5)
  })

  it('does not throw when clicking with no onLyricClick handler', () => {
    render(<LyricsPanel lyrics={sampleLyrics} activeIndex={0} />)
    fireEvent.click(screen.getByText('First line'))
  })
})
