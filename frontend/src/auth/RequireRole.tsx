import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'

type RequireRoleProps = {
  role: string
  children: ReactNode
}

export function RequireRole({ role, children }: RequireRoleProps) {
  const currentRole = localStorage.getItem('user_role')
  if (currentRole !== role) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}
