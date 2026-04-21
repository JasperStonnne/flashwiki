import { createContext, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'

import { setAuthHandlers } from '../api/client'

const ACCESS_TOKEN_KEY = 'access_token'
const USER_ROLE_KEY = 'user_role'

type AuthContextValue = {
  accessToken: string | null
  userRole: string | null
  login: (token: string, role: string) => void
  logout: () => void
  refresh: () => Promise<boolean>
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

type AuthProviderProps = {
  children: ReactNode
}

function readStorage(key: string): string | null {
  return localStorage.getItem(key)
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [accessToken, setAccessToken] = useState<string | null>(() => readStorage(ACCESS_TOKEN_KEY))
  const [userRole, setUserRole] = useState<string | null>(() => readStorage(USER_ROLE_KEY))

  function login(token: string, role: string) {
    setAccessToken(token)
    setUserRole(role)
    localStorage.setItem(ACCESS_TOKEN_KEY, token)
    localStorage.setItem(USER_ROLE_KEY, role)
  }

  function logout() {
    setAccessToken(null)
    setUserRole(null)
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(USER_ROLE_KEY)
    window.location.href = '/login'
  }

  async function refresh(): Promise<boolean> {
    logout()
    return false
  }

  useEffect(() => {
    setAuthHandlers({
      getAccessToken: () => accessToken,
      refresh,
      logout,
    })

    return () => {
      setAuthHandlers(null)
    }
  }, [accessToken])

  return (
    <AuthContext.Provider
      value={{
        accessToken,
        userRole,
        login,
        logout,
        refresh,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return context
}
