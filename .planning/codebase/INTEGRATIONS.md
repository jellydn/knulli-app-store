# External Integrations

**Analysis Date:** 2026-09-21

## APIs & External Services

**Package release downloads:**
- GitHub Releases — the only permitted release source. `internal/manifest/manifest.go` (`immutableReleaseURL`) requires an HTTPS `github.com` URL under `<repository>/releases/download/<tag>/<asset>` and rejects `latest` aliases; `internal/installer/download.go` (`validateDownloadURL`) additionally rejects `/latest/` in the path.
- Client: `net/http` with a custom `http.Client` in `internal/installer/download.go` — 10-minute timeout, max 5 redirects, every redirect must stay HTTPS
- Verification: exact byte size plus SHA-256 before the archive reaches the staging area

**GitHub REST API (catalogue maintenance only):**
- `GET https://api.github.com/repos/{owner}/{repo}/releases?per_page=20` in `internal/updatecheck/check.go`
- Headers: `Accept: application/vnd.github+json`, `X-GitHub-Api-Version: 2022-11-28`, optional `Authorization: Bearer $GITHUB_TOKEN`
- Retries: 2 by default on 429 / 5xx / rate-limit-403, honouring `Retry-After` capped at 30s; response body read with a 2 MiB limit
- Read-only by design: the checker never downloads an asset and never edits a manifest

**Knulli game-list reload (loopback):**
- `GET http://127.0.0.1:1234/reloadgames` in `internal/installer/refresh.go` (`emulationStationReloadURL`)
- Client: 3-second timeout, `Proxy: nil` so an emulator proxy cannot intercept loopback
- Overridable in tests via `Manager.RefreshURL` / `Manager.RefreshClient`
- Semantics: HTTP 2xx means *accepted/queued*, never *completed*. Any failure sets `RestartRequired` while the operation stays committed.

## Data Storage

**Databases:**
- None. All state is JSON files below `/userdata/system/knulli-app-store`.

**File Storage:**
- Local filesystem only. No S3/GCS/blob provider, no object storage.
- Managed package state: `/userdata/system/knulli-app-store/installed/<id>.json` (`internal/installer/state.go`)
- Original-file backups: `/userdata/system/knulli-app-store/originals/<id>/<sha256-of-path>` (`internal/installer/files.go`)
- Transaction journals: `/userdata/system/knulli-app-store/transaction-*/journal.json` (`internal/safefs/transaction.go`)
- Lifecycle/retry records: `/userdata/system/knulli-app-store/lifecycle/<id>.json` (`internal/installer/lifecycle.go`)
- Force-reinstall recovery backups: `/userdata/system/knulli-app-store/recovery-backups/<id>/<timestamp>/` (`internal/installer/recovery.go`)
- Diagnostic exports: `/userdata/system/knulli-app-store/diagnostics/*.txt` (`internal/diagnostics/log.go`)
- Controller mappings: `/userdata/system/configs/knulli-app-store/controller-mappings.json` (`internal/input/store.go`)
- Active log: `/userdata/system/logs/knulli-app-store.log` with one rotation at `.log.1`, both capped at 512 KiB

**Caching:**
- In-memory only: `installer.StatusCache` (`internal/installer/cache.go`) memoizes health checks keyed on the digest of the recorded installed state. A nil `*StatusCache` disables caching and re-hashes every file.

## Authentication & Identity

**Auth Provider:**
- None for users — there is no login, account, or session.
- Package integrity uses ed25519 signatures on the catalogue index (`internal/catalog/sign.go`). The private key is generated per device build (`scripts/device-build.sh`) and the public key is compiled in through `embeddedPublicKeyHex`. When no key is embedded, `catalog.Load` skips signature verification (the local/desktop path).
- Device identity is read from Knulli-owned files, never inferred: `OS_NAME="knulli"` in `/etc/os-release`, version from `/usr/share/knulli/knulli.version`, board from `boot/boot/knulli.board` (fallbacks `etc/knulli-device`, `boot/batocera.board`) — see `internal/platform/platform.go`.
- Controller identity is `device|guid:<guid>` or `device|name:<name>` (`internal/input/store.go`, `Identity.Key()`).

## Monitoring & Observability

**Error Tracking:**
- None. No Sentry/Datadog/OTel integration.

**Logs:**
- Structured single-line events via `internal/diagnostics/log.go`: `event=<name> key="value" ...`
- Every value is redacted before it is written: `Redact()` strips ANSI/terminal control sequences, masks `token|password|passwd|secret|api[-_]?key|authorization` values, and drops userinfo, query, and fragment from any URL
- Field names are sanitized to `[a-z0-9_-]` by `safeField`
- Bounded writer with rotation at 512 KiB; existing over-long logs are truncated and re-redacted on `Open`
- User-requested export bundles the platform summary, catalogue listing, and both log files as plain text

## CI/CD & Deployment

**Hosting:**
- GitHub Releases. Device zips plus `SHA256SUMS.txt`.

**CI Pipeline:**
- `.github/workflows/check.yml` — three jobs: `test` (gofmt, vet both tag sets, staticcheck both tag sets, `go test -race`, `make cover`, `go test -tags sdl`, static aarch64 build, deterministic catalogue check), `gui-walkthrough` (headless `make walkthrough` with `SDL_VIDEODRIVER=dummy`, uploads `build/walkthrough` as a 14-day artifact), `device-artifact` (matrix over `trimui-smart-pro` and `magicx-zero-28`, runs `scripts/device-build.sh`)
- `.github/workflows/release.yml` — publishes the exact artifacts a green `check` run produced as a dated pre-release `vYYYY.MM.DD-<sha7>`; a `v*` tag rebuilds both targets through the same script and cuts a full release
- `.github/workflows/catalogue-updates.yml` — weekly (Mondays 06:17 UTC) release metadata check plus a 30-day review-report artifact; read-only, `persist-credentials: false`
- All third-party actions are pinned to full commit SHAs with a version comment
- `prek.toml` mirrors CI locally: fast hooks on pre-commit, `go test -race` and `staticcheck` (both tag sets) on pre-push

## Environment Configuration

**Required env vars:**
- None required to build or run locally.
- `GITHUB_TOKEN` — only for `go run ./cmd/check-updates`; raises the GitHub API rate limit. Never placed in reports.
- `SDL_VIDEODRIVER=dummy` — CI-only, for headless walkthroughs.
- `SDL_GAMECONTROLLERCONFIG` — set by Knulli at runtime; the GUI prefers it over a reviewed fallback mapping.

**Secrets location:**
- No secret files repo-side. The catalogue signing key is generated per build into a `mktemp` directory inside `scripts/device-build.sh` and removed by an `EXIT` trap.
- `GITHUB_TOKEN` comes from the Actions-provided `github.token`; `release.yml` needs `contents: write` and `actions: read`.
- `.gitignore` excludes `/build/`, `/dist/`, `/.amp/`.

## Webhooks & Callbacks

**Incoming:**
- None. Nothing listens on a socket. The only HTTP server in tests is `httptest.NewServer` for fake releases and refresh endpoints.

**Outgoing:**
- `GET http://127.0.0.1:1234/reloadgames` — the single outbound non-download request (post-commit game-list refresh)
- `GET https://github.com/...` release asset downloads
- `GET https://api.github.com/...` release metadata from the read-only checker

---

*Integration audit: 2026-09-21*
