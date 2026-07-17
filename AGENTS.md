# AGENTS.md

Authoritative reference for AI agents working in this repo. Supersedes `PLAN.md` and `CLAUDE.md` where they conflict — both are stale.

---

## Build & Deploy

```bash
# Cross-compile for Raspberry Pi (arm64 Linux)
make build          # GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o mcsd .

# Build + rsync to Pi + install
make deploy         # default PI=pi@raspberrypi.local
PI=user@host make deploy
                    # Uses ssh -t (required on Raspberry Pi OS Bookworm+ — passwordless sudo removed)

make clean
```

No tests. Validation = smoke test on Pi.

Run locally (skips systemd code paths):
```bash
go build -o mcsd . && ./mcsd --help
```

---

## Architecture

Single static binary, two runtime roles:

1. **CLI** (`mcsd <command>`) — stateless; reads filesystem + queries systemd D-Bus.
2. **HTTP daemon** (`mcsd systemd serve`) — REST API for instance management, SSE log streaming, file browser. Installed as `mcsd.service`.

The hidden `systemd` subcommand group is called only by systemd as `ExecStart=`/`ExecStop=`. Users never call it directly.

### Package responsibilities

| Package | Role |
|---------|------|
| `cli/` | Kong command structs + interactive prompts. Zero business logic — delegates entirely to `core` and `api`. |
| `core/` | All server management: config I/O, systemd D-Bus, RCON, instance lifecycle, launch. |
| `api/` | HTTP handlers, SSE log streaming, file browser, vendor discovery. |
| `vendors/` | Pluggable server type registry. Adding a vendor = implement the `Vendor` interface. |

### Execution model

```
mcsd start <id>
  → D-Bus StartUnit
  → systemd ExecStart = mcsd systemd launch <id>
      core.Launch(id):
        assert INVOCATION_ID set
        LoadInstance(id)
        resolveArgv()
        os.Chdir(InstanceDir(id))
        syscall.Exec()   ← process replaced by JVM/binary

mcsd stop <id>
  → D-Bus StopUnit
  → systemd ExecStop = mcsd systemd stop <id>
      instance.RCON("stop")   ← graceful shutdown
      TimeoutStopSec=120 → SIGKILL backstop
```

`mcsd create` provisions and registers the unit but does **not** enable it. Use `mcsd enable <id>` for start-on-boot.

### Storage layout (runtime, on the Pi)

```
/srv/mcsd/config.json                                          # global: memory_budget, port
/srv/mcsd/instances/<id>/config.json                           # instance definition (InstanceConfig)
/srv/mcsd/instances/<id>/server.properties                     # written once at create; source of truth for ports
/etc/systemd/system/mcsd.service                               # daemon unit (embedded in binary)
/etc/systemd/system/mcsd-server@.service                       # instance template unit (embedded in binary)
/etc/systemd/system/mcsd-server@<id>.service.d/override.conf   # MemoryMax= per instance
```

---

## Key invariants

- `core.CheckInit()` called at the top of every CLI command `Run()` except `init`/`deinit`.
- All config/instance writes go through `writeAtomic()` (write to `.tmp`, then `os.Rename`) — never `os.WriteFile` directly on live paths.
- `InstanceDir(id)` is always `DefaultBasePath + "/" + id`. No per-struct base path field.
- `server.properties` is written once at create (`initServerProperties`), then owned by the user. mcsd reads it; never overwrites. Source of truth for game port, RCON port, and RCON password. (ADR-0001)
- `core.Launch()` aborts if `INVOCATION_ID` env var is absent.
- `CheckBudget()` is called inside `instance.Start()` before the D-Bus call.
- `LoadInstance()` wraps `os.ErrNotExist` with `%w` — the API layer uses `errors.Is(err, os.ErrNotExist)` to return 404.
- The systemd service files are embedded via `//go:embed` in `core/core.go`. Editing `core/services/*.service` changes what `mcsd init` installs.
- Memory enforcement = systemd `MemoryMax=` drop-in written by `instance.WriteDropIn()`.

---

## Package reference

