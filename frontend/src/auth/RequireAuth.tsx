import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuth } from './AuthContext'

type RequireAuthProps = {
  children: ReactNode
}

export function RequireAuth({ children }: RequireAuthProps) {
  const { accessToken } = useAuth()
  if (!accessToken) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}
