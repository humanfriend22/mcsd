export interface Instance {
  id: string
  name: string
  vendor: string
  version?: string
  java_args?: string[]
  server_args?: string[]
  memory: number
  ports?: { game: number; rcon: number }
  state: 'active' | 'inactive' | 'activating' | 'deactivating' | 'failed'
  enabled: boolean
  uptime_seconds?: number
  memory_used?: number
}

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

export interface FileEntry {
  name: string
  size: number
  is_dir: boolean
  modified: string
}