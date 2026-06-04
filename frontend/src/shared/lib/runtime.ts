export function readArrayCache<T = any>(key: string): T[] {
  try {
    const raw = localStorage.getItem(key)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

export function writeJsonCache(key: string, value: unknown) {
  localStorage.setItem(key, JSON.stringify(value))
}

export function readNumberCache(key: string) {
  const raw = localStorage.getItem(key)
  if (!raw) return null
  const value = Number(raw)
  return Number.isFinite(value) ? value : null
}

export function persistNumberCache(key: string, value: number | null) {
  if (value === null || value === undefined) {
    localStorage.removeItem(key)
    return
  }
  localStorage.setItem(key, String(value))
}

export function debounce<T extends (...args: any[]) => void>(fn: T, wait = 300) {
  let timer: ReturnType<typeof setTimeout> | null = null
  return (...args: Parameters<T>) => {
    if (timer) window.clearTimeout(timer)
    timer = window.setTimeout(() => {
      timer = null
      fn(...args)
    }, wait)
  }
}

export function createLatestOnly() {
  const requestSeq: Record<string, number> = {}
  return async function latestOnly<T>(key: string, loader: () => Promise<T>, apply: (data: T) => void) {
    const seq = (requestSeq[key] || 0) + 1
    requestSeq[key] = seq
    const data = await loader()
    if (requestSeq[key] === seq) apply(data)
  }
}

export function createApiClient(apiBase: string, getToken: () => string, onUnauthorized?: () => void) {
  async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
    const response = await request(path, options)
    const contentType = response.headers.get('Content-Type') || ''
    if (contentType.includes('application/vnd') || contentType.includes('application/zip')) return response.blob() as T
    const body = await response.json().catch(() => ({}))
    if (!response.ok || body.code) {
      if (response.status === 401 && !path.startsWith('/auth/login')) onUnauthorized?.()
      const error = new Error(body.msg || `请求失败（${response.status}）`)
      ;(error as any).data = body.data
      ;(error as any).status = response.status
      ;(error as any).unauthorized = response.status === 401
      throw error
    }
    return body.data as T
  }

  async function downloadBlob(path: string, options: RequestInit = {}, fallbackName = 'download.xlsx') {
    const response = await request(path, options)
    if (!response.ok) {
      if (response.status === 401 && !path.startsWith('/auth/login')) onUnauthorized?.()
      const body = await response.json().catch(() => ({}))
      const error = new Error(body.msg || `下载失败（${response.status}）`)
      ;(error as any).status = response.status
      ;(error as any).unauthorized = response.status === 401
      throw error
    }
    const blob = await response.blob()
    const fileName = parseDownloadFileName(response.headers.get('Content-Disposition')) || fallbackName
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = fileName
    link.click()
    URL.revokeObjectURL(url)
  }

  function request(path: string, options: RequestInit = {}) {
    const headers = new Headers(options.headers)
    if (!(options.body instanceof FormData)) headers.set('Content-Type', 'application/json')
    const token = getToken()
    if (token) headers.set('Authorization', `Bearer ${token}`)
    return fetch(`${apiBase}${path}`, { ...options, headers })
  }

  return { api, downloadBlob }
}

function parseDownloadFileName(disposition: string | null) {
  if (!disposition) return ''
  const encoded = disposition.match(/filename\*=UTF-8''([^;]+)/i)?.[1]
  if (encoded) return decodeURIComponent(encoded)
  return disposition.match(/filename="?([^"]+)"?/i)?.[1] || ''
}
