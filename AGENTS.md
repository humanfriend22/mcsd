# mcsd — Minecraft Server Daemon

A self-hosted Minecraft server manager for Linux (primarily Raspberry Pi) with deep systemd integration. Manages multiple server instances via systemd template units, enforces memory budgets, and provides both a REST API and a web UI.

## Architecture Overview

```
mcsd/
├── src/                    # Go backend
│   ├── main.go             # Entry point: init D-Bus → CLI dispatch
│   ├── core/               # Business logic: instances, systemd, RCON, config
│   ├── api/                # HTTP REST API handlers
│   ├── cli/                # Kong CLI commands (interactive + daemon)
│   ├── vendors/            # Pluggable server type registry
│   ├── helpers/            # ID validation, atomic file writes
│   └── web/                # go:embed for Nuxt SPA (build tag: embed_web)
├── web/                    # Nuxt 4 SPA (Vue 3 + NaiveUI + Tailwind)
│   └── app/
│       ├── services/api.ts # Centralized API client + reactive state + polling
│       ├── components/     # UI components (ServerCard, FileBrowser, etc.)
│       ├── pages/          # Routes: index (dashboard), instances/[id]
│       └── composables/    # useInstance wrapper
├── test/                   # Standalone systemd D-Bus test tool
├── compose.yml             # Docker dev environment (Debian + systemd)
├── Dockerfile              # Multi-stage: jrei/systemd-debian + OpenJDK 25
└── mise.toml               # Task runner: build, embed, deploy pipeline
```

## Go Backend (`src/`)

### Module & Dependencies

- **Module**: `mcsd` (Go 1.26.3)
- **CLI framework**: `github.com/alecthomas/kong`
- **Interactive forms**: `github.com/charmbracelet/huh`
- **TUI (console)**: `github.com/gdamore/tcell/v2`
- **systemd D-Bus**: `github.com/coreos/go-systemd/v22`
- **Raw D-Bus**: `github.com/godbus/dbus/v5`

### Package Responsibilities

#### `core/` — Business Logic

| File | Responsibility |
|------|---------------|
| `core.go` | Global init/deinit, systemd service file embedding, `EnsureReady()` |
| `instance.go` | `Instance` struct, `Launch()` (ExecStart), `resolveArgs()`, memory budget checks |
| `instance_config.go` | `InstanceConfig` JSON schema, `Load/WriteInstanceConfig()`, `ValidateIDUnique()` |
| `instance_model.go` | `LoadInstance()`, `ListInstances()`, CRUD methods, RCON, enable/disable |
| `config.go` | Global `Config` (memory budget, port), `/proc/meminfo` reader |
| `systemd.go` | `sdManager` — D-Bus wrapper for Start/Stop/Enable/Disable/Status/List/ActiveStats |
| `ports.go` | `Ports` struct, `ReadPorts()` / `WriteServerProperties()` (server.properties) |
| `rcon.go` | Full RCON protocol client (auth, send, recv) |
| `java.go` | `DiscoverJavaBinaries()` — scans PATH/JAVA_HOME, parses `java -version` |
| `errors.go` | `ValidationError`, `NotFoundError` (used for HTTP status mapping) |
| `services/` | Embedded systemd unit files |

**Key patterns:**
- `InstanceConfig` is a pure JSON data struct — no methods except `Validate()`
- `Instance` embeds `*InstanceConfig` and adds runtime state (ports, state, uptime, memory)
- `LoadInstance(id)` assembles an `Instance` from config.json + server.properties + systemd D-Bus
- All file writes use `WriteAtomic()` (write to `.tmp`, then rename)
- `SDManager` is a package-level singleton initialized once per process via `InitSDManager()`

#### `api/` — HTTP REST API

| File | Endpoint(s) |
|------|------------|
| `api.go` | Route registration, CORS middleware, JSON helpers |
| `instances.go` | CRUD: `GET/POST/PATCH/DELETE /api/instances[/{id}]`, `POST .../start|stop|enable|disable|upgrade|rcon` |
| `state.go` | `GET /api/state` — real-time run state for all instances (includes pre-start readiness check) |
| `snapshot.go` | `GET /api/vitals` — host vitals (RAM, CPU, disk) + memory budget usage |
| `vendors.go` | `GET /api/init` — vendors list, public IP, discovered Java binaries |
| `files.go` | `GET/POST/DELETE /api/instances/{id}/files` — file browser (list, read, write, upload, delete) |
| `logs.go` | `GET /api/instances/{id}/logs` — SSE stream via `journalctl -f` |
| `ready.go` | `GET /api/instances/{id}/ready` — readiness check (ports + memory budget) |
| `rcon_cache.go` | In-memory RCON connection cache with 30s idle TTL |

