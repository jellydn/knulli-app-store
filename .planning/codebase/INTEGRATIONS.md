# External Integrations

**Analysis Date:** 2026-09-19

## APIs & External Services

**Package releases:**
- GitHub release assets over HTTPS - the installer downloads the URL pinned in the reviewed manifest and rejects anything that is not HTTPS, version-pinned, and immutable (`internal/installer/download.go` `validateDownloadURL`, `internal/manifest/manifest.go` `immutableReleaseURL`)
- SDK/Client: `net/http` with a bounded body (`maximumReleaseBytes` = 512 MiB) and a client built by `defaultHTTPClient`
- Auth: none. URL credentials are rejected, and `internal/diagnostics/log.go` redacts credentials, query strings, and fragments from anything logged

**Catalogue update metadata:**
- GitHub Releases REST API - the weekly read-only checker reads release metadata only, never assets (`internal/updatecheck/check.go`)
- SDK/Client: `net/http` with retry/backoff honouring `Retry-After` (`retryable`, `retryDelay`, `Checker.sleep`)
- Auth: optional `GITHUB_TOKEN`; `.github/workflows/catalogue-updates.yml` passes `github.token` and the workflow pins `contents: read` with `persist-credentials: false`

**EmulationStation live refresh:**
- `GET http://127.0.0.1:1234/reloadgames` after a committed menu change (`internal/installer/refresh.go` `emulationStationReloadURL`)
- SDK/Client: `net/http`; the manager takes `RefreshClient` and `RefreshURL` so tests substitute a local server
- Auth: loopback only. A 200 means queued, not completed; any failure leaves the operation committed and reports `RestartRequired`

**PortMaster:**
- Featured external provider recorded at `catalogue/providers/portmaster.json` with `catalogue_policy: "external-only"` and `launch.mechanism: "knulli-official"`
- No code calls a PortMaster API. The `internal/catalog` build ignores `catalogue/providers/` and reads only `catalogue/packages/`
- ADR: `docs/adr/0005-portmaster-external-provider.md`

**SDL 2.0 (native library, not a network service):**
- `internal/sdlui/run.go` opens the window, renderer, and SDL GameController through cgo
- Runtime controller mappings come from SDL (`SDL_GameControllerDB` as parsed by `internal/input/session.go`), not from a raw evdev node

## Data Storage

**Databases:**
- None. There is no database, no ORM, and no embedded store. All persistent state is JSON and plain files under `/userdata`.

**File Storage:**
- Manager state per package: `/userdata/system/knulli-app-store/<id>.json` (`internal/installer/state.go` `managerPath`, `statePath`)
- Lifecycle/retry state: `/userdata/system/knulli-app-store/lifecycle/` (`internal/installer/lifecycle.go`, schema `org.knulli.app-store/lifecycle-state/v1`)
- Recovery backups for force reinstall: `/userdata/system/knulli-app-store/recovery-backups/<package-id>/` (`internal/installer/installer.go` `recoveryBackupDirectory`)
- Existing-file backups inside the transaction: `originalPath` records under manager state
- Controller mappings: `/userdata/system/configs/knulli-app-store/controller-mappings.json` (`internal/input/store.go`, schema `org.knulli.app-store/controller-mappings/v1`)
- Log: `/userdata/system/logs/knulli-app-store.log` with one rotated copy (`internal/diagnostics/log.go`)
- Diagnostic exports: `/userdata/system/knulli-app-store/diagnostics` (`internal/diagnostics/log.go` `diagnosticDir`, `Log.Export`)
- Package destinations: exactly one declared destination per manifest, always below `/userdata` (for example `/userdata/roms/tools/PlayTime` in `catalogue/packages/io.github.unitreign.playtime.json`)
- Partitioning and backups are file-level; there is no object storage, bucket, or blob service

**Caching:**
- None, deliberately. Every release is downloaded, hashed, and staged per operation before any destination write (`internal/installer/installer.go` `stage`)

## Authentication & Identity

**Auth Provider:**
- None for the App Store itself. There are no accounts, sessions, or tokens for end users.
- The trust chain is: catalogue review status plus an immutable pinned SHA-256 (`internal/manifest/manifest.go` `Release`), and on device an ed25519 signature over `catalog-index.json` verified with a public key compiled into the build (`internal/catalog/sign.go` `Verify`, `EmbeddedPublicKey`)
- `catalog.Load` fails closed when the sidecar signature is absent or does not match (`internal/catalog/catalog.go`)

## Monitoring & Observability

**Error Tracking:**
- None. No Sentry, no telemetry, and no outbound reporting of user activity.

**Logs:**
- `internal/diagnostics` writes timestamped UTC events with a name and key/value pairs; the log is capped at 512 KiB with a single `.1` rotation (`maximumBytes`, `boundedWriter.rotate`)
- Settings in the GUI can export a redacted diagnostics bundle through `appstore.Service.ExportDiagnostics`
- CI uploads `catalogue-update-report.json` + `.md` and the `gui-walkthrough` screen evidence as workflow artifacts

## CI/CD & Deployment

**Hosting:**
- GitHub repository `jellydn/knulli-app-store`. There is no hosted application server and no runtime service to deploy.

**CI Pipeline (`.github/workflows/check.yml`):**
- `test` - gofmt gate, `go vet` and `staticcheck` with and without the `sdl` tag, `go test -race ./...`, `go test -tags sdl ./...`, static `CGO_ENABLED=0` aarch64 CLI build, and a determinism check that builds the catalogue twice and `cmp`s the output
- `gui-walkthrough` - `make walkthrough` under `SDL_VIDEODRIVER: dummy`, uploading `build/walkthrough` as the `gui-walkthrough` artifact with `if-no-files-found: error`
- `device-artifact` - per-device matrix (`trimui-smart-pro`, `magicx-zero-28`) in a Bookworm container: build a signed index, cross-compile both binaries with the public key injected, verify `file`/`readelf` (two NEEDED entries, `GLIBC_2.34`), and package with `scripts/package-device.sh`

**Release Pipeline (`.github/workflows/release.yml`):**
- `publish` - triggered by `workflow_run` on a green `check` run for `main`, re-publishes the verified artifacts as a dated pre-release (`v2026.09.16-6655b2d` style) with `SHA256SUMS.txt`; nothing is rebuilt
- `tag-build` / `tag-release` - a `v*` tag rebuilds both device packages with the same signed-catalogue, cross-compile, and ABI steps, then publishes a full release

## Environment Configuration

**Required env vars:**
- None to build or run the CLI or GUI

**Optional env vars:**
- `GITHUB_TOKEN` - raises GitHub API rate limits for `cmd/check-updates`; must never appear in a generated report

**Secrets location:**
- GitHub Actions `github.token` only, scoped to the workflows that request it
- The catalogue signing key is generated in a temporary directory per device build and never persisted (`.github/workflows/check.yml`, trap on exit)

## Webhooks & Callbacks

**Incoming:**
- None. The app exposes no server, port, or endpoint.

**Outgoing:**
- HTTPS GET of the reviewed release asset (`internal/installer/download.go`)
- GitHub Releases API GET for the update report (`internal/updatecheck/check.go`)
- Loopback GET to EmulationStation `/reloadgames` (`internal/installer/refresh.go`)

---

*Integration audit: 2026-09-19*
