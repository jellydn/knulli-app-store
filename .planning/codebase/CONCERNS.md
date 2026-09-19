# Codebase Concerns

**Analysis Date:** 2026-09-19

## Tech Debt

**Per-build catalogue signing key:**
- Issue: device artifacts generate a throwaway ed25519 key per build, sign `catalog-index.json`, and compile the matching public key in with `-X ...internal/catalog.embeddedPublicKeyHex`. Every signed index therefore requires a matching new binary.
- Files: `internal/catalog/sign.go` (`embeddedPublicKeyHex`, `EmbeddedPublicKey`), `.github/workflows/check.yml` (device-artifact job, `KEY_DIR=$(mktemp -d)` with an exit trap), `.github/workflows/release.yml` (tag-build job)
- Impact: a catalogue refresh cannot ship independently of an app build; rotation policy does not exist yet
- Fix approach: introduce a long-lived production key plus a documented rotation path once the index is distributed apart from the binary (`docs/security-model.md` already names this as required)

**Duplicated version comparison:**
- Issue: two independent implementations compare versions, one for platform compatibility and one for the update checker. `normalizeVersion` also exists in both packages with different behavior.
- Files: `internal/platform/platform.go` (`comparableVersions`, `compareVersions`, `versionPart`, `numericVersion`) and `internal/updatecheck/check.go` (`compareVersions`, `versionParts`, `normalizeVersion`, `versionParts`)
- Impact: a fix in one comparison can silently diverge from the other, so a package could be compatible but reported as out of date, or vice versa
- Fix approach: extract one version-ordering helper both callers share, then keep only the caller-specific policy on top

**Duplicated device build block across workflows:**
- Issue: the signed-catalogue build, the cross-compile, and the ABI verification (`file`, `readelf` for `libSDL2-2.0.so.0`, exactly two `NEEDED` entries, `GLIBC_2.34`) are copy-pasted between the `device-artifact` job in `check.yml` and the `tag-build` job in `release.yml`
- Impact: relaxation of an ABI gate or a new link flag can land in one path and not the other, so a tag release could ship something CI never validated
- Fix approach: move the block into one reusable workflow (or a script under `scripts/`) and call it from both jobs

**Mixed GitHub Action pinning:**
- Issue: `catalogue-updates.yml` pins every action to a full commit SHA, while `check.yml` and `release.yml` use floating major tags (`actions/checkout@v7`, `actions/upload-artifact@v7`)
- Impact: the highest-trust workflow is the least pinned one; a compromised major tag would execute in the release path
- Fix approach: pin all workflows by SHA and let Renovate keep the comments current

**Installer orchestrator size:**
- Issue: `internal/installer/installer.go` is 819 lines and holds the whole commit path plus `installFiles`, `adoptFiles`, `inventoryExisting`, `applyExecutableModes`, `applyBinaryPatches`, and `recoveryBackup`
- Impact: the file is the hardest place to review, and it is the file where a mistake is most expensive
- Fix approach: the stage split (`checkPreconditions` / `stage` / `commit`) is already named; move commit helpers into sibling files by concern (files, adoption, recovery, patches) before adding new behavior

**Manager carries lifecycle methods:**
- Issue: `LifecycleState`, `RecordLifecycle`, `ClearLifecycle`, and `LifecycleEvent` hang off `installer.Manager` although lifecycle/retry provenance is a separate schema (`org.knulli.app-store/lifecycle-state/v1`) with its own path
- Files: `internal/installer/lifecycle.go`
- Impact: the manager surface grows with each retry rule, and lifecycle storage cannot be tested or replaced independently
- Fix approach: give lifecycle its own small type with a path and keep the manager as a caller

**Unbounded notice queue:**
- Issue: `ui.Toasts.Push` appends without a cap, and only `Live` (called each rendered frame) drops expired entries; rendering truncates to `toastMaxLines = 2`
- Files: `internal/ui/toast.go`, `internal/sdlui/layout.go` (`toastMaxLines`), `internal/sdlui/draw.go` (`drawToast`)
- Impact: a fast burst of completed operations grows the slice until the next read; harmless today, a slow leak if the frame loop ever stops reading
- Fix approach: cap the queue at push time (keep the newest N) so the bound does not depend on the render loop

## Known Bugs

