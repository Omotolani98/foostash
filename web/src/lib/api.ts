export class ApiError extends Error {
  code: string
  status: number
  constructor(code: string, message: string, status: number) {
    super(message)
    this.code = code
    this.status = status
  }
}

const TOKEN_KEY = "foostash.token"
const EMAIL_KEY = "foostash.email"
const ORG_KEY = "foostash.org"

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setSession(token: string, email: string, orgId: string) {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(EMAIL_KEY, email)
  localStorage.setItem(ORG_KEY, orgId)
}

export function clearSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(EMAIL_KEY)
  localStorage.removeItem(ORG_KEY)
}

export function getSession() {
  const token = getToken()
  if (!token) return null
  return {
    token,
    email: localStorage.getItem(EMAIL_KEY) ?? "",
    orgId: localStorage.getItem(ORG_KEY) ?? "",
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers["Content-Type"] = "application/json"
  const token = getToken()
  if (token) headers["Authorization"] = `Bearer ${token}`

  const res = await fetch(path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  const text = await res.text()
  const data = text ? JSON.parse(text) : null

  if (!res.ok) {
    const err = data?.error
    throw new ApiError(
      err?.code ?? "error",
      err?.message ?? `HTTP ${res.status}`,
      res.status
    )
  }
  return data as T
}

export type AuthResponse = {
  token: string
  user_id: string
  org_id: string
  email: string
}

export type Project = { id: string; slug: string; name: string }
export type Environment = { id: string; slug: string }
export type PullResponse = {
  project: string
  environment: string
  secrets: Record<string, string>
  version: number
  pulled_at: string
}
export type DiffResponse = {
  project: string
  left: string
  right: string
  only_left: string[]
  only_right: string[]
  different_values: string[]
  identical: string[]
}

export const api = {
  register: (org_name: string, email: string, password: string) =>
    request<AuthResponse>("POST", "/v1/auth/register", { org_name, email, password }),
  login: (email: string, password: string) =>
    request<AuthResponse>("POST", "/v1/auth/login", { email, password }),

  listProjects: () =>
    request<{ projects: Project[] }>("GET", "/v1/projects"),
  createProject: (slug: string, name: string) =>
    request<Project>("POST", "/v1/projects", { slug, name }),
  deleteProject: (slug: string) =>
    request<void>("DELETE", `/v1/projects/${slug}`),

  listEnvs: (slug: string) =>
    request<{ environments: Environment[] }>("GET", `/v1/projects/${slug}/envs`),
  createEnv: (slug: string, envSlug: string) =>
    request<Environment>("POST", `/v1/projects/${slug}/envs`, { slug: envSlug }),
  deleteEnv: (slug: string, env: string) =>
    request<void>("DELETE", `/v1/projects/${slug}/envs/${env}`),

  pullSecrets: (slug: string, env: string) =>
    request<PullResponse>("GET", `/v1/projects/${slug}/envs/${env}/secrets`),
  setSecrets: (slug: string, env: string, secrets: Record<string, string>) =>
    request<{ set: string[]; version: number }>("POST", `/v1/projects/${slug}/envs/${env}/secrets`, { secrets }),
  deleteSecret: (slug: string, env: string, key: string) =>
    request<void>("DELETE", `/v1/projects/${slug}/envs/${env}/secrets/${encodeURIComponent(key)}`),
  diff: (slug: string, left: string, right: string) =>
    request<DiffResponse>("GET", `/v1/projects/${slug}/envs/${left}/diff/${right}`),
}