**Error mapping:**
- `ValidationError` → 400 Bad Request
- `NotFoundError` → 404 Not Found
- Everything else → 500 Internal Server Error

#### `cli/` — Kong CLI

```
mcsd init [--memory MB]       # Set up systemd services + config
mcsd deinit [-f]              # Tear down everything

mcsd instance create          # Interactive wizard (huh forms)
mcsd instance start <id>
mcsd instance stop <id>
mcsd instance restart <id>
mcsd instance status [id]     # Proxies to `systemctl status`
mcsd instance list
mcsd instance delete <id> [-f]
mcsd instance enable <id>
mcsd instance disable <id>
mcsd instance rcon <id> <cmd>
mcsd instance edit <id>       # Opens config.json in $EDITOR
mcsd instance console <id>    # Full TUI: live logs + RCON input (tcell)

mcsd start                    # Start mcsd.service (daemon)
mcsd stop
mcsd enable
mcsd disable
mcsd status

mcsd systemd launch <id>      # ExecStart= handler (called by systemd)
mcsd systemd stop <id>        # ExecStop= handler (RCON stop)
mcsd systemd serve            # HTTP daemon entry point
```

- All commands except `init`/`deinit` require `EnsureReady()` to pass
- `BeforeApply` hook in Kong handles this check

#### `vendors/` — Server Type Registry

| File | Vendor |
|------|--------|
| `vendors.go` | `Vendor` interface, `All` slice, `Get()`, `IsValid()`, `Executable()` |
| `fabric.go` | Fabric — fetches versions from `meta.fabricmc.net`, builds download URL |
| `http.go` | `getJSON()`, `DownloadFile()` utilities |

Currently only Fabric is registered. Bedrock is referenced but not implemented in `All`.

**Adding a vendor**: Implement the `Vendor` interface (Name, Versions, DownloadURL) and append to `All`.

#### `helpers/` — Utilities

- `ValidateID(id)` — regex `^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`, blocks `..` and `/`
- `WriteAtomic(path, data, mode)` — write to `.tmp` then `os.Rename`

### Systemd Integration

**Unit files** (embedded via `//go:embed`):

| Unit | Purpose |
|------|---------|
| `mcsd.service` | HTTP daemon — `ExecStart=/usr/local/bin/mcsd systemd serve` |
| `mcsd-instance@.service` | Per-instance — `ExecStart=/usr/local/bin/mcsd systemd launch %i` |

**Instance template unit features:**
- `MemoryAccounting=yes` — tracks per-instance memory via D-Bus
- `LogNamespace=mcsd-%i` — isolated journal per instance
- Sandboxing: `NoNewPrivileges`, `ProtectSystem=strict`, `PrivateTmp`, `ReadWritePaths` limited to instance dir
- `TimeoutStopSec=120` with `SendSIGKILL=yes` — graceful shutdown via RCON `stop` command
- `StartLimitBurst=3` / `StartLimitIntervalSec=60` — crash loop protection

**D-Bus operations** (via `sdManager`):
- Start/Stop/Restart/Enable/Disable/ResetFailed/Reload
- Status (ActiveState, SubState)
- IsEnabled (via raw D-Bus `GetUnitFileState`)
- ActiveStats (ActiveEnterTimestamp + MemoryCurrent)
- List (by glob pattern)

### Data Model

```go
// Persisted to /srv/mcsd/instances/<id>/config.json
type InstanceConfig struct {
    ID         string   `json:"id"`
    Name       string   `json:"name"`
    Vendor     string   `json:"vendor"`
    Version    string   `json:"version"`
    Binary     string   `json:"binary"`        // java binary path (empty = "java")
    JavaArgs   []string `json:"java_args"`
    ServerArgs []string `json:"server_args"`
    Memory     int      `json:"memory"`         // MB
}

// Full runtime representation
type Instance struct {
    *InstanceConfig
    Ports         Ports  `json:"ports"`           // from server.properties
    State         string `json:"state"`           // from systemd
    Enabled       bool   `json:"enabled"`         // from systemd
    UptimeSeconds int    `json:"uptime_seconds"`
    MemoryUsed    int    `json:"memory_used"`     // MB from D-Bus
}

type Ports struct {
    Game         int    `json:"game"`
    RCON         int    `json:"rcon"`
    RCONPassword string `json:"rcon_password"`
}

// Global config at /srv/mcsd/config.json
type Config struct {
    MemoryBudget int `json:"memory_budget"`  // total MB across all instances
    Port         int `json:"port"`           // daemon HTTP port (default 8080)
}
```

