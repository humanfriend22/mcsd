<!-- GSD:project-start source:PROJECT.md -->

## Project

**MCSD Dashboard**

MCSD is a clean, self-hosted dashboard for managing Minecraft server instances on a systemd Linux host. It's a Go daemon (core business logic shared by an HTTP API and a CLI) plus a Vue/Nuxt web frontend. The web UI is a convenience layer — the CLI must remain a fully reliable, standalone alternative for managing instances without it. Mod management is the eventual goal, but this milestone is about finishing and polishing the core management experience first.

**Core Value:** An admin can reliably manage the full lifecycle of their Minecraft server instances (create, configure, monitor, control) through *either* the CLI or the web dashboard — both backed by the same core logic, neither dependent on the other.

### Constraints

- **Architecture**: CLI and API/web must remain independent, parity-preserving consumers of `src/core/` — no work should make the CLI depend on the API or vice versa.
- **Tech stack**: Go backend (existing conventions in `.planning/codebase/CONVENTIONS.md`), Vue/Nuxt + Naive UI + Tailwind frontend, `huh` for CLI wizards, `kong` for CLI parsing, `coreos/go-systemd` for D-Bus.
- **Security**: No auth currently — deployment assumption is a trusted home network. Do not add scope-creep security work (CORS/auth) this milestone.

<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->

## Technology Stack

## Languages

- Go 1.26.3 - Backend daemon, CLI, API server
- TypeScript - Frontend components and services
- Vue Single-File Components (.vue) - UI framework
- JSON - Configuration and data serialization
- Bash - Build and deployment scripts

## Runtime

- Go runtime (Linux ARM64 target, Darwin development)
- Node.js runtime via Bun (web frontend)
- Linux systemd (production service management)
- Bun (latest) - JavaScript/TypeScript package manager for web
- Go modules - Dependency management
- Lockfiles: `web/package.json` (Bun), `src/go.mod` + `src/go.sum` (Go)

## Frameworks

- Go standard library `net/http` - HTTP server and routing
- Nuxt 4.5.0 - Vue meta-framework for static site generation
- Vue 3.5.35 - Progressive reactive UI framework
- Naive UI 2.44.1 - Vue 3 component library
- TailwindCSS 6.14.0 via `@nuxtjs/tailwindcss` - Utility-first CSS framework
- Ionicons 5 (`@vicons/ionicons5`) - Icon set
- alecthomas/kong 1.15.0 - Declarative CLI framework
- charmbracelet/huh 1.0.0 - Interactive terminal prompts
- charmbracelet/bubbletea 1.3.6 - TUI framework (includes tcell, bubbles, lipgloss)
- coreos/go-systemd/v22 - systemd unit and service management
- godbus/dbus/v5 - D-Bus message bus integration
- mise.toml - Task runner (via `mise` CLI)
- Nuxt generate - Static site generation
- Go compiler with ldflags optimization (`-s -w` for size reduction)

## Key Dependencies

- alecthomas/kong v1.15.0 - CLI parsing and command routing (`src/cli/`)
- charmbracelet/huh v1.0.0 - Interactive CLI wizards (`src/cli/wizard_helpers.go`)
- coreos/go-systemd v22.7.0 - Manages mcsd systemd service and instance services (`src/core/systemd.go`)
- godbus/dbus v5.1.0 - Queries systemd state (status, memory, uptime) via D-Bus (`src/core/systemd.go`)
- nuxt v4.5.0 - Generates static web UI embedded in Go binary
- vue v3.5.35 - Reactive components for instance management UI
- naive-ui v2.44.1 - Pre-built component library (dialogs, dropdowns, forms)
- @vueuse/core v14.3.0 - Vue composition utilities (event sources for log streaming)
- unplugin-auto-import - Auto-import Vue 3 composition API
- unplugin-vue-components - Auto-import Naive UI components
- rolldown-vite (via vite override) - Fast build tool
- charmbracelet/bubbles - Reusable TUI components
- charmbracelet/lipgloss - Terminal styling library
- charmbracelet/colorprofile - Terminal color detection
- gdamore/tcell/v2 - Terminal cell-based rendering

## Configuration

- `mise.toml` - Defines tool versions and build tasks
- `web/tsconfig.json` - TypeScript config (references auto-generated .nuxt configs)
- `web/nuxt.config.ts` - Nuxt runtime configuration
- `.env` files present but not committed (via `.gitignore`) - Secret management
- `src/go.mod` - Go module declaration and dependencies
- `src/go.sum` - Go dependency lock file
- `web/package.json` - Bun package specification
- `/srv/mcsd/config.json` - Global daemon configuration (memory budget)
- `/srv/mcsd/instances/{id}/` - Per-instance directories
- `mcsd.service` - Main daemon service (embedded in binary, deployed at `/etc/systemd/system/`)
- `mcsd-instance@.service` - Template for per-instance services (deployed at `/etc/systemd/system/`)

