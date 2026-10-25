# External Integrations

**Analysis Date:** 2026-10-25

**Release scope:** Release target: Store v2026.10.25.1. Hardware reports remain scoped to the artifacts actually tested. Published v2026.10.25 at `14490119` was the earlier audit baseline. [Picker 0.2.1](https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.2.1) is independently published. It fixes settings limits/malformed fields, idle redraw, per-scan path caching and ABI/provenance checks without changing the launcher, saved settings schema, `state/` preservation, firmware SDL requirements or exact-path local ES API contract. It adds no hardware evidence and is UNTESTED on every matrix; Picker 0.2.0 reports remain historical exact-version evidence. The other six current versions retain Smart Pro-only evidence; all seven review statuses remain experimental.

## APIs & External Services

**Reviewed release downloads:**
- The installer downloads the exact HTTPS release URL in a reviewed manifest. All current release URLs point to GitHub release assets. The manifest pins byte size and SHA-256; validation rejects mutable release aliases. Source: `catalogue/packages/`, `internal/manifest/manifest.go`, `internal/installer/download.go`.
- Client: Go `net/http`, with a default 10-minute timeout and HTTPS-only redirects. The redirect callback rejects a chain when it has five prior requests. The installer checks exact bytes and SHA-256 before extraction. No API token is added to package downloads. Source: `internal/installer/download.go`, `internal/installer/stage.go`.
- ZIP, tar.gz, and reviewed base64-encoded tar.gz payloads are handled as data. A shell-script prefix is never executed. Source: `internal/archive/archive.go`, `internal/installer/stage.go`, `catalogue/packages/io.github.misantronic.raofflineproxy.json`.

**Store self-update:**
- Endpoint: `GET https://api.github.com/repos/jellydn/knulli-app-store/releases?per_page=20`. Client: the manager HTTP client or the default download client. Auth: none; the code sends an Accept header, not a token. Source: `internal/installer/selfupdate.go`.
- The check skips drafts but includes prereleases. It selects a matching target ZIP in API order, but rejects equal or older tags when the shared numeric comparator can order them. Hash builds and ambiguous prereleases still have no monotonic guarantee. The URL must be an exact official tagged asset URL. The API must supply a positive bounded size and a valid `sha256:` digest. Source: `internal/installer/selfupdate.go`, `internal/version/version.go`.
- The updater verifies the ZIP and requires the exact seven-file Store inventory. It changes Store files, not installed app records. The GUI enables this feature only for a stamped target at the canonical executable path. Source: `internal/installer/selfupdate.go`, `internal/appstore/selfupdate.go`, `cmd/knulli-app-ui/main.go`.

**Upstream release checker:**
- Endpoint: GitHub REST `/repos/{owner}/{repo}/releases?per_page=20`. Client: Go `net/http`, with a default 15-second timeout, a 2 MiB response-read bound, and two retries for selected HTTP failures. Source: `internal/updatecheck/check.go`.
- Auth: optional bearer `GITHUB_TOKEN`. Requests set `Accept: application/vnd.github+json` and `X-GitHub-Api-Version: 2022-11-28`. Reports compare stable releases and list ignored prereleases. Source: `cmd/check-updates/main.go`, `internal/updatecheck/check.go`.
- This tool writes JSON and Markdown reports for human review. It does not download assets or edit manifests. Source: `cmd/check-updates/main.go`, `internal/updatecheck/report.go`.

**Knulli EmulationStation:**
- Endpoint: loopback `GET http://127.0.0.1:1234/reloadgames` after a committed game list change. Client: Go `net/http`, with a default 3-second timeout and no proxy. Auth: none. Source: `internal/installer/refresh.go`.
- HTTP 2xx means accepted/queued, not completed. Failure sets restart required; it does not undo the operation. Menu edits use local XML and exact launch-path ownership rules. Source: `internal/installer/refresh.go`, `internal/gamelist/menu.go`, `internal/gamelist/xml.go`, `CONTEXT.md`.

