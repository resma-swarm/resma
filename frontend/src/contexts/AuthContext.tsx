import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from "react"
import { api, getTokens, setTokens, clearTokens } from "@/api/client"
import { deleteSSESession } from "@/hooks/sse-session-api"
import { sseSessionManager } from "@/hooks/sse-session-manager"

interface User {
  id: number
  username: string
  role: string
  name?: string
}

interface AuthContextType {
  initialized: boolean
  user: User | null
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  onboarding: (username: string, password: string) => Promise<void>
  logout: () => void
  updateUser: (patch: Partial<User>) => void
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [initialized, setInitialized] = useState(false)
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  const checkAuth = useCallback(async () => {
    setLoading(true)
    try {
      const status = await api.get<{ initialized: boolean }>("/auth/status")
      setInitialized(status.initialized)

      if (status.initialized) {
        const { access } = getTokens()
        if (access) {
          try {
            const me = await api.get<User>("/auth/me")
            setUser(me)
            // Renovar sessão SSE no page load — o cookie pode ter expirado
            // (TTL 10min) durante uma sessão longa sem refresh da página.
            sseSessionManager.ensureSession().catch(() => {
              // SSE session é opcional — fallback para polling
            })
          } catch {
            clearTokens()
            setUser(null)
          }
        }
      }
    } catch {
      setInitialized(false)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    checkAuth()
  }, [checkAuth])

  const login = useCallback(async (username: string, password: string) => {
    const res = await api.post<{ access_token: string; refresh_token: string }>("/auth/login", {
      username,
      password,
    })
    setTokens(res.access_token, res.refresh_token)
    // Criar sessão SSE (cookie HttpOnly) para EventSource.
    // O manager agenda o refresh proativo antes de expirar.
    sseSessionManager.ensureSession().catch(() => {
      // SSE session é opcional — fallback para polling
    })
    const me = await api.get<User>("/auth/me")
    setUser(me)
  }, [])

  const onboarding = useCallback(async (username: string, password: string) => {
    const res = await api.post<{ access_token: string; refresh_token: string }>("/auth/onboarding", {
      username,
      password,
    })
    setTokens(res.access_token, res.refresh_token)
    // Criar sessão SSE (cookie HttpOnly) para EventSource.
    sseSessionManager.ensureSession().catch(() => {
      // SSE session é opcional — fallback para polling
    })
    setInitialized(true)
    const me = await api.get<User>("/auth/me")
    setUser(me)
  }, [])

  const logout = useCallback(async () => {
    const { refresh } = getTokens()
    if (refresh) {
      try {
        await api.post("/auth/logout", { refresh_token: refresh })
      } catch {
        // ignore
      }
    }
    // Invalidar sessão SSE (cookie HttpOnly) e parar timer de refresh
    sseSessionManager.stop()
    try {
      await deleteSSESession()
    } catch {
      // ignore
    }
    clearTokens()
    setUser(null)
  }, [])

  const updateUser = useCallback((patch: Partial<User>) => {
    setUser((prev) => (prev ? { ...prev, ...patch } : prev))
  }, [])

  return (
    <AuthContext.Provider value={{ initialized, user, loading, login, onboarding, logout, updateUser }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth must be used within AuthProvider")
  return ctx
}