### `core/`

**`core.go`** — constants, embedded service files, init/deinit.

```go
const DefaultBasePath  = "/srv/mcsd/instances"
const GlobalConfigPath = "/srv/mcsd/config.json"
const DaemonServicePath         = "/etc/systemd/system/mcsd.service"
const DaemonServiceTemplatePath = "/etc/systemd/system/mcsd-server@.service"

Init(memoryBudget int) error          // write config + service files
InitSystemd(client *SystemdClient) error
DeInit(client *SystemdClient) error   // full teardown
EnsureReady() error                   // verify installation + clean orphans
```

**`config.go`**

```go
type Config struct {
    MemoryBudget int `json:"memory_budget"`
    Port         int `json:"port,omitempty"` // daemon HTTP port; 0 = default 8080
}

TotalSystemMemory() (int, error)   // reads /proc/meminfo
LoadConfig() (*Config, error)
WriteConfig(cfg *Config) error
```

**`instance.go`** — all instance logic in one file.

```go
type InstanceConfig struct {
    ID         string   `json:"id"`
    Name       string   `json:"name"`
    Type       string   `json:"type"`
    Version    string   `json:"mc_version,omitempty"`
    Executable string   `json:"executable"`
    JavaArgs   []string `json:"java_args,omitempty"`
    ServerArgs []string `json:"server_args,omitempty"`
    Memory     int      `json:"memory"`
}

// Ports is never persisted — always read from server.properties at runtime.
type Ports struct {
    Game         int
    RCON         int
    RCONPassword string  // never exposed in API responses
}

type CreateRequest struct {
    Instance    InstanceConfig
    Ports       Ports
    DownloadURL string
}

InstanceDir(id string) string
LoadInstance(id string) (*InstanceConfig, error)      // wraps os.ErrNotExist with %w for 404 detection
WriteInstance(id string, inst *InstanceConfig) error
ListInstances() ([]string, error)
ValidateID(id string) error
ValidatePorts(ports Ports) error
CheckPortConflict(excludeID string, ports Ports) error
CheckPortConflictRunning(excludeID string, ports Ports, client *SystemdClient) error
CheckBudget(config *Config, client *SystemdClient, newMemMB int) error
DeleteByID(id string, client *SystemdClient) error

(i *InstanceConfig) Validate() error
(i *InstanceConfig) Create(client *SystemdClient, ports Ports) error   // provision dir, write files, register unit
(i *InstanceConfig) Delete(client *SystemdClient) error
(i *InstanceConfig) Start(client *SystemdClient) error                 // checks budget + port conflicts
(i *InstanceConfig) Stop/Restart(client *SystemdClient) error
(i *InstanceConfig) Enable/Disable(client *SystemdClient) error
(i *InstanceConfig) Status(client *SystemdClient) (*ServiceStatus, error)
(i *InstanceConfig) Download(url string) error
(i *InstanceConfig) ReadPorts() (Ports, error)                         // reads server.properties
(i *InstanceConfig) WriteDropIn() error                                // MemoryMax= systemd drop-in
(i *InstanceConfig) RCON(cmd string) (string, error)

// Launch — syscall.Exec entry point for systemd ExecStart
Launch(id string) error
AikarFlags []string   // recommended JVM GC flags
```

**`systemd.go`**

```go
type ServiceStatus struct {
    Name, State string
    ActiveSince time.Time
}

NewSystemdClient() (*SystemdClient, error)
(c *SystemdClient) Close()
(c *SystemdClient) Start/Stop/Restart(unit string) error
(c *SystemdClient) Status(unit string) (*ServiceStatus, error)
(c *SystemdClient) ActiveSince(unit string) (time.Time, error)
(c *SystemdClient) List(pattern string) ([]ServiceStatus, error)
(c *SystemdClient) Reload() error
(c *SystemdClient) Enable/Disable(unit string) error
UnitToID(unit string) string   // "mcsd-server@foo.service" → "foo"
```

**`rcon.go`**

```go
DialRCON(addr, password string) (*RCONClient, error)
(c *RCONClient) Send(cmd string) (string, error)
(c *RCONClient) Close() error
```