## Platform Requirements

- macOS (Darwin) with Lima VM or Linux with ARM64 support
- Bun runtime for web development
- Go 1.26.3+
- Lima for VM management (Linux kernel target)
- Linux with systemd (ARM64 or x86_64)
- Java (11+) for Minecraft server runtime
- systemd for service management
- D-Bus daemon for service status queries
- Minimum 512 MB RAM reserved per instance

<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->

## Conventions

## Naming Patterns

- Go files: lowercase with underscores (`instance.go`, `instance_model.go`, `instance_config.go`)
- TypeScript/Vue files: kebab-case for components (`ServerCard.vue`), camelCase for utilities (`api.ts`, `format.ts`)
- Test files: `_test.go` suffix for Go tests
- Go: PascalCase for exported functions (e.g., `Launch()`, `ValidateID()`, `DiscoverJavaBinaries()`)
- Go: camelCase for unexported functions (e.g., `resolveArgs()`, `checkPortAvailable()`)
- TypeScript: camelCase for all functions (e.g., `formatUptime()`, `useInstances()`, `loadInstance()`)
- Vue composables: `use` prefix convention (e.g., `useInstances()`, `useVitals()`, `useInstanceLogs()`)
- Go: camelCase for local variables and parameters
- TypeScript: camelCase for all variables
- Vue: ref names use camelCase (e.g., `const copied = ref(false)`, `const expanded = ref(['servers'])`)
- Go struct fields: PascalCase (exported) or camelCase (unexported)
- TypeScript interfaces: PascalCase (e.g., `Instance`, `Vitals`, `InitData`, `FileEntry`)
- Go structs: PascalCase for exported types (e.g., `ValidationError`, `ServerError`, `NotFoundError`)
- Go JSON struct tags: snake_case (e.g., `json:"active_since"`, `json:"memory_used"`)

## Code Style

- Go: Standard Go formatting rules (gofmt)
- TypeScript: 2-space indentation (inferred from Nuxt project)
- Vue templates: Multi-line attribute formatting for readability
- Line length: No strict limit observed; pragmatic wrapping for readability
- No `.eslintrc` or linting configuration found in web project
- No `golangci-lint` or similar Go linter config found
- Rely on IDE formatting and manual review

## Import Organization

- TypeScript: `~` resolves to web project root (`web/app/`)
- Go modules: `mcsd` prefix for internal packages (e.g., `mcsd/core`, `mcsd/cli`, `mcsd/api`)

## Error Handling

- **Go:** Custom error types for categorization:
- **TypeScript:** Try/catch blocks with error message fallback:
- Error response format (JSON): `{"error": "error message"}`

## Logging

- Go CLI: `fmt.Printf()` for user-facing output and progress messages
- Go startup: `log.Fatal(err)` for fatal initialization errors
- API errors: JSON response with `{"error": "message"}` field
- Web: Console logging via browser console (implicit with Vue)
- CLI TUI: Event-based logging via `tcell` screen events

## Comments

- Exported function declarations (all exported functions have doc comments in Go)
- Complex logic or non-obvious algorithms
- Breaking changes or important behavioral notes (e.g., "// TODO: create a REPL-like interface...")
- HTML/template sections: descriptive comments for logical blocks (e.g., `<!-- Config strip -->`)
- Not consistently applied in TypeScript code
- Go: Standard doc comment format above exported functions (e.g., `// Launch is the ExecStart entry point...`)

## Function Design

- Go: Explicit parameter passing; minimal global state
- TypeScript: Composition API parameters via props/inject; composable functions return reactive state
- Go: Error as final return value (e.g., `func Launch(id string) error`)
- TypeScript: Explicit return types with generics (e.g., `async function sendDataRequest<T>(...)`)
- Vue composables: Return objects with state and methods (e.g., `return { logs, error, open, close }`)

## Module Design

- Go: Packages export functions/types; no default exports
- TypeScript: Named exports + exported types (e.g., `export type Instance = {...}`, `export function useInstances() {...}`)
- Vue: Single-file components with script setup; auto-imported via unplugin-vue-components
- Not used in this codebase; direct imports from module files

## API Response Structure