**PortMaster and installed apps:**
- PortMaster is an external-only provider declaration. The catalogue builder loads package manifests, not the provider file. No PortMaster catalogue client or replacement installer is implemented. Source: `catalogue/providers/portmaster.json`, `internal/catalog/catalog.go`, `cmd/knulli-app-ui/main.go`.
- App network declarations describe installed apps, not Store SDK integrations. For example, upstream-managed PocketCurator, RAOfflineProxy, and RetroShelf declare their own runtime risks and network use. The Store does not sandbox those runtimes or execute their upstream install/uninstall scripts. Source: `catalogue/packages/io.github.tomtombombadil.pocketcurator.json`, `catalogue/packages/io.github.misantronic.raofflineproxy.json`, `catalogue/packages/org.retroshelf.knulli.json`, `internal/installer/stage.go`, `internal/installer/uninstall.go`.

## Data Storage

**Databases:**
- No application database server or database client is configured. Installed state and lifecycle records use local JSON files. Source: `go.mod`, `internal/installer/state.go`, `internal/installer/lifecycle.go`.

**File Storage:**
- Local filesystem only. The manager base is `/userdata/system/knulli-app-store`, defined by `manifest.ManagerStatePath`. `-root` redirects device paths into a fixture root. Source: `internal/manifest/manifest.go`, `internal/installer/state.go`, `cmd/knulli-app/main.go`.
- Installed records use `installed/<id>.json`; original-file backups use `originals/<id>/`; lifecycle/retry records use `lifecycle/<id>.json`. Source: `internal/installer/state.go`, `internal/installer/files.go`, `internal/installer/lifecycle.go`.
- The manager uses a local `lock`, `work-*` staging directories, and `transaction-*` directories with `journal.json`. Recovery runs under the manager lock. Source: `internal/installer/stage.go`, `internal/safefs/transaction.go`.
- Force-reinstall recovery backups use `recovery-backups/<id>/`. Self-update stages in `store-update-*`. Source: `internal/installer/recovery.go`, `internal/installer/selfupdate.go`.
- Store binaries and catalogue live at `/userdata/roms/ports/knulli-app-store`. The Store launcher and icon are sibling Ports files. The shared menu is `/userdata/roms/ports/gamelist.xml`. Source: `internal/installer/selfupdate.go`, `scripts/package-device.sh`.
- Legacy controller mapping files are not read or written. Input assignments are automatic and are not persisted. Source: `internal/input/session.go`, `internal/input/input_test.go`.

**Caching:**
- `StatusCache` keeps health results in memory, keyed by the installed-state digest. Manager mutations invalidate the package. An outside file edit needs cache invalidation or a fresh cache before the next full hash check. A nil cache disables reuse. Source: `internal/installer/cache.go`, `internal/installer/status.go`, `internal/appstore/service.go`.

## Authentication & Identity

**Auth Provider:**
- No user accounts or login provider are configured. Catalogue trust uses ed25519 signatures, not user authentication. Source: `go.mod`, `internal/catalog/sign.go`, `internal/catalog/catalog.go`.
- Device builds generate a fresh signing key and pass its public-key linker flag to both builds. The GUI embeds the key used at catalogue open; the CLI's unused key is eliminated, so do not claim it embeds a verification key. The private key is temporary. A local GUI build without an embedded key loads an unsigned catalogue without signature verification. Source: `scripts/device-build.sh`, `internal/catalog/catalog.go`, `internal/catalog/sign.go`, `internal/appstore/service.go`; linked-artifact behavior verified in source thread `T-01a16e63-3dfe-7782-8ac5-a9a71a55726c`.
- Platform identity comes from firmware/version/board files and explicit overrides. Each detected value retains its source. Device identity is not inferred from a shared SoC. SDL controller identity is diagnostic data, not a saved-binding key. Source: `internal/platform/platform.go`, `internal/input/session.go`, `CONTEXT.md`.

## Monitoring & Observability

**Error Tracking:**
- No hosted error-tracking or telemetry integration is configured in the application dependencies or diagnostics implementation. Source: `go.mod`, `internal/diagnostics/log.go`.

**Logs:**
- The Store writes UTC structured `event=name key="value"` lines. Redaction removes terminal controls, masks credential-like values, and removes URL credentials, queries, and fragments. Source: `internal/diagnostics/log.go`.
- The active path is `/userdata/system/logs/knulli-app-store.log`, with one `.1` rotation. The diagnostics writer limits each to 512 KiB. Launcher shell redirection and toolkit stderr are separate write paths; they do not pass through the Go event writer. Source: `internal/diagnostics/log.go`, `internal/sdlui/run.go`, `packaging/trimui-smart-pro/Knulli App Store.sh`.
- User-requested exports combine diagnostic lines and logs in `/userdata/system/knulli-app-store/diagnostics`. Export files use mode 0600. No upload is performed by this code. Source: `internal/diagnostics/log.go`, `internal/appstore/service.go`.