**`helpers.go`**

```go
writeAtomic(path string, data []byte, mode os.FileMode) error
```

---

### `api/`

**`api.go`** — entry point, shared types, route registration.

```go
Serve(port int) error   // port 0 = 8080

// instanceResponse embeds *InstanceConfig — all config fields appear flat in JSON
type instanceResponse struct {
    *core.InstanceConfig
    Ports  *instancePorts `json:"ports,omitempty"`   // nil if server.properties unreadable
    Status instanceStatus `json:"status"`
}

buildInstanceResponse(id string, client *core.SystemdClient) (*instanceResponse, error)
writeJSON(w, status, v)
writeError(w, err)   // maps os.ErrNotExist → 404
```

**Routes:**

```
GET    /instances                        list all
POST   /instances                        create + download JAR
GET    /instances/{id}
PATCH  /instances/{id}                   JSON Merge Patch semantics (pointer fields)
DELETE /instances/{id}

POST   /instances/{id}/start
POST   /instances/{id}/stop
POST   /instances/{id}/restart
POST   /instances/{id}/enable
POST   /instances/{id}/disable
POST   /instances/{id}/upgrade           {vendor, versions} → re-download JAR + update config; 409 if running

POST   /instances/{id}/rcon              {command} → {response}
GET    /instances/{id}/logs              SSE stream; logs since last server start

GET    /instances/{id}/files             list dir  (?path=)
GET    /instances/{id}/files/content     read file (?path=)
POST   /instances/{id}/files/content     write file (?path=) — atomic write
DELETE /instances/{id}/files             delete file/dir (?path=); cannot delete instance root
POST   /instances/{id}/files             multipart upload (?path=destdir)

GET    /vendors                          list vendors + version fields + available options
```

**`logs.go`** — SSE via `journalctl -u mcsd-server@<id>.service -f --output=cat --since=<ActiveSince>`. Uses `r.Context()` for cleanup on disconnect.

**`files.go`** — `safeJoin(base, relPath)` rejects paths that escape the instance directory. `atomicWrite` used for all file writes.

---

### `vendors/`

```go
type VersionField struct {
    Name    string
    Options func() ([]string, error)   // nil = free-form input
}

type Vendor interface {
    Name() string
    VersionFields() []VersionField
    DownloadURL(v VersionSelection) string
}

type VersionSelection map[string]string

var All []Vendor   // registration order = display order
Lookup(name string) Vendor
IsValid(name string) bool
```

Current vendors: `FabricVendor` (fetches versions from meta.fabricmc.net), `BedrockVendor` (no download URL — manual install).

**Adding a new vendor:**
1. Create `vendors/<name>.go`, implement `Vendor` interface.
2. Register in `vendors/vendors.go` — add to `var All`.
3. All current vendors are Java-based. Non-Java vendor requires: `IsJava()` in `vendors.go`, branch in `core/instance.go:resolveArgv()`, executable path prompt in `cli/create.go:runCreateWizard()`.

---

### `cli/`

Thin command layer. Uses Kong (struct-tagged CLI). Each command is a struct with `Run() error`. Zero business logic — delegates to `core`.

| File | Contents |
|------|----------|
| `cli.go` | `Execute()`, root `CLI` struct |
| `create.go` | `CreateCmd`, `runCreateWizard()`, `prompt*` helpers |
| `init.go` | `InitCmd`, `DeinitCmd` |
| `daemon.go` | `DaemonCmd` — start/stop/status of `mcsd.service` |
| `instance.go` | `StartCmd`/`StopCmd`/`RestartCmd`/`StatusCmd`/`ListCmd`/`DeleteCmd`/`RCONCmd`/`EnableCmd`/`DisableCmd`/`ConsoleCmd` |
| `console.go` | `runConsole()` — tcell TUI, journalctl tail + RCON input |
| `edit.go` | `EditCmd` — `$EDITOR` on `config.json` |
| `systemd.go` | `SystemdCmd` (hidden) — `launch`/`stop`/`serve` called only by systemd |