- All API responses use `writeJSON(w, statusCode, data)` helper
- Error responses: `{"error": "message"}`
- Success responses: Typed data structures (e.g., `Instance`, `Vitals`)

## Vue Component Patterns

- TypeScript with `<script setup lang="ts">`
- Props: `defineProps<{ instance: Instance }>()` with type definition
- Emits: `defineEmits<{ (e: 'open', id: string): void }>()`
- Computed properties for derived state
- Refs for local state (e.g., `const copied = ref(false)`)
- Naive UI components for consistent design (NCard, NButton, NTag, NIcon)
- Tailwind CSS classes for styling
- Accessibility attributes (role, tabindex, aria-label)
- Event handlers with `@click`, `@keydown`, `@update:...`

## HTTP Method Conventions

- `GET /api/...` - Read/list operations
- `POST /api/.../start` - Action commands (start, stop, enable, disable)
- `PATCH /api/instances/{id}` - Update operations
- `DELETE /api/...` - Delete operations
- Path parameters in braces: `GET /api/instances/{id}`

<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->

## Architecture

## System Overview

```text

```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| **Instance Model** | Compose config, ports, and systemd state into unified Instance struct | `src/core/instance_model.go` |
| **Instance Config** | Persist instance metadata (ID, name, vendor, version, memory) to JSON | `src/core/instance_config.go` |
| **Systemd Manager** | D-Bus interface to systemd: unit control, status, resource monitoring | `src/core/systemd.go` |
| **Ports Manager** | Parse/write server.properties, validate port ranges and conflicts | `src/core/ports.go` |
| **Core Bootstrap** | Initialize/deinit mcsd on host, manage global config | `src/core/core.go` |
| **RCON Client** | Minecraft Remote Console protocol implementation | `src/core/rcon.go` |
| **Instance Lifecycle** | Launch, stop, download, upgrade operations | `src/core/instance.go` |
| **API Router** | HTTP route definitions and middleware (CORS, error serialization) | `src/api/api.go` |
| **API Instances** | CRUD and control endpoints for server instances | `src/api/instances.go` |
| **API Files** | List/read/write/delete file operations with path validation | `src/api/files.go` |
| **API Logs** | Server log streaming via HTTP chunked response | `src/api/logs.go` |
| **API RCON** | RCON command execution endpoint | `src/api/rcon.go` |
| **API Vitals** | System resource metrics endpoint (RAM, CPU, disk) | `src/api/vitals.go` |
| **CLI Handler** | Command routing via kong parser, pre-flight checks | `src/cli/cli.go` |
| **Web Static** | Embedded SPA file serving with fallback to index.html | `src/web/web.go` |
| **Vendors** | Interface for server distributors (Fabric, Paper) | `src/vendors/vendors.go` |
| **Frontend API Service** | Wrapper around fetch, composable state management | `web/app/services/api.ts` |
| **Frontend Pages** | Routable page components for dashboard and instance detail | `web/app/pages/` |
| **Frontend Components** | Reusable UI elements (header, tabs, stats, file browser) | `web/app/components/` |

## Pattern Overview

- **Single-process daemon model**: One mcsd daemon manages all instances via systemd templated units
- **Config-driven**: All instance metadata persisted as JSON, systemd provides runtime state
- **D-Bus integration**: Systemd communication via coreos/go-systemd for reliable unit control and metrics
- **Resource budgeting**: Global memory budget divided across instances with validation at create/patch
- **Embedded frontend**: Vue/Nuxt SPA compiled into Go binary, served as static files

## Layers

- Purpose: User-facing UI for server management and monitoring
- Location: `web/app/`
- Contains: Nuxt pages, Vue components, TypeScript services
- Depends on: REST API endpoints at `http://localhost:8080/api/`
- Used by: End users via browser
- Purpose: HTTP request handling and JSON serialization
- Location: `src/api/`
- Contains: Route handlers, error serialization, CORS middleware
- Depends on: Core business logic layer
- Used by: Frontend SPA, CLI via systemd scripts
- Purpose: Instance lifecycle, validation, systemd integration
- Location: `src/core/`
- Contains: Instance model, config persistence, D-Bus manager, RCON, port management
- Depends on: Systemd D-Bus, filesystem operations, vendor metadata
- Used by: API layer, CLI layer
- Purpose: Command-line interface for scripted and interactive management
- Location: `src/cli/`
- Contains: Command handlers with interactive wizards
- Depends on: Core business logic layer
- Used by: System administrators, systemd service templates
- Purpose: Filesystem and systemd service management
- Location: System resources (D-Bus, `/srv/mcsd/`, `/etc/systemd/system/`)
- Contains: JSON config files, systemd unit templates, service files
- Depends on: Nothing (provides foundation)
- Used by: Core layer exclusively

