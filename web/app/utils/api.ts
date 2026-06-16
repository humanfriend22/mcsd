import { useMessage } from 'naive-ui'
import type { Instance, Vendor, Snapshot, FileEntry } from '~/types/api'

const message = useMessage()

async function sendRequest<T>(
  url: string,
  method: 'GET' | 'POST' | 'PATCH' | 'DELETE' = 'GET',
  body?: Record<string, unknown>,
): Promise<T | null> {
  try {
    const init: RequestInit = { method }
    if (body !== undefined) {
      init.body = JSON.stringify(body)
      init.headers = { 'Content-Type': 'application/json' }
    }
    const res = await fetch(url, init)
    if (!res.ok) {
      const err = await res.json().catch(() => ({})) as any
      message.error(err?.error ?? `Request failed (${res.status})`)
      return null
    }
    if (res.status === 204) return {} as unknown as T
    return await res.json() as T
  } catch (err: any) {
    message.error(err?.message ?? 'Request failed')
    return null
  }
}

export const listVendors = () => sendRequest<Vendor[]>('/api/vendors')
export const listInstances = () => sendRequest<Instance[]>('/api/instances')
export const getInstance = (id: string) => sendRequest<Instance>(`/api/instances/${id}`)
export const patchInstance = (id: string, data: Record<string, unknown>) =>
  sendRequest<Instance>(`/api/instances/${id}`, 'PATCH', data)
export const startInstance = (id: string) => sendRequest(`/api/instances/${id}/start`, 'POST')
export const stopInstance = (id: string) => sendRequest(`/api/instances/${id}/stop`, 'POST')
export const enableInstance = (id: string) => sendRequest(`/api/instances/${id}/enable`, 'POST')
export const disableInstance = (id: string) => sendRequest(`/api/instances/${id}/disable`, 'POST')
export const deleteInstance = (id: string) => sendRequest(`/api/instances/${id}`, 'DELETE')
export const upgradeInstance = (id: string, vendor: string, version: string) =>
  sendRequest<Instance>(`/api/instances/${id}/upgrade`, 'POST', { vendor, version })
export const rconCommand = (id: string, command: string) =>
  sendRequest<{ response: string }>(`/api/instances/${id}/rcon`, 'POST', { command })
export const getVitals = () => sendRequest<Snapshot>('/api/vitals')
export const getPublicIp = () => sendRequest<{ ip: string }>('/api/public-ip')
export const listFiles = (id: string, path: string) =>
  sendRequest<FileEntry[]>(`/api/instances/${id}/files?path=${encodeURIComponent(path)}`)
export const deleteFile = (id: string, path: string) =>
  sendRequest(`/api/instances/${id}/files?path=${encodeURIComponent(path)}`, 'DELETE')

export async function readFile(id: string, path: string): Promise<string | null> {
  try {
    const res = await fetch(`/api/instances/${id}/files/content?path=${encodeURIComponent(path)}`)
    if (!res.ok) {
      const err = await res.json().catch(() => ({})) as any
      message.error(err?.error ?? `Request failed (${res.status})`)
      return null
    }
    return await res.text()
  } catch (err: any) {
    message.error(err?.message ?? 'Request failed')
    return null
  }
}

export async function writeFile(id: string, path: string, content: string): Promise<boolean> {
  try {
    const res = await fetch(`/api/instances/${id}/files/content?path=${encodeURIComponent(path)}`, {
      method: 'POST',
      body: content,
      headers: { 'Content-Type': 'text/plain; charset=utf-8' },
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({})) as any
      message.error(err?.error ?? `Request failed (${res.status})`)
      return false
    }
    return true
  } catch (err: any) {
    message.error(err?.message ?? 'Request failed')
    return false
  }
}

export async function uploadFile(id: string, path: string, file: File): Promise<boolean> {
  const form = new FormData()
  form.append('file', file)
  try {
    const res = await fetch(`/api/instances/${id}/files?path=${encodeURIComponent(path)}`, {
      method: 'POST',
      body: form,
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({})) as any
      message.error(err?.error ?? `Request failed (${res.status})`)
      return false
    }
    return true
  } catch (err: any) {
    message.error(err?.message ?? 'Request failed')
    return false
  }
}