**No tracked TODO/FIXME/HACK/XXX in source:**
- Symptoms: none filed in code comments
- Files: scanned `cmd/`, `internal/`, `catalogue/`, `schema/`, `scripts/`, `packaging/` — zero matches
- Trigger: n/a
- Workaround: product limitations are recorded in `docs/security-model.md`, `docs/real-device-tests.md`, and the dated catalogue reviews instead of in code

**MagicX GUI coverage incomplete:**
- Symptoms: a real MagicX diagnostic confirms Knulli Scarab, `aarch64`, 640×480, and SDL GameController `magicx-input`, but full GUI and package lifecycle testing is not done
- Files: `docs/magicx-zero-28.md`, `docs/real-device-tests.md`, `catalogue/packages/io.github.unitreign.playtime.json` (notes)
- Trigger: a run on a MagicX Zero 28
- Workaround: treat MagicX as experimental; only screenshot and layout evidence exists at 640×480

**Grout device evidence is one version behind:**
- Symptoms: the recorded Smart Pro test covers Grout 5.1.0.0, not the current 5.2.0.0 release
- Files: `docs/real-device-tests.md`, `docs/security-model.md` (residual risks), `catalogue/packages/app.romm.grout.json`
- Trigger: reviewing Grout 5.2.0.0
- Workaround: Grout stays broad-experimental; the Store stages a verified binary patch to disable the self-updater

**RetSend ships with open upstream blockers:**
- Symptoms: the reviewed build's own tracker lists eight open admission blockers (TLS pinning, key-file mode, symlink-safe writes, overwrite default, quotas, signed releases, provenance attestation, SBOM), and the release tag `prerelease` is mutable so the manifest pins bytes and not the tag
- Files: `docs/catalogue-review-2026-09-17.md`, `catalogue/packages/io.github.jellydn.retsend.json`, `internal/installer/installer_test.go` (RetSend lifecycle test)
- Trigger: installing or reviewing RetSend
- Workaround: the manifest disables the self-updater, pins size and SHA-256, keeps identity and history outside the package tree, and stays `experimental` with no device test

## Security Considerations

**No package runtime sandbox:**
- Risk: after install, upstream binaries run with the user's access to `/userdata`
- Files: `docs/security-model.md`, `internal/installer/installer.go`
- Current mitigation: catalogue review, pinned SHA-256, one declared destination, backups, a disclosure-only `network` field
- Recommendations: keep documenting that `network` is a disclosure and not an OS grant; do not present it as enforcement

**Symlink TOCTOU on local paths:**
- Risk: a hostile local process can race a checked directory into a symlink between the check and the write
- Files: `internal/safefs/guard.go` (`rejectSymlinkParents`, `allows`), `internal/installer/installer.go`
- Current mitigation: every existing component of a write path is rejected if it is a symlink
- Recommendations: keep the operator-trust statement prominent — never point `-root` at an untrusted tree, and `scripts/desktop-fixture.sh` must refuse non-fixture directories (it does, via its `.desktop-fixture` marker)

**Blast radius of root:**
- Risk: root can change files below approved paths regardless of installer checks
- Files: `docs/security-model.md`, `internal/safefs/transaction.go`
- Current mitigation: manager state lives outside package write policy, ownership is recorded, pre-existing files are backed up, uninstall restores originals
- Recommendations: keep state and write policy separate; do not add a write path under manager state

**Recovery from a corrupt journal is manual:**
- Risk: a corrupt transaction journal blocks the next locked operation until the leftover directory is inspected
- Files: `internal/safefs/transaction.go` (`Recover`, `readJournal`, `rollbackSnapshots`), `internal/installer/installer.go` (`rollback`)
- Current mitigation: `Recover` runs on the next locked start and refuses to guess at a damaged record
- Recommendations: surface the exact leftover path in the errors the GUI shows, so an operator does not have to read the journal by hand

**Diagnostic export surface:**
- Risk: exported logs could leak URLs, tokens, or paths
- Files: `internal/diagnostics/log.go` (`Redact`, `sensitiveValue`, `webURL`, `terminalControl`, `Export`)
- Current mitigation: redaction before write, 512 KiB cap with one rotation, export triggered by the user and limited to log lines
- Recommendations: keep `GITHUB_TOKEN` out of the update reports (`internal/updatecheck/report.go` writes only compared metadata) and keep adding a redaction case whenever a new URL shape is logged