**Filesystem layout per instance** (`/srv/mcsd/instances/<id>/`):
```
config.json          # InstanceConfig
server.properties    # game/rcon ports (written by mcsd)
eula.txt             # eula=true
server.jar           # Fabric JAR (downloaded)
```

### Memory Budget System

- Global `Config.MemoryBudget` sets total RAM cap
- `ReservedMemory(excludeID)` sums `Memory` from all **active** instances
- `EnabledMemory(excludeID)` sums `Memory` from all **enabled** instances
- Checked before: `Start()`, `Enable()`, `Create()`
- On startup, budget is capped to `system RAM - 512 MB` if config exceeds physical RAM

## Web Frontend (`web/`)

### Stack

- **Nuxt 4** (SPA mode, `ssr: false`)
- **Vue 3** Composition API
- **NaiveUI** component library (dark theme)
- **Tailwind CSS**
- **@vueuse/core** for `useEventSource`
- **Bun** as package manager / build tool

### Structure

```
web/app/
├── app.vue                 # Root: sidebar navigation + NuxtPage outlet
├── services/
│   ├── api.ts              # Single source of truth: types, fetch, state, polling, composables
│   └── error.ts            # NaiveUI discrete message API for error toasts
├── components/
│   ├── ServerCard.vue      # Dashboard grid card
│   ├── InstanceHeader.vue  # Detail page header
│   ├── InstanceOverviewTab.vue  # Controls, stats, logs, RCON
│   ├── InstanceSettingsTab.vue  # Edit config, upgrade, delete
│   ├── InstanceStatCard.vue     # Reusable stat card with progress bar
│   ├── FileBrowser.vue          # REST file browser
│   ├── LogViewer.vue            # Log display component
│   └── StatusDot.vue            # Colored state indicator
├── pages/
│   ├── index.vue           # Dashboard: host vitals + instance grid
│   └── instances/[id].vue  # Instance detail page
├── composables/
│   └── useInstance.ts      # Single-instance wrapper around useInstances()
└── utils/
    └── format.ts           # formatUptime(), formatMemoryMB()
```

### API Layer (`services/api.ts`)

**All backend communication goes through this single file.** It acts as both API client and state store (module-level `ref()`s shared via ES module singleton semantics — no Pinia/Vuex).

**Types exported:**
- `Instance` — merged config + runtime state
- `InstanceState` — `'active' | 'inactive' | 'activating' | 'deactivating' | 'failed'`
- `Vitals` — host system metrics + memory budget
- `InitData` — vendors, public IP, Java binaries
- `FileEntry` — file browser entry

**Polling:**
- Instance states: `GET /api/state` every 1s (via `useInstances()`)
- Host vitals: `GET /api/vitals` every 5s (via `useVitals()`)
- Init data: `GET /api/init` once on load (via `useInit()`)
- Live logs: SSE via `EventSource` (via `useInstanceLogs(id)`)

**Request infrastructure:**
- `sendRequest()` — for mutations (POST/PATCH/DELETE), returns boolean
- `sendDataRequest<T>()` — for reads + JSON mutations, returns typed data
- In dev mode, requests proxy to `http://raspberrypi.local:8080`

## API Reference

### Instance Endpoints

| Method | Path | Handler | Response |
|--------|------|---------|----------|
| `GET` | `/api/instances` | `listInstances` | `Instance[]` |
| `GET` | `/api/instances/{id}` | `getInstance` | `Instance` |
| `POST` | `/api/instances` | `createInstance` | `Instance` (201) |
| `PATCH` | `/api/instances/{id}` | `patchInstance` | `Instance` |
| `DELETE` | `/api/instances/{id}` | `deleteInstance` | 204 |

### Action Endpoints

| Method | Path | Handler | Response |
|--------|------|---------|----------|
| `POST` | `/api/instances/{id}/start` | `startInstance` | 204 |
| `POST` | `/api/instances/{id}/stop` | `stopInstance` | 204 |
| `POST` | `/api/instances/{id}/enable` | `enableInstance` | 204 |
| `POST` | `/api/instances/{id}/disable` | `disableInstance` | 204 |
| `POST` | `/api/instances/{id}/upgrade` | `upgradeInstance` | `Instance` |
| `POST` | `/api/instances/{id}/rcon` | `rconInstance` | `{ response: string }` |
| `GET` | `/api/instances/{id}/ready` | `checkReady` | `{ ready: bool, error: string }` |

