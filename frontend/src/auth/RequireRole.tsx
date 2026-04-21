import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuth } from './AuthContext'

type RequireRoleProps = {
  role: string
  children: ReactNode
}

export function RequireRole({ role, children }: RequireRoleProps) {
  const { userRole } = useAuth()
  const currentRole = userRole
  if (currentRole !== role) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}
