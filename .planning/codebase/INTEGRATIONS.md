# External Integrations

**Analysis Date:** 2026-09-15

## APIs & External Services

**Package releases:**
- GitHub release assets over HTTPS - installer downloads pinned ZIP/`tar.gz` URLs from reviewed manifests
- SDK/Client: `net/http` in `internal/installer/download.go`
- Auth: none; URL credentials are rejected by redaction and HTTPS-only policy

**Catalogue update metadata:**
- GitHub Releases REST API - weekly read-only checker in `internal/updatecheck` and `cmd/check-updates`
- SDK/Client: `net/http`
- Auth: optional `GITHUB_TOKEN` for rate limits (`.github/workflows/catalogue-updates.yml` uses `github.token`)

**EmulationStation live refresh:**
- `GET http://127.0.0.1:1234/reloadgames` after a committed menu change (`internal/installer/refresh.go`)
- SDK/Client: `net/http`
- Auth: loopback only; success means accepted, not completed

**PortMaster:**
- Featured external provider at `catalogue/providers/portmaster.json`
- Launch mechanism: Knulli official integration; this repo does not call PortMaster APIs

## Data Storage

**Databases:**
- None. Installed state, backups, and controller mappings are files under `/userdata`.

**File Storage:**
- Local Knulli filesystem only
- Manager state: `/userdata/system/knulli-app-store/`
- Logs: `/userdata/system/logs/knulli-app-store.log`
- Controller mappings: `/userdata/system/configs/knulli-app-store/controller-mappings.json`
- Diagnostic exports: `/userdata/system/knulli-app-store/diagnostics`
- Package destinations: declared paths below `/userdata` (often `/userdata/roms/tools` or `/userdata/roms/ports`)

**Caching:**
- None. Staging directories are per-operation temp trees.

## Authentication & Identity

**Auth Provider:**
- None for the App Store itself
- Implementation: catalogue review plus SHA-256 of immutable release bytes; no user accounts

## Monitoring & Observability

**Error Tracking:**
- None

**Logs:**
- `internal/diagnostics` writes a redacted, 512 KiB bounded log with one rotated copy
- Settings can export a redacted diagnostic text bundle
- CI uploads catalogue-update JSON/Markdown reports as Actions artifacts

## CI/CD & Deployment

**Hosting:**
- GitHub repository `jellydn/knulli-app-store`; no hosted app server

**CI Pipeline:**
- `.github/workflows/check.yml` - format, vet, staticcheck, race tests, SDL tests, static aarch64 CLI, deterministic catalogue, experimental device ZIP artifacts
- `.github/workflows/catalogue-updates.yml` - Monday 06:17 UTC plus `workflow_dispatch`; read-only release metadata report

## Environment Configuration

**Required env vars:**
- None for CLI/GUI
- `GITHUB_TOKEN` optional for update checks

**Secrets location:**
- GitHub Actions `github.token` for the weekly checker only
- Diagnostic code redacts URL credentials, query strings, fragments, and common secret fields

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- HTTPS GET of reviewed release URLs
- GitHub Releases API GET for update reports
- Loopback GET to EmulationStation `/reloadgames`

---

*Integration audit: 2026-09-15*