**Download URL policy:**
- Risk: a manifest could point at a mutable or non-HTTPS asset
- Files: `internal/installer/download.go` (`validateDownloadURL`, `maximumReleaseBytes`), `internal/manifest/manifest.go` (`immutableReleaseURL`, `isHTTPS`)
- Current mitigation: HTTPS-only, version-pinned URLs, exact size and SHA-256 verified before extraction
- Recommendations: keep the mutable-release treatment (pin bytes, refuse the tag) as the standard for any fork-published asset

## Performance Bottlenecks

**Full archive stage before any write:**
- Problem: up to 512 MiB compressed and installed must be downloaded, hashed, and extracted into a temp tree before a destination changes
- Files: `internal/installer/download.go` (`maximumReleaseBytes`), `internal/installer/installer.go` (`maximumAdoptionBytes`, `stage`), `internal/archive/archive.go`
- Cause: safety requires a complete hash and inventory before mutation, and there is no cache
- Improvement path: keep the design; free-space is already checked before download. Do not stream-install, because it would move the checksum after the first write

**SDL software raster and CPU blit:**
- Problem: the GUI renders a 640×360 canvas and scales it with `golang.org/x/image/draw` on every frame
- Files: `internal/sdlui/run.go` (`renderOutput`, `outputRectangle`), `internal/sdlui/draw.go`
- Cause: a portable raster UI avoids a GPU text stack and keeps one code path for both target resolutions
- Improvement path: acceptable for a catalogue list; revisit only if a real-device frame time says otherwise

**Walkthrough cost grows with every flow:**
- Problem: `make walkthrough` renders every screen state at both resolutions and replays each flow's keys through the real binary
- Files: `scripts/desktop-walk.sh`, `internal/sdlui/walk.go`, `internal/sdlui/draw_test.go`, `.github/workflows/check.yml` (`timeout-minutes: 15`)
- Cause: evidence is produced by driving the real GUI rather than by mocking it
- Improvement path: the timeout is the tripwire; if it is ever raised, split the offline and `--install` flows into separate jobs instead

## Fragile Areas

**Installer lifecycle:**
- Files: `internal/installer/installer.go`, `internal/installer/state.go`, `internal/installer/status.go`
- Why fragile: ownership, preserved paths, menu ownership, recovery backups, and refresh outcomes interact, and installed state must be written last
- Safe modification: add a temporary-root test that fails if the new write or rollback is wrong, and route every mutation through the transaction
- Test coverage: strong (34 tests, 1,421 lines), including force reinstall and the RetSend package

**Menu XML rewriting:**
- Files: `internal/gamelist/xml.go`, `internal/gamelist/menu.go`
- Why fragile: the rewrite must preserve unknown elements and attributes byte-for-byte in effect, and must touch only an entry the installer created and that is still unchanged
- Safe modification: extend `internal/installer/testdata/gamelist.xml` with the new shape before changing the encoder, and keep `Derive`/`Apply` separate so plan and effect stay testable apart
- Test coverage: good (11 tests) on ownership rules; encoder fidelity relies on round-trip cases

**Platform detection:**
- Files: `internal/platform/platform.go`
- Why fragile: Knulli still inherits `ID=buildroot`, so detection reads `OS_NAME` from `/etc/os-release`, then `/usr/share/knulli/knulli.version`, then a board file with three candidate locations; framebuffer and ABI files differ by image
- Safe modification: add a detection fixture and keep the `WithX` override flags, so an unrecognized board is blocked rather than guessed
- Test coverage: good unit fixtures (21 tests); real-image validation for H700 boards still needs evidence

**SDL cgo adapter and ABI contract:**
- Files: `internal/sdlui/run.go` (cgo preamble, `openController`, `connectController`), `internal/sdlui/events.go`
- Why fragile: the hand-written wrappers may import only SDL symbols present on the oldest supported Knulli image, and the binary must link exactly `libSDL2-2.0.so.0` plus glibc no newer than 2.34
- Safe modification: change imported symbols only together with the CI `readelf` checks and both target resolutions
- Test coverage: compile plus layout and event tests, never real kernel SDL loading

**Input setup state machine:**
- Files: `internal/input/session.go` (520 lines, 20+ mode/effect transitions), `internal/input/calibration.go`, `internal/input/footer.go`
- Why fragile: every mode must keep a reachable exit, and keyboard codes (`KeyEnter = 100` upward in `internal/input/keyboard.go`) must stay above every SDL GameController button number
- Safe modification: add the flow to `scripts/desktop-walk.sh` and assert the screens it must reach; treat the keyboard offset block as fixed
- Test coverage: broad (17 input tests plus paging, keyboard, review, and footer suites) and clamped by the source audit

