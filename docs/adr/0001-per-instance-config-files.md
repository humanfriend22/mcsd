# Per-instance config.json, not one global instance registry

Considered consolidating all instances' `config.json` into a single `mcsd.json` manifest to reduce file count and simplify `ListInstanceConfigs`. Rejected: CLI and API are independent processes with no shared daemon or lock service (hard project constraint), so a single file means read-modify-write-whole-file on every edit plus cross-process write races that don't exist today. Per-instance files give free isolation — a corrupt write degrades one instance (see `newErrorInstance`), not the fleet. Global `mcsd.json` (`src/core/config.go`) stays scoped to mcsd-wide settings only (memory budget), never per-instance data.

## Status
accepted
