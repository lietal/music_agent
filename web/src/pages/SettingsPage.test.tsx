import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import SettingsPage from './SettingsPage'

describe('SettingsPage', () => {
  it('renders user info', async () => {
    render(<MemoryRouter><SettingsPage /></MemoryRouter>)
    await waitFor(() => expect(screen.getByText('Test User')).toBeDefined(), { timeout: 2000 })
    expect(screen.getByText('password')).toBeDefined()
  })
})
