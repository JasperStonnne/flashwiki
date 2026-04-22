import type { ReactNode } from 'react'
import { createContext, useContext, useEffect, useRef, useState } from 'react'

import { API_ENDPOINTS } from '../api/endpoints'
import { setAuthHandlers } from '../api/client'

const ACCESS_TOKEN_KEY = 'access_token'
const REFRESH_TOKEN_KEY = 'refresh_token'
const USER_ROLE_KEY = 'user_role'

type TokenPair = {
  access_token: string
  refresh_token: string
}

type Envelope<T> =
  | { success: true; data: T; error: null }
  | { success: false; data: null; error: { code: string; message: string } }

type AuthContextValue = {
  accessToken: string | null
  userRole: string | null
  login: (accessToken: string, refreshToken: string, role: string) => void
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

  function clearAuthState() {
    setAccessToken(null)
    setUserRole(null)
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
    localStorage.removeItem(USER_ROLE_KEY)
  }

  function login(nextAccessToken: string, nextRefreshToken: string, role: string) {
    setAccessToken(nextAccessToken)
    setUserRole(role)
    localStorage.setItem(ACCESS_TOKEN_KEY, nextAccessToken)
    localStorage.setItem(REFRESH_TOKEN_KEY, nextRefreshToken)
    localStorage.setItem(USER_ROLE_KEY, role)
  }

  function logout() {
    const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY)
    const token = localStorage.getItem(ACCESS_TOKEN_KEY)

    if (refreshToken) {
      fetch(API_ENDPOINTS.authLogout, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
      }).catch(() => {
        // no-op: local cleanup is the critical path
      })
    }

    clearAuthState()
    window.location.href = '/login'
  }

  async function refresh(): Promise<boolean> {
    const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY)
    if (!refreshToken) {
      return false
    }

    try {
      const response = await fetch(API_ENDPOINTS.authRefresh, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      })
      if (!response.ok) {
        return false
      }

      const envelope = (await response.json()) as Envelope<TokenPair>
      if (!envelope.success) {
        return false
      }

      setAccessToken(envelope.data.access_token)
      localStorage.setItem(ACCESS_TOKEN_KEY, envelope.data.access_token)
      localStorage.setItem(REFRESH_TOKEN_KEY, envelope.data.refresh_token)
      return true
    } catch {
      return false
    }
  }

  const refreshRef = useRef(refresh)
  refreshRef.current = refresh

  useEffect(() => {
    setAuthHandlers({
      getAccessToken: () => localStorage.getItem(ACCESS_TOKEN_KEY),
      refresh: () => refreshRef.current(),
      logout,
    })

    return () => {
      setAuthHandlers(null)
    }
  }, [accessToken])

  useEffect(() => {
    if (!accessToken) {
      return
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsURL = `${protocol}//${window.location.host}/api/ws?token=${encodeURIComponent(accessToken)}`
    const socket = new WebSocket(wsURL)

    socket.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as { type?: string }
        if (message.type === 'force_logout') {
          clearAuthState()
          window.location.href = '/login'
        }
      } catch {
        // ignore non-json message
      }
    }

    return () => {
      socket.close()
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
