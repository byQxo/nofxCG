import React, { createContext, useContext, useEffect, useState } from 'react'
import { flushSync } from 'react-dom'
import { useNavigate } from 'react-router-dom'
import { invalidateSystemConfig } from '../lib/config'
import {
  clearAuthSession,
  getCurrentAuthSession,
  loadPersistedAuthSession,
  reset401Flag,
  setAuthSession,
} from '../lib/httpClient'
import { getPostAuthPath, setUserMode, type UserMode } from '../lib/onboarding'
import { ROUTES } from '../router/paths'

interface User {
  id: string
  email: string
}

interface AuthContextType {
  user: User | null
  token: string | null
  login: (
    adminKey: string,
    remember?: boolean,
    mode?: UserMode
  ) => Promise<{
    success: boolean
    message?: string
  }>
  loginAdmin: (
    adminKey: string,
    remember?: boolean
  ) => Promise<{
    success: boolean
    message?: string
  }>
  register: (
    _email: string,
    _password: string,
    _betaCode?: string,
    _mode?: UserMode
  ) => Promise<{ success: boolean; message?: string }>
  resetPassword: (
    _email: string,
    _newPassword: string
  ) => Promise<{ success: boolean; message?: string }>
  logout: () => Promise<void>
  isLoading: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

const LOCAL_ADMIN_USER: User = {
  id: 'local-admin',
  email: 'Local Admin',
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate()
  const [user, setUser] = useState<User | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    reset401Flag()

    const session = loadPersistedAuthSession()
    if (!session?.accessToken) {
      setIsLoading(false)
      return
    }

    fetch('/api/auth/status', {
      headers: {
        Authorization: `Bearer ${session.accessToken}`,
      },
    })
      .then(async (response) => response.json())
      .then((data: { is_logged_in?: boolean }) => {
        if (data.is_logged_in) {
          flushSync(() => {
            setUser(LOCAL_ADMIN_USER)
            setToken(session.accessToken)
          })
          return
        }
        clearAuthSession()
      })
      .catch(() => {
        clearAuthSession()
      })
      .finally(() => {
        setIsLoading(false)
      })
  }, [])

  useEffect(() => {
    const handleUnauthorized = () => {
      setUser(null)
      setToken(null)
      clearAuthSession()
    }

    window.addEventListener('unauthorized', handleUnauthorized)
    return () => {
      window.removeEventListener('unauthorized', handleUnauthorized)
    }
  }, [])

  const handlePostAuthSuccess = (
    accessToken: string,
    refreshToken: string,
    remember: boolean,
    mode?: UserMode
  ) => {
    reset401Flag()

    if (mode) {
      setUserMode(mode)
    }

    setAuthSession({
      accessToken,
      refreshToken,
      remember,
    })

    flushSync(() => {
      setToken(accessToken)
      setUser(LOCAL_ADMIN_USER)
    })

    const returnUrl = sessionStorage.getItem('returnUrl')
    const nextPath = returnUrl || getPostAuthPath(mode)
    if (returnUrl) {
      sessionStorage.removeItem('returnUrl')
    }

    navigate(nextPath)
  }

  const login = async (
    adminKey: string,
    remember = false,
    mode?: UserMode
  ) => {
    try {
      const response = await fetch('/api/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ admin_key: adminKey }),
      })

      const data = await response.json()
      if (!response.ok) {
        return {
          success: false,
          message: data.error || '管理员密钥错误或已被锁定',
        }
      }

      handlePostAuthSuccess(
        data.access_token,
        data.refresh_token,
        remember,
        mode
      )
      return { success: true }
    } catch {
      return { success: false, message: 'Login failed, please try again' }
    }
  }

  const loginAdmin = async (adminKey: string, remember = false) => {
    return login(adminKey, remember)
  }

  const register = async () => {
    return {
      success: false,
      message: 'Local offline mode removed the registration flow',
    }
  }

  const resetPassword = async () => {
    return {
      success: false,
      message: 'Local offline mode removed the password reset flow',
    }
  }

  const logout = async () => {
    const currentSession = getCurrentAuthSession()
    if (currentSession?.accessToken) {
      await fetch('/api/auth/logout', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${currentSession.accessToken}`,
        },
      }).catch(() => {
        /* ignore logout network errors */
      })
    }

    clearAuthSession()
    setUser(null)
    setToken(null)
    invalidateSystemConfig()
    navigate(ROUTES.login)
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        login,
        loginAdmin,
        register,
        resetPassword,
        logout,
        isLoading,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