---

## Domain glossary

**Instance** — a managed Minecraft server: its config, working directory, systemd unit, and memory drop-in, identified by a unique ID. Avoid: Server, container, process.

**Instance ID** — unique alphanumeric slug (hyphens/underscores allowed, max 63 chars). Maps to systemd unit and on-disk directory.

**Instance directory** — `/srv/mcsd/instances/<id>/`. All instance files live here. File browser in the web UI is scoped to this directory.

**Instance config** — `/srv/mcsd/instances/<id>/config.json`. mcsd's record of an instance (type, memory, executable, JVM args). Does not include port settings — those live in `server.properties`. Maps to `core.InstanceConfig`.

**Vendor** — a pluggable server type (e.g. Fabric, Bedrock) that knows how to build a download URL and what version fields to prompt for. Implemented via the `Vendor` interface in `vendors/`.

**Version fields** — the vendor-specific inputs needed to identify a server release. Some fetch options from an external API; others are free-form.

**Launch** — the `syscall.Exec` step that replaces `mcsd systemd launch <id>` with the JVM or server binary. Only ever called by systemd as `ExecStart=`.

**Drop-in** — a systemd override file at `/etc/systemd/system/mcsd-server@<id>.service.d/override.conf` that sets `MemoryMax=`. Written by mcsd; enforces per-instance memory limit.

**Budget** — the global `memory_budget` in `/srv/mcsd/config.json`. mcsd rejects a start request if the instance's memory would exceed the budget across all running instances.

**Daemon** — the `mcsd systemd serve` HTTP API process, installed as `mcsd.service`. Managed via `mcsd daemon start|stop|status`. Exposes instance lifecycle, file browsing, SSE log streaming, and vendor discovery over HTTP.

**Web UI** — a Nuxt SPA embedded in the `mcsd` binary via `//go:embed`. Built from `web/`. Served by the daemon at the daemon port. Ships as part of `mcsd` — no separate install. Built with NaiveUI (dark theme) and Tailwind CSS. (ADR-0002, ADR-0004)

**Daemon port** — the HTTP port the daemon listens on. Stored as `port` in `/srv/mcsd/config.json`. Defaults to `8080`. Edit the file and restart `mcsd.service` to change.

**server.properties** — the Minecraft server's own configuration file. Written once by mcsd at instance creation, then owned by the user. Source of truth for game port, RCON port, and RCON password. mcsd reads from it; never overwrites after creation. (ADR-0001)

---

## ADRs

| # | Title | Short summary |
|---|-------|---------------|
| ADR-0001 | `server.properties` write-once | Ports live in server.properties, not config.json; mcsd never overwrites after creation |
| ADR-0002 | Web UI bundled in binary | Nuxt SPA embedded via `//go:embed` — no separate install |
| ADR-0003 | REST file browser over SFTP | REST API in `api/files.go` instead of SFTP daemon |
| ADR-0004 | NaiveUI + Tailwind stack | NaiveUI for components (dark theme), Tailwind for utility styling |
| ADR-0005 | Tabbed instance detail | Home/Files/Settings tabs from v1; Files and Settings are placeholders until v2 |
| ADR-0006 | Host Vitals panel | Separate `HostVitals` domain above instances grid; `thresholdColor` extracted to shared util; single-host model for v1 |

Full text at `docs/adr/`.

---

## Web UI conventions

- **All polling loops live in `app.vue`.** Page components and composables must not start their own `setInterval` / `useIntervalFn` / recursive-`setTimeout` poll loops. Shared reactive data (e.g. the instances list) is owned by `app.vue` and passed down via `provide`.

---

## Agent skills

### Issue tracker
Issues live as local markdown files under `.scratch/`. See `docs/agents/issue-tracker.md`.

### Triage labels
See `docs/agents/triage-labels.md`.

### Domain docs
`/grill-with-docs` — use for planning any new API shape, storage decisions, or feature design. Will validate against this glossary and create ADRs when warranted. Single context: `AGENTS.md` (glossary section) + `docs/adr/`.
