# Instance detail page uses tabs: Overview / Files / Mods / Settings

The instance detail page (`/instances/[id]`) is structured as four tabs: Overview, Files, Mods, and Settings.

**Overview** contains lifecycle controls (Start/Stop, autostart toggle), instance stats (uptime, memory used) in the page header, and the live log stream with inline RCON console.

**Files** is a full file browser: breadcrumb navigation, flat directory listing, inline textarea editor for text files (`.properties`, `.json`, `.yml`, `.yaml`, `.toml`, `.txt`, `.log`, `.conf`, `.cfg`), drag-and-drop upload, and popconfirm-gated delete.

**Mods** is a placeholder for v2 Modrinth/CurseForge integration.

**Settings** has name editing (PATCH `/instances/{id}`) and read-only port display (ports live in `server.properties` per ADR-0001; port editing is planned for a future version). Instance deletion lives here behind a popconfirm.

A **Config** tab was considered for editing Java args and memory allocation but was dropped — users can edit `config.json` directly through the Files tab. This avoids a duplicate editing surface for fields that are already human-readable in the file.

The tab structure commits to the navigation contract early so that filling in Mods content in v2 is a matter of replacing placeholder content rather than restructuring the page.