### System Endpoints

| Method | Path | Handler | Response |
|--------|------|---------|----------|
| `GET` | `/api/state` | `getAllStates` | `Record<string, InstanceRunState>` |
| `GET` | `/api/vitals` | `getVitals` | `Snapshot` |
| `GET` | `/api/init` | `listVendors` | `InitData` |

### File Endpoints

| Method | Path | Handler | Response |
|--------|------|---------|----------|
| `GET` | `/api/instances/{id}/files` | `listFiles` | `FileEntry[]` |
| `GET` | `/api/instances/{id}/files/content` | `readFile` | Raw binary |
| `POST` | `/api/instances/{id}/files/content` | `writeFile` | 204 |
| `DELETE` | `/api/instances/{id}/files` | `deleteFile` | 204 |
| `POST` | `/api/instances/{id}/files` | `uploadFile` | `{ path: string }` (201) |

### Streaming Endpoints

| Method | Path | Handler | Response |
|--------|------|---------|----------|
| `GET` | `/api/instances/{id}/logs` | `streamLogs` | SSE (`text/event-stream`) |

### Response Shapes

```typescript
// GET /api/state
type InstanceRunState = {
    state: 'active' | 'inactive' | 'failed' | 'activating' | 'deactivating'
    enabled: boolean
    active_since?: string    // ISO timestamp (only when active)
    memory_used?: number     // MB (only when active)
    start_check?: string     // error message (only when NOT active)
}

// GET /api/vitals
type Snapshot = {
    host: {
        memory_total: number   // MB
        memory_used: number    // MB
        load_avg_1: number
        cpu_cores: number
        disk_total_gb: number
        disk_used_gb: number
    }
    budget: {
        total: number          // from config.json
        used: number           // sum of active instances
    }
}

// GET /api/init
type InitData = {
    vendors: { name: string; versions: string[] }[]
    public_ip: string
    local_ip: string
    java_binaries: { path: string; version: string; vendor: string }[]
}
```

## Key Constants & Paths

| Constant | Value |
|----------|-------|
| `DefaultBasePath` | `/srv/mcsd/instances` |
| `GlobalConfigPath` | `/srv/mcsd/config.json` |
| `DaemonServicePath` | `/etc/systemd/system/mcsd.service` |
| `DaemonServiceTemplatePath` | `/etc/systemd/system/mcsd-instance@.service` |
| Default HTTP port | `8080` |
| Min instance memory | `512 MB` |
| Min system reserve | `512 MB` |
| RCON idle TTL | `30 seconds` |
| Instance ID regex | `^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$` |

## Build Pipeline

```bash
# Development
cd web && bun run dev          # Nuxt dev server

# Production build
mise run build-web             # bun run generate → web/dist
mise run embed                 # cp web/.output/public → src/web/dist
mise run build                 # go build -tags embed_web → mcsd binary
mise run deploy                # scp to Pi
mise run deploy-hot            # stop → scp → restart
```

The `embed_web` build tag controls whether the Nuxt SPA is embedded in the binary. Without it, the web package is a no-op stub.

## Development Notes

### Environment

- **Target platform**: `linux/arm64` (Raspberry Pi)
- **Dev host**: macOS (cross-compilation via `GOOS=linux GOARCH=arm64`)
- **Remote**: `pi@192.168.64.11` (configurable via `.env` + `mise.toml`)
- **Docker**: `compose.yml` runs a privileged Debian container with systemd for local testing

### Conventions

- All file writes are atomic (`.tmp` + `rename`)
- Instance IDs are validated with regex, no path separators allowed
- `InstanceConfig` has no lifecycle methods — pure data struct
- `Instance` is assembled at runtime from multiple data sources
- Error types: `ValidationError` (400), `NotFoundError` (404), `error` (500)
- No comments in Go code unless explicitly requested
- No optional/omitempty fields in API responses — fully declarative JSON

### Known TODOs

- REPL-like CLI interface for instance commands (avoids retyping instance ID)
- Bedrock vendor support (referenced but not in `vendors.All`)
- Instance caching for D-Bus-heavy operations (noted in `instance-struct-refactor.txt`)

### Testing

- `test/main.go` — standalone D-Bus unit property inspector (not a test suite)
- No formal test suite exists; testing is manual via CLI + web UI
- The `test/main` binary is pre-built for quick D-Bus debugging

## Git

- `.gitignore`: ignores `.env`, `mcsd` binary, `src/web/dist/**`
- The `mcsd` binary in the repo root is the compiled output (deployed to Pi)
