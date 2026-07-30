import { computed, inject, ref, watch } from 'vue'
import type { ComputedRef, InjectionKey } from 'vue'
import { useEventSource } from '@vueuse/core'
import { useRuntimeConfig } from '#app'
import { handleError, handleRequestError } from './error'

let requestPrefix: string | null = null
function getRequestPrefix(): string {
    if (requestPrefix === null) {
        requestPrefix = import.meta.dev ? `http://localhost:8080` : ''
    }
    return requestPrefix
}

export type InstanceState = 'active' | 'inactive' | 'activating' | 'deactivating' | 'failed'
export interface Instance {
    id: string
    name: string
    vendor: string
    version: string
    build: number
    binary: string
    java_args: string[]
    server_args: string[]
    memory: number
    ports: {
        game: number;
        rcon: number;
        rcon_password: string
    }
    state: InstanceState
    enabled: boolean
    active_since: string | null
    memory_used: number
}

export interface Vitals {
    host: {
        memory_total: number
        memory_used: number
        load_avg_1: number
        cpu_cores: number
        disk_total_gb: number
        disk_used_gb: number
    }
    budget: {
        total: number
        used: number
    }
}

export interface InitData {
    vendors: { name: string; versions: string[] }[]
    public_ip: string
    local_ip: string
    java_binaries: { path: string; version: string; vendor: string }[]
}

export interface FileEntry {
    name: string
    size: number
    is_dir: boolean
    modified: string
}

async function sendRequest(
    url: string,
    method: 'POST' | 'PATCH' | 'DELETE',
    body?: Record<string, unknown>,
): Promise<boolean> {
    try {
        const requestInit: RequestInit = { method }
        if (body) {
            requestInit.body = JSON.stringify(body)
            requestInit.headers = { 'Content-Type': 'application/json' }
        }
        const response = await fetch(getRequestPrefix() + url, requestInit)
        await handleRequestError(response)
        return true
    } catch (err: any) {
        handleError(err?.message ?? 'Request failed', err?.status)
        return false
    }
}

async function sendDataRequest<T>(
    url: string,
    method: 'GET' | 'POST' | 'PATCH' = 'GET',
    body?: Record<string, unknown>,
): Promise<T | null> {
    try {
        const requestInit: RequestInit = { method }
        if (body) {
            requestInit.body = JSON.stringify(body)
            requestInit.headers = { 'Content-Type': 'application/json' }
        }
        const response = await fetch(getRequestPrefix() + url, requestInit)
        await handleRequestError(response)
        return await response.json() as T
    } catch (err: any) {
        handleError(err?.message ?? 'Request failed', err?.status)
        return null
    }
}

// Instances Composable
export const instances = ref<Instance[] | null>(null)
let instancePollInterval: number | null = null

async function loadInstances() {
    instances.value = await sendDataRequest<Instance[]>('/api/instances')
}

export function useInstances() {
    if (!instancePollInterval) {
        loadInstances()
        instancePollInterval = setInterval(loadInstances, 2000)
    }
    return instances
}

export function useInstance(id: string) {
    return computed(() => instances.value?.find(i => i.id === id) || null);
};

// Instance provided by the instance detail page to its descendants. Pages only
// mount those descendants once the instance has loaded, so consumers can treat
// it as always present.
export const InstanceKey: InjectionKey<ComputedRef<Instance | null>> = Symbol('instance')

export function useCurrentInstance(): ComputedRef<Instance> {
    const instance = inject(InstanceKey)
    if (!instance) throw new Error('useCurrentInstance() must be used within an instance page')
    return instance as ComputedRef<Instance>
}

export async function loadInstance(id: string) {
    const data = await sendDataRequest<Instance>(`/api/instances/${id}`)
    if (!data) return
    if (!instances.value) instances.value = []
    const idx = instances.value.findIndex(i => i.id === id)
    if (idx >= 0) instances.value[idx] = data
    else instances.value.push(data)
}

// Initial Data Composable
const initData = ref<InitData | null>(null)