## Data Flow

### Primary Request Path (API → Core → Systemd/Filesystem)

### Instance Lifecycle Flow (Create → Download → Start)

### RCON Console Flow (Frontend → API → RCON Protocol → Instance)

- **Persistent state**: Instance configs stored as JSON files in `/srv/mcsd/instances/{id}/config.json`
- **Live state**: Systemd unit status (active/inactive/failed), memory usage, PIDs via D-Bus queries
- **Transient cache**: In-memory instance list refreshed every 2-5 seconds in background goroutine (`src/api/cache.go`)

## Key Abstractions

- Purpose: Unified representation merging persisted config + runtime state
- Examples: `src/core/instance_model.go` (struct definition), `src/api/instances.go` (API serialization)
- Pattern: Struct composition via embedding (`InstanceConfig` + live fields like `State`, `MemoryUsed`)
- Purpose: Pluggable support for different Minecraft server distributions
- Examples: `src/vendors/fabric.go`, `src/vendors/paper.go` implementing `Vendor` interface
- Pattern: Interface with methods `Name()`, `Versions()`, `Builds()`, `DownloadURL()`; registry pattern with `Get(name)`
- Purpose: Systemd unit state snapshot from D-Bus
- Examples: Used in `src/core/systemd.go:17` and composed into Instance
- Pattern: Immutable value type with fields `State`, `SubState`, `Description`
- Purpose: Network configuration for Minecraft (game port, RCON port, password)
- Examples: Parsed from `server.properties`, validated in `src/core/ports.go:21`
- Pattern: Value type with `Validate()` method; read/write patterns for properties file
- Purpose: Semantic error categorization for proper HTTP status codes
- Examples: `ValidationError` (400), `NotFoundError` (404), `ServerError` (500) in `src/utils/errors.go`
- Pattern: Empty struct with `Error()` string method; used in API middleware for status mapping

## Entry Points

- Location: `src/main.go:12` → `cli.Execute()` (`src/cli/cli.go:57`)
- Triggers: Binary invocation with subcommand (e.g., `mcsd instance start <id>`)
- Responsibilities: Parse CLI flags, call core operations, display results
- Location: `src/main.go:12` → `cli.Execute()` (`src/cli/cli.go:24`) → systemd handler
- Triggers: Systemd service start (`systemctl start mcsd` or `systemctl start mcsd-instance@<id>`)
- Responsibilities: Initialize core, run `api.Serve(port)` infinite loop
- Location: `src/api/api.go:17` (`Serve()` function)
- Triggers: Called from CLI daemon handler
- Responsibilities: Register HTTP routes, start listener on port 8080
- Location: `src/main.go:13` → `core.InitSDManager()` (`src/core/systemd.go:33`)
- Triggers: Process startup before anything else
- Responsibilities: Establish D-Bus connection to systemd; fatal if fails

## Architectural Constraints

- **Threading:** Go goroutines only; no worker pools. Background cache refresh runs in single goroutine. Systemd D-Bus operations block synchronously.
- **Global state:** `SDManager` (D-Bus connection singleton) initialized at process start in `src/core/systemd.go:24`. Frontend instance cache stored in-memory in `src/api/cache.go`.
- **Circular imports:** None detected; import direction flows from API → Core → Utils/Vendors
- **Single instance per process:** Only one mcsd daemon per host; multiple servers managed as systemd templated units (`mcsd-instance@{id}.service`)
- **Filesystem assumptions:** `/srv/mcsd/` as base path; requires root or sudo for systemd operations
- **Synchronous I/O:** No async file operations; all filesystem and D-Bus calls block

## Anti-Patterns

### Lazy Initialization of D-Bus Connection

### Direct Filesystem Path Construction

### Mutable Shared Instance State

### No Request Validation Middleware

## Error Handling

- **ValidationError** (400): Client error, bad input, validation failures. Examples: port out of range, insufficient memory budget, duplicate ID
- **NotFoundError** (404): Resource doesn't exist. Examples: instance not found, file not found
- **ServerError** (500): Server-side failures. Examples: systemd D-Bus failure, filesystem I/O errors, download failure
- **Error serialization** in `src/api/api.go:87` checks error type via `errors.As()` and returns appropriate status code

## Cross-Cutting Concerns

<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->

## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->

## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:

- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->

## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
