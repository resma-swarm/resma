const API_BASE = "/api"

function getTokens() {
  return {
    access: localStorage.getItem("resma_access_token"),
    refresh: localStorage.getItem("resma_refresh_token"),
  }
}

function setTokens(access: string, refresh: string) {
  localStorage.setItem("resma_access_token", access)
  localStorage.setItem("resma_refresh_token", refresh)
}

function clearTokens() {
  localStorage.removeItem("resma_access_token")
  localStorage.removeItem("resma_refresh_token")
}

let isRefreshing = false
let refreshPromise: Promise<void> | null = null

async function doRefresh(): Promise<void> {
  const { refresh } = getTokens()
  if (!refresh) throw new Error("No refresh token")

  const res = await fetch(`${API_BASE}/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refresh }),
  })

  if (!res.ok) {
    clearTokens()
    throw new Error("Refresh failed")
  }

  const data = await res.json()
  localStorage.setItem("resma_access_token", data.access_token)
}

async function fetchAPI<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const { access } = getTokens()
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...options.headers as Record<string, string>,
  }

  if (access) {
    headers["Authorization"] = `Bearer ${access}`
  }

  let res = await fetch(`${API_BASE}${path}`, { ...options, headers, credentials: "include" })

  if (res.status === 401 && access) {
    if (!isRefreshing) {
      isRefreshing = true
      refreshPromise = doRefresh().finally(() => {
        isRefreshing = false
      })
    }
    await refreshPromise
    const { access: newAccess } = getTokens()
    if (newAccess) {
      headers["Authorization"] = `Bearer ${newAccess}`
    }
    res = await fetch(`${API_BASE}${path}`, { ...options, headers, credentials: "include" })
  }

  if (!res.ok) {
    const error = await res.json().catch(() => ({ detail: res.statusText }))
    throw new Error(error.detail || `HTTP ${res.status}`)
  }

  return res.json()
}

export const api = {
  get: <T>(path: string) => fetchAPI<T>(path),
  post: <T>(path: string, body?: unknown) =>
    fetchAPI<T>(path, {
      method: "POST",
      body: body ? JSON.stringify(body) : undefined,
    }),
  put: <T>(path: string, body?: unknown) =>
    fetchAPI<T>(path, {
      method: "PUT",
      body: body ? JSON.stringify(body) : undefined,
    }),
  delete: <T>(path: string) =>
    fetchAPI<T>(path, {
      method: "DELETE",
    }),
  patch: <T>(path: string, body?: unknown) =>
    fetchAPI<T>(path, {
      method: "PATCH",
      body: body ? JSON.stringify(body) : undefined,
    }),
  blob: async (path: string) => {
    const { access } = getTokens()
    const headers: Record<string, string> = {}
    if (access) headers["Authorization"] = `Bearer ${access}`
    let res = await fetch(`${API_BASE}${path}`, { headers, credentials: "include" })
    if (res.status === 401 && access) {
      if (!isRefreshing) {
        isRefreshing = true
        refreshPromise = doRefresh().finally(() => { isRefreshing = false })
      }
      await refreshPromise
      const { access: newAccess } = getTokens()
      if (newAccess) headers["Authorization"] = `Bearer ${newAccess}`
      res = await fetch(`${API_BASE}${path}`, { headers, credentials: "include" })
    }
    if (!res.ok) {
      const error = await res.json().catch(() => ({ detail: res.statusText }))
      throw new Error(error.detail || `HTTP ${res.status}`)
    }
    return res.blob()
  },
}

export { getTokens, setTokens, clearTokens }
