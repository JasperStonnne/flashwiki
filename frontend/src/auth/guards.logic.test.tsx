import type { ReactElement } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { Navigate } from 'react-router-dom'

import * as authContext from './AuthContext'
import { RequireAuth } from './RequireAuth'
import { RequireRole } from './RequireRole'

function mockAuth(accessToken: string | null, userRole: string | null) {
  vi.spyOn(authContext, 'useAuth').mockReturnValue({
    accessToken,
    userRole,
    login: vi.fn(),
    logout: vi.fn(),
    refresh: vi.fn().mockResolvedValue(false),
  })
}

describe('guard logic', () => {
  it('RequireAuth returns Navigate when token is missing', () => {
    mockAuth(null, null)
    const element = RequireAuth({ children: <div>Home</div> }) as ReactElement<{ to: string }>
    expect(element.type).toBe(Navigate)
    expect(element.props.to).toBe('/login')
  })

  it('RequireRole lets manager pass through', () => {
    mockAuth('x', 'manager')
    const element = RequireRole({
      role: 'manager',
      children: <div>Admin Users</div>,
    }) as ReactElement<{ children: ReactElement<{ children: string }> }>
    expect(element.type).not.toBe(Navigate)
    expect(element.props.children.props.children).toBe('Admin Users')
  })
})
