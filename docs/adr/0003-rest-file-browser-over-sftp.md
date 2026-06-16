# REST file browser served by the daemon, not SFTP-only

The web UI needs file management scoped to the instance directory (primarily for mod management — browsing, uploading, and deleting files in `mods/`).

SFTP is already available on the Pi via SSH at no cost. The alternative was to skip web-based file management entirely and tell users to use a client like FileZilla.

We chose to build a REST file browser in the daemon (five endpoints: list, read, write, delete, upload) because the web UI is the primary interface for non-technical users who should not need a separate SFTP client. Mod management is scoped to the web UI (not CLI), so the REST API is the only path for those users.

Path traversal protection is required: all file operations must resolve the requested path and assert it remains under the instance directory before executing.
