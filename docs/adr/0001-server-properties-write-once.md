# server.properties is written once at creation and never overwritten

mcsd writes `server.properties` exactly once — during `mcsd create` via `initServerProperties`. After that, the file belongs to the user and is the source of truth for game port, RCON port, and RCON password. mcsd reads from it (via `ReadPorts`) but never writes to it again.

The alternative was to re-write `server.properties` on every launch from the instance's internal config — keeping mcsd's config authoritative. We rejected this because it would silently overwrite any manual edits the user makes to the file (mod configs, performance tuning, etc.), which is the primary workflow for managing a Fabric server.
