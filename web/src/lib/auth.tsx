import { createContext, useContext, useEffect, useState, type ReactNode } from "react"
import { api, clearSession, getSession, setSession } from "./api"

type Session = { token: string; email: string; orgId: string }

type AuthContextValue = {
  session: Session | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (orgName: string, email: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSessionState] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const s = getSession()
    if (s) setSessionState(s)
    setLoading(false)
  }, [])

  const login = async (email: string, password: string) => {
    const res = await api.login(email, password)
    setSession(res.token, res.email, res.org_id)
    setSessionState({ token: res.token, email: res.email, orgId: res.org_id })
  }

  const register = async (orgName: string, email: string, password: string) => {
    const res = await api.register(orgName, email, password)
    setSession(res.token, res.email, res.org_id)
    setSessionState({ token: res.token, email: res.email, orgId: res.org_id })
  }

  const logout = () => {
    clearSession()
    setSessionState(null)
  }

  return (
    <AuthContext.Provider value={{ session, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth must be used within AuthProvider")
  return ctx
}
