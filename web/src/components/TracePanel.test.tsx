import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import TracePanel from './TracePanel'

describe('TracePanel', () => {
  it('renders empty state', () => {
    render(<TracePanel steps={[]} />)
    expect(screen.getByText('Trace')).toBeDefined()
  })
  it('renders steps', () => {
    render(<TracePanel steps={[{ id: 't1', type: 'plan', name: 'Plan', status: 'done' }]} />)
    expect(screen.getByText('Plan')).toBeDefined()
  })
  it('has close button', () => {
    const { container } = render(<TracePanel steps={[{ id: 't1', type: 'plan', name: 'P', status: 'done' }]} />)
    expect(container.querySelectorAll('button').length).toBeGreaterThan(0)
  })
})