## CI/CD & Deployment

**Hosting:**
- GitHub Releases hosts device ZIPs and `SHA256SUMS.txt`. There is no application web-host deployment in these workflows. Source: `.github/workflows/release.yml`.

**CI Pipeline:**
- `check.yml` runs on pull requests and main pushes. It checks formatting, vet and Staticcheck for both tag sets, race tests, coverage, SDL tests, the static arm64 CLI, and catalogue determinism. A separate job records offline GUI evidence. Device builds run for both targets after the test job. Source: `.github/workflows/check.yml`.
- `release.yml` republishes artifacts from a successful same-repository main check run as a dated prerelease. A `v*` tag instead rebuilds both targets and publishes a full release. Existing release assets can be replaced with `gh release upload --clobber`. Source: `.github/workflows/release.yml`.
- `catalogue-updates.yml` runs Monday at 06:17 UTC or on manual dispatch. It tests the checker, queries release metadata, and retains review reports for 30 days. GUI evidence is retained for 14 days; check-run device artifacts for 30 days. Source: `.github/workflows/catalogue-updates.yml`, `.github/workflows/check.yml`.
- Action references use full commit SHAs. Both check and tag-release device builds use `golang:1.27.1-bookworm`, a version tag rather than an image digest. The build script independently checks host `zip`, `file`, and `readelf` even when the arm64 cross-toolchain and SDL libraries are already installed. Source: `.github/workflows/check.yml`, `.github/workflows/release.yml`, `scripts/device-build.sh`.

## Environment Configuration

**Required env vars:**
- None for ordinary core CLI use. The GUI rejects nonempty `INPUT_CAPTURE`. Device launchers choose `STORE_CONTROLLER_LAYOUT`; desktop CI selects dummy SDL video/audio drivers. Source: `cmd/knulli-app/main.go`, `internal/sdlui/run.go`, `packaging/trimui-smart-pro/Knulli App Store.sh`, `.github/workflows/check.yml`.
- `GITHUB_TOKEN` is optional locally for the maintenance checker and supplied by Actions in its weekly job. Release shell steps use `GH_TOKEN` and `GH_REPO`. Store self-update does not use these tokens. Source: `cmd/check-updates/main.go`, `.github/workflows/catalogue-updates.yml`, `.github/workflows/release.yml`, `internal/installer/selfupdate.go`.

**Secrets location:**
- Per-build signing keys live in a temporary directory removed by a shell trap. Public-key linker flags are supplied to both builds; the GUI retains and uses the key. Source: `scripts/device-build.sh`, `internal/catalog/catalog.go`, `internal/appstore/service.go`.
- GitHub Actions supplies workflow tokens. Verification and catalogue-update workflows request read-only contents access. The release workflow requests `actions: read` and `contents: write`. Source: `.github/workflows/check.yml`, `.github/workflows/catalogue-updates.yml`, `.github/workflows/release.yml`.

## Webhooks & Callbacks

**Incoming:**
- No application HTTP server or incoming webhook endpoint is implemented in the commands. GitHub Actions events trigger workflows; they are not Store endpoints. Source: `cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`, `cmd/check-updates/main.go`, `.github/workflows/`.

**Outgoing:**
- Reviewed release GET requests, GitHub REST metadata requests, and the loopback game list refresh are the Store's network calls. App runtime connections are separate. Source: `internal/installer/download.go`, `internal/installer/selfupdate.go`, `internal/updatecheck/check.go`, `internal/installer/refresh.go`, `catalogue/packages/`.
- This refresh checked public v2026.10.25 API metadata, ran network-backed desktop lifecycle flows and verified both locally built package signatures/inventories. See `TESTING.md`; these are not a general live-service availability test. Separately attributed evidence from source thread `T-01a16e63-3dfe-7782-8ac5-a9a71a55726c` verifies both public release ZIP catalogue signatures against their embedded GUI keys; it does not establish a CLI-embedded key or native testing of those exact release ZIPs. See `docs/real-device-tests.md` for the native development-build evidence and its limits.

---

*Integration audit: 2026-10-25*
