import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import SongCards from './SongCards'
import type { Song } from '../types'

const songs: Song[] = [
  { id: 's1', title: '晴天', artist: '周杰伦', coverUrl: 'https://example.com/1.jpg' },
]

describe('SongCards', () => {
  it('renders song title', () => {
    render(<SongCards songs={songs} />)
    expect(screen.getByText('晴天')).toBeDefined()
  })

  it('renders empty grid for no songs', () => {
    const { container } = render(<SongCards songs={[]} />)
    expect(container.querySelector('.grid')).toBeTruthy()
  })
})
