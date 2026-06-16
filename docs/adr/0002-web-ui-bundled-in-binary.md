# Web UI is bundled in the mcsd binary

The Nuxt SPA lives in `web/` and is embedded into the `mcsd` binary via `//go:embed`. The daemon serves it at its HTTP port alongside the API. There is no separate `mcsd-web` package.

The alternative was a separate installable package with its own release cycle. We rejected this because the primary deployment target is a single Raspberry Pi — one binary to install and update is simpler than keeping two packages version-aligned. CLI-only users pay a slightly larger binary but gain nothing by a separate package.