async function loadInit() {
    initData.value = await sendDataRequest<InitData>('/api/init')
}

export function useInit() {
    if (!initData.value) loadInit()
    return initData
}

// Vitals Composable
const vitals = ref<Vitals | null>(null);
let vitalsPollInterval: number | null = null;

async function loadVitals() {
    vitals.value = await sendDataRequest<Vitals>('/api/vitals');
}

export function useVitals() {
    if (!vitalsPollInterval) {
        loadVitals()
        vitalsPollInterval = setInterval(loadVitals, 5000)
    }
    return vitals;
}

// Logs
export function useInstanceLogs(id: string) {
    const logs = ref<string[]>([])
    const error = ref<string | null>(null)

    const { data: sseData, open: sseOpen, close: sseClose } =
        useEventSource(getRequestPrefix() + `/api/instances/${id}/logs`, [], { autoReconnect: true, immediate: false })

    watch(sseData, (line) => {
        if (!line) return

        if (line === 'latest.log not found') {
            error.value = 'latest.log not found'
            return
        }

        logs.value.push(line)
    })

    function open() {
        sseClose()
        logs.value = []
        error.value = null
        sseOpen()
    }

    function close() {
        sseClose()
        logs.value = []
        error.value = null
    }

    return { logs, error, open, close }
}

// HTTP Routes
export const patchInstance = (id: string, data: Record<string, unknown>) =>
    sendDataRequest<Instance>(`/api/instances/${id}`, 'PATCH', data)
export const startInstance = (id: string) => sendRequest(`/api/instances/${id}/start`, 'POST')
export const stopInstance = (id: string) => sendRequest(`/api/instances/${id}/stop`, 'POST')
export const enableInstance = (id: string) => sendRequest(`/api/instances/${id}/enable`, 'POST')
export const disableInstance = (id: string) => sendRequest(`/api/instances/${id}/disable`, 'POST')
export const deleteInstance = (id: string) => sendRequest(`/api/instances/${id}`, 'DELETE')
export const upgradeInstance = (id: string, vendor: string, version: string, build?: number) =>
    sendDataRequest<Instance>(`/api/instances/${id}/upgrade`, 'POST', { vendor, version, build })

export const rconCommand = (id: string, command: string) =>
    sendDataRequest<{ response: string }>(`/api/instances/${id}/rcon`, 'POST', { command })
export const listFiles = (id: string, path: string) =>
    sendDataRequest<FileEntry[]>(`/api/instances/${id}/files?path=${encodeURIComponent(path)}`)
export const deleteFile = (id: string, path: string) =>
    sendRequest(`/api/instances/${id}/files?path=${encodeURIComponent(path)}`, 'DELETE')

// File HTTP Routes
export async function readFile(id: string, path: string): Promise<string | null> {
    try {
        const response = await fetch(getRequestPrefix() + `/api/instances/${id}/files/content?path=${encodeURIComponent(path)}`)
        await handleRequestError(response);
        return await response.text()
    } catch (err: any) {
        handleError(err?.message ?? 'Read file request failed', err?.status)
        return null
    }
}

export async function writeFile(id: string, path: string, content: string): Promise<boolean> {
    try {
        const response = await fetch(getRequestPrefix() + `/api/instances/${id}/files/content?path=${encodeURIComponent(path)}`, {
            method: 'POST',
            body: content,
            headers: { 'Content-Type': 'text/plain; charset=utf-8' },
        })
        await handleRequestError(response);
        return true
    } catch (err: any) {
        handleError(err?.message ?? 'Write file request failed', err?.status)
        return false
    }
}

export async function uploadFile(id: string, path: string, file: File): Promise<boolean> {
    const form = new FormData()
    form.append('file', file)
    try {
        const response = await fetch(getRequestPrefix() + `/api/instances/${id}/files?path=${encodeURIComponent(path)}`, {
            method: 'POST',
            body: form,
        })
        await handleRequestError(response);
        return true
    } catch (err: any) {
        handleError(err?.message ?? 'Request failed', err?.status)
        return false
    }
}
