# External Integrations

**Analysis Date:** 2026-10-21

## APIs & External Services

**Reviewed package downloads:**
- Pinned HTTPS GitHub release assets from each manifest's repository. Semantic validation rejects mutable release aliases (`internal/manifest/manifest.go`).
- Standard-library HTTP client: 10-minute timeout, HTTPS-only redirects, redirect count bound, exact byte count and SHA-256 verification before extraction (`internal/installer/download.go`).

**Store self-update:**
- `GET https://api.github.com/repos/jellydn/knulli-app-store/releases?per_page=20`, including prereleases but excluding drafts (`internal/installer/selfupdate.go`).
- Selects the first matching target asset in API order; requires an exact official tagged download URL, positive bounded size, and a GitHub-provided `sha256:` digest. It is not a semantic-version comparison.
- No API token is used. Reuses the manager HTTP client and verifies the downloaded archive before replacing the exact seven-file inventory.
- The packaged GUI enables this only at the canonical Store executable path with a stamped target (`cmd/knulli-app-ui/main.go`, `internal/appstore/selfupdate.go`).

**Catalogue maintenance:**
- Read-only GitHub REST release metadata queries per repository, with API-version and Accept headers and optional bearer `GITHUB_TOKEN` (`internal/updatecheck/check.go`, `cmd/check-updates/main.go`).
- Default 15-second client timeout, 2 retries for retryable responses, bounded response reads, and capped backoff. Reports never download assets or change manifests (`internal/updatecheck/check.go`, `internal/updatecheck/report.go`).

**EmulationStation game list refresh:**
- Loopback `GET http://127.0.0.1:1234/reloadgames` after committed menu changes; default 3-second client timeout with proxy use disabled (`internal/installer/refresh.go`).
- HTTP 2xx means accepted/queued, not completed. Failure requests a restart without undoing the committed operation.

**PortMaster:** External provider declaration only in `catalogue/providers/portmaster.json`; its catalogue is not imported into the package installer. See `docs/adr/0005-portmaster-external-provider.md`.

## Data Storage

**Databases:** None. Persisted state uses local files, not a database server (`internal/installer/state.go`, `internal/installer/lifecycle.go`).

**File Storage:**
- Manager base: `/userdata/system/knulli-app-store`, shared through `manifest.ManagerStatePath` (`internal/manifest/manifest.go`).
- Installed records: `installed/<id>.json`; original-file backups: `originals/<id>/`; lifecycle/retry records: `lifecycle/<id>.json` (`internal/installer/state.go`, `internal/installer/files.go`, `internal/installer/lifecycle.go`).
- Transaction directories: `transaction-*/journal.json`; staging directories: `work-*` and `store-update-*` (`internal/safefs/transaction.go`, `internal/installer/stage.go`, `internal/installer/selfupdate.go`).
- Force-reinstall recovery backups: `recovery-backups/<id>/` (`internal/installer/recovery.go`).
- Store binaries/catalogue: `/userdata/roms/ports/knulli-app-store`; launcher and icon are siblings under Ports (`internal/installer/selfupdate.go`).
- Legacy controller mapping files are neither read nor written; old files remain untouched (`internal/input/session.go`, `internal/input/input_test.go`).

**Caching:** In-memory `StatusCache` keyed on installed-state digest. Installer operations invalidate the affected package; outside edits need invalidation or a new service instance to trigger re-hashing (`internal/installer/cache.go`, `internal/appstore/service.go`).

## Authentication & Identity

**Auth Provider:** No user accounts or login. Catalogue trust uses ed25519, with a fresh signing key generated per device build and the public key embedded in both binaries (`internal/catalog/sign.go`, `scripts/device-build.sh`). `catalog.Load` skips signature verification when no key is embedded (`internal/catalog/catalog.go`).

**Device identity:** Knulli-owned firmware/version/board files, with explicit provenance and overrides, not a shared-SoC guess (`internal/platform/platform.go`). SDL controller GUID/name is diagnostic identity, not a key for persisted custom bindings (`internal/input/session.go`).

## Monitoring & Observability

**Error Tracking:** No hosted telemetry/error service is configured in `go.mod` or `internal/diagnostics/log.go`.

**Logs:**
- Structured `event=name key="value"` lines, terminal-control removal, credential-key masking, and URL credential/query/fragment redaction (`internal/diagnostics/log.go`).
- Active `/userdata/system/logs/knulli-app-store.log` plus one `.1` rotation, each bounded at 512 KiB.
- User-requested plain-text exports include platform/catalogue information and logs under `/userdata/system/knulli-app-store/diagnostics`, written with mode 0600.

## CI/CD & Deployment

**Hosting:** GitHub Releases, device ZIPs, and `SHA256SUMS.txt` (`.github/workflows/release.yml`).

**CI Pipeline:**
- `check.yml`: formatting, vet and Staticcheck for both tag sets, race tests, coverage gate, SDL tests, static aarch64 CLI, deterministic catalogue, offline GUI walkthrough, and two device artifacts.
- `release.yml`: a successful same-repository main check run republishes its artifacts as a dated prerelease; `v*` tag pushes rebuild both targets and publish a full release.
- `catalogue-updates.yml`: Monday 06:17 UTC read-only metadata report, retained for 30 days.
- Action references are pinned to full SHAs in `.github/workflows/`.

## Environment Configuration

**Required env vars:** None for local CLI/core use. Controller/headless variables and build flags are listed in `STACK.md`; device scripts stamp Store target/version and catalogue key (`scripts/device-build.sh`).

**Secrets location:** Per-build catalogue private keys live in a temporary directory removed by a trap. Actions supplies GitHub tokens; release permissions include `contents: write` and `actions: read` (`scripts/device-build.sh`, `.github/workflows/release.yml`). No secret value belongs in a report.

## Webhooks & Callbacks

**Incoming:** No application server or incoming webhook endpoint in `cmd/`; Actions events drive CI workflows.

**Outgoing:** GitHub asset downloads, GitHub REST metadata for maintenance and Store update, and the loopback game list reload. Client policies live with their callers in `internal/installer/download.go`, `internal/installer/selfupdate.go`, `internal/updatecheck/check.go`, and `internal/installer/refresh.go`.

---

*Integration audit: 2026-10-21*