## Scaling Limits

**Catalogue size:**
- Current capacity: five package manifests plus one external provider
- Limit: `catalog.Build` reads every `*.json`, re-encodes canonically, and hashes each; `appstore.Service.Items` calls `manager.Status` per package on every load, which hashes managed files for installed packages
- Scaling path: fine for dozens. Before hundreds, cache status per package identity (hashes plus observed mode) rather than re-hashing on each `Items` call, and revisit list virtualization in `internal/sdlui/layout.go` (`listRows` is a compile-time row count)

**Release and adoption size:**
- Current capacity: 512 MiB compressed and 512 MiB installed (`maximumReleaseBytes`), with the same ceiling for `maximumAdoptionBytes`
- Limit: handheld SD card space and temp staging
- Scaling path: raise only with a documented device-storage review; the free-space check already runs before download

**Footer and notice text:**
- Current capacity: `footerCharacterLimit = 86` (`internal/sdlui/hints.go`), 2 notice lines, `toastCharacterLimit` derived from panel width
- Limit: a long package name or a long reason must be shortened, and `internal/sdlui/draw.go` `shorten`/`wrapText` are the only tools
- Scaling path: prefer shortening at the source (labels, verdict messages) over widening the geometry

## Dependencies at Risk

**golang.org/x/image:**
- Risk: used only for scaling and text in the GUI; Renovate keeps it current alongside the Go baseline
- Impact: a GUI compile failure if the module is yanked or stops supporting the toolchain
- Migration plan: the CLI is CGO-free and independent of this module, so the installer keeps working if the GUI cannot build

**SDL2 and glibc 2.34:**
- Risk: Knulli publishes no compatibility contract for these; CI verifies against Debian Bookworm, so the check proves the build host, not the device
- Impact: the GUI may fail to start on a different libc or SDL build
- Migration plan: record `ldd`/`readelf` evidence per Knulli release in `docs/real-device-tests.md`

**GitHub API surface:**
- Risk: `internal/updatecheck` depends on release-list shape, pagination, and rate-limit headers
- Impact: the weekly report degrades to an error column, which is visible but easy to ignore
- Migration plan: the checker is read-only and never edits a manifest, so a break costs a report, not a release

## Missing Critical Features

**Package runtime sandbox:**
- Problem: installed programs are not confined after launch
- Blocks: running untrusted upstream code with any real containment

**Long-lived catalogue signing key:**
- Problem: trust is rooted in a key that changes with every build
- Blocks: shipping a catalogue index without a matching new binary

**Actionable RAOfflineProxy and PocketCurator:**
- Problem: RAOfflineProxy's Knulli asset is a self-extracting script, and PocketCurator's release is mutable; neither passes the supported-archive, inventory, narrow-write, or updater-safety policy
- Blocks: installing those community-approved titles through this manager (both remain recorded as approved but blocked in `README.md`)

**Coverage gate:**
- Problem: no coverage threshold exists, and no coverage command is part of `make check`
- Blocks: detecting a new package shipped with no tests

## Test Coverage Gaps

**On-device controller and renderer:**
- What's not tested: automated controller input, the accelerated SDL backend, and per-step Smart Pro/MagicX lifecycle results
- Files: `docs/real-device-tests.md`, `internal/sdlui/`
- Risk: layout, event, and walkthrough tests can all pass while a real GameController mapping or renderer fails
- Priority: High for MagicX GUI, Medium for itemized Smart Pro evidence

**H700 and other board detection:**
- What's not tested: real Knulli image paths versus the fixtures in `internal/platform/platform_test.go`
- Risk: operators must pass explicit flags when detection is incomplete, and an unknown board blocks rather than installs
- Priority: Medium

**Release workflow itself:**
- What's not tested: `.github/workflows/release.yml` has no test and no dry run; the only exercise is an actual tag
- Risk: the duplicated ABI block and the dated-tag derivation are validated in production
- Priority: Medium, and it drops to Low once the device build block is shared with `check.yml`

**Signing key rotation:**
- What's not tested: any path where a signed index outlives the binary that embedded its key
- Files: `internal/catalog/sign.go`, `.github/workflows/check.yml`
- Risk: an operator could treat a per-build signature as a stable trust root
- Priority: Medium until the index is distributed on its own

---

*Concerns audit: 2026-09-19*
