import { computed, ref } from 'vue'
import { createDiscreteApi, darkTheme } from 'naive-ui'
import { useEventSource } from '@vueuse/core'

const { message } = createDiscreteApi(['message'], { configProviderProps: { theme: darkTheme } })

const REQUEST_PREFIX = import.meta.dev ? 'http://raspberrypi.local:8080' : ''

const instanceConfigs = ref<InstanceConfig[]>([])
const instanceStates = ref<Record<string, InstanceRunState>>({})
const instancesLoaded = ref(false)
const initData = ref<InitData | null>(null)
const initLoaded = ref(false)
const vitalsSnapshot = ref<Snapshot | null>(null)

let instancePollHandle: ReturnType<typeof setInterval> | null = null
let vitalsPollHandle: ReturnType<typeof setInterval> | null = null
let instanceLoadPromise: Promise<void> | null = null
let stateLoadPromise: Promise<void> | null = null
let initLoadPromise: Promise<void> | null = null
let vitalsLoadPromise: Promise<void> | null = null

export interface InstanceConfig {
    id: string
    name: string
    vendor: string
    version?: string
    java_args?: string[]
    server_args?: string[]
    memory: number
    ports?: { game: number; rcon: number }
}

export interface InstanceRunState {
    state: 'active' | 'inactive' | 'activating' | 'deactivating' | 'failed'
    enabled: boolean
    active_since?: string
    memory_used?: number
}

export type Instance = InstanceConfig & InstanceRunState

export interface HostVitals {
    memory_total: number
    memory_used: number
    load_avg_1: number
    cpu_cores: number
    disk_total_gb: number
    disk_used_gb: number
}

export interface BudgetVitals {
    total: number
    used: number
}

export interface Snapshot {
    host: HostVitals
    budget: BudgetVitals
}

export interface Vendor {
    name: string
    versions: string[]
}

export interface InitData {
    vendors: Vendor[]
    public_ip: string
}

export interface FileEntry {
    name: string
    size: number
    is_dir: boolean
    modified: string
}

const instances = computed<Instance[]>(() =>
    instanceConfigs.value.map(c => ({
        ...c,
        ...(instanceStates.value[c.id] ?? { state: 'inactive' as const, enabled: false }),
    }))
)

export async function loadInstanceData() {
    if (instanceLoadPromise) return instanceLoadPromise
    instanceLoadPromise = (async () => {
        const data = await listInstances()
        if (data) {
            instanceConfigs.value = data
            instancesLoaded.value = true
        }
    })().finally(() => {
        instanceLoadPromise = null
    })
    return instanceLoadPromise
}

export async function loadInstanceStates() {
    if (stateLoadPromise) return stateLoadPromise
    stateLoadPromise = (async () => {
        const data = await getAllStates()
        if (data) instanceStates.value = data
    })().finally(() => {
        stateLoadPromise = null
    })
    return stateLoadPromise
}

export function setInstanceConfig(id: string, config: InstanceConfig) {
    const idx = instanceConfigs.value.findIndex(c => c.id === id)
    if (idx >= 0) instanceConfigs.value[idx] = config
}

export async function loadInit() {
    if (initLoadPromise) return initLoadPromise
    initLoadPromise = (async () => {
        const data = await getInit()
        if (data) {
            initData.value = data
            initLoaded.value = true
        }
    })().finally(() => {
        initLoadPromise = null
    })
    return initLoadPromise
}

function ensureInstancePolling() {
    void loadInstanceData()
    void loadInstanceStates()
    if (instancePollHandle) return
    instancePollHandle = setInterval(() => {
        void loadInstanceStates()
    }, 1000)
}

export function useInstances() {
    ensureInstancePolling()
    return { instances, loaded: instancesLoaded, refresh: loadInstanceData, refreshStates: loadInstanceStates }
}

export function useInit() {
    void loadInit()
    return {
        vendors: computed(() => initData.value?.vendors ?? []),
        publicIp: computed(() => initData.value?.public_ip ?? null),
        loaded: initLoaded,
        refresh: loadInit,
    }
}

export async function loadVitals() {
    if (vitalsLoadPromise) return vitalsLoadPromise
    vitalsLoadPromise = (async () => {
        const data = await getVitals()
        if (data) vitalsSnapshot.value = data
    })().finally(() => {
        vitalsLoadPromise = null
    })
    return vitalsLoadPromise
}

function ensureVitalsPolling() {
    void loadVitals()
    if (vitalsPollHandle) return
    vitalsPollHandle = setInterval(() => {
        void loadVitals()
    }, 5000)
}

export function useVitals() {
    ensureVitalsPolling()
    return { snapshot: vitalsSnapshot, refresh: loadVitals }
}

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
        const res = await fetch(REQUEST_PREFIX + url, init)
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

export const getInit = () => sendRequest<InitData>('/api/init')
export const listInstances = () => sendRequest<InstanceConfig[]>('/api/instances')
export const getInstance = (id: string) => sendRequest<InstanceConfig>(`/api/instances/${id}`)
export const getAllStates = () => sendRequest<Record<string, InstanceRunState>>('/api/state')
export const getVitals = () => sendRequest<Snapshot>('/api/vitals')
export const patchInstance = (id: string, data: Record<string, unknown>) =>
    sendRequest<InstanceConfig>(`/api/instances/${id}`, 'PATCH', data)
export const startInstance = (id: string) => sendRequest(`/api/instances/${id}/start`, 'POST')
export const stopInstance = (id: string) => sendRequest(`/api/instances/${id}/stop`, 'POST')
export const enableInstance = (id: string) => sendRequest(`/api/instances/${id}/enable`, 'POST')
export const disableInstance = (id: string) => sendRequest(`/api/instances/${id}/disable`, 'POST')
export const deleteInstance = (id: string) => sendRequest(`/api/instances/${id}`, 'DELETE')
export const upgradeInstance = (id: string, vendor: string, version: string) =>
    sendRequest<InstanceConfig>(`/api/instances/${id}/upgrade`, 'POST', { vendor, version })
export const rconCommand = (id: string, command: string) =>
    sendRequest<{ response: string }>(`/api/instances/${id}/rcon`, 'POST', { command })
export const useInstanceLogs = (id: string) =>
    useEventSource(REQUEST_PREFIX + `/api/instances/${id}/logs`, [], { autoReconnect: true })
export const listFiles = (id: string, path: string) =>
    sendRequest<FileEntry[]>(`/api/instances/${id}/files?path=${encodeURIComponent(path)}`)
export const deleteFile = (id: string, path: string) =>
    sendRequest(`/api/instances/${id}/files?path=${encodeURIComponent(path)}`, 'DELETE')

export async function readFile(id: string, path: string): Promise<string | null> {
    try {
        const res = await fetch(REQUEST_PREFIX + `/api/instances/${id}/files/content?path=${encodeURIComponent(path)}`)
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
        const res = await fetch(REQUEST_PREFIX + `/api/instances/${id}/files/content?path=${encodeURIComponent(path)}`, {
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
        const res = await fetch(REQUEST_PREFIX + `/api/instances/${id}/files?path=${encodeURIComponent(path)}`, {
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