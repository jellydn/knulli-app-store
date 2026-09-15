# Codebase Concerns

**Analysis Date:** 2026-09-15

## Tech Debt

**Catalogue index signing is not implemented:**
- Issue: `catalog.Build` is deterministic and hash-bound, but release automation does not yet sign the index
- Files: `internal/catalog/catalog.go`, `docs/architecture.md`, `docs/adr/0002-declarative-reviewed-catalogue.md`
- Impact: A replaced `catalog-index.json` on device is not cryptographically authenticated by this project
- Fix approach: Add signing and key distribution in the trusted release workflow before calling the index trusted

**Power-loss journal is missing:**
- Issue: Transactions roll back in-process; restart after power loss is not recovered from a journal
- Files: `internal/safefs/transaction.go`, `docs/security-model.md`
- Impact: A kill during commit can leave backups and partial state that need manual repair
- Fix approach: Persist a journal before mutation and resume or roll back on next start

## Known Bugs

**No tracked FIXME/TODO in source:**
- Symptoms: none filed in code comments
- Files: scanned `internal/`, `cmd/`
- Trigger: n/a
- Workaround: product limitations are documented in `docs/security-model.md` and catalogue notes instead

**MagicX GUI and Grout coverage incomplete:**
- Symptoms: MagicX diagnostic confirms platform facts; full GUI and Grout package tests are not done
- Files: `docs/magicx-zero-28.md`, `docs/real-device-tests.md`, `catalogue/packages/app.romm.grout.json`
- Trigger: run on MagicX Zero 28
- Workaround: treat MagicX as experimental; Grout remains Smart Pro only

## Security Considerations

**No package runtime sandbox:**
- Risk: after install, upstream binaries run with user-data access
- Files: `docs/security-model.md`, `internal/installer/installer.go`
- Current mitigation: review, pinned SHA-256, narrow write paths, no remote scripts
- Recommendations: keep `network` as disclosure; do not treat it as an OS grant

**Symlink TOCTOU on local paths:**
- Risk: a hostile local process can race a checked directory into a symlink
- Files: `internal/safefs/guard.go`
- Current mitigation: reject symlink parents at resolve time
- Recommendations: document operator trust in the local root; never point `-root` at an untrusted tree

**Root and `/userdata` policy:**
- Risk: root can still change files below approved paths despite installer checks
- Files: `docs/security-model.md`
- Current mitigation: ownership records, health checks, backups of pre-existing files
- Recommendations: keep manager state outside package write policy (already done)

**Diagnostic export surface:**
- Risk: logs could leak URLs or secrets
- Files: `internal/diagnostics/log.go`
- Current mitigation: redaction, size cap, export excludes package config, credentials, ROMs
- Recommendations: keep GITHUB_TOKEN out of update reports (already required)

## Performance Bottlenecks

**Full archive stage before write:**
- Problem: 512 MiB compressed and installed cap means large staging on constrained cards
- Files: `internal/installer/download.go`, `internal/archive/archive.go`
- Cause: safety requires complete hash and extract before destination mutation
- Improvement path: keep the cap; do not stream-install. Free-space checks already run first

**SDL software fallback and CPU blit:**
- Problem: GUI renders a 640×360 canvas then scales with `golang.org/x/image/draw`
- Files: `internal/sdlui/run.go`, `internal/sdlui/draw.go`
- Cause: portable raster UI without a GPU text stack
- Improvement path: acceptable for catalogue UI; revisit only if frame time is a real-device issue

## Fragile Areas

**Installer lifecycle:**
- Files: `internal/installer/installer.go`, `internal/installer/gamelist.go`, `internal/installer/state.go`
- Why fragile: ownership, preserve paths, menu XML, and refresh outcomes interact
- Safe modification: add a temporary-root test that would fail if the new write or rollback is wrong
- Test coverage: strong lifecycle tests; real-device reports do not itemize each step

**Platform detection:**
- Files: `internal/platform/platform.go`
- Why fragile: Knulli still inherits `ID=buildroot`; board and framebuffer files differ by image
- Safe modification: keep explicit flags; add detection fixtures before new source files
- Test coverage: good unit fixtures; H700 file names still need real-image validation

**SDL cgo adapter:**
- Files: `internal/sdlui/run.go`
- Why fragile: hand-written C wrappers and glibc 2.34 / `libSDL2-2.0.so.0` ABI checks
- Safe modification: change imported SDL symbols only with CI `readelf` updates and both target resolutions
- Test coverage: compile + layout tests, not on-device input

## Scaling Limits

**Catalogue size:**
- Current capacity: four package manifests plus one external provider
- Limit: `catalog.Build` reads every `*.json` and hashes canonical bytes; fine for dozens, untested at hundreds
- Scaling path: keep one file per package; add pagination in the GUI before a large catalogue

**Release size:**
- Current capacity: 512 MiB compressed and installed (`maximumReleaseBytes`)
- Limit: handheld storage and staging
- Scaling path: raise only with a documented device-storage review

## Dependencies at Risk

**golang.org/x/image:**
- Risk: GUI text and scale path only; keep it on a current module line with the Go baseline
- Impact: GUI compile if the module is yanked or incompatible with a later Go bump
- Migration plan: upgrade with the toolchain; keep the CLI CGO-free and independent of this module

**SDL2 and glibc 2.34:**
- Risk: Knulli does not publish this as a compatibility contract; CI uses Debian Bookworm
- Impact: GUI may fail to start on a different libc/SDL build
- Migration plan: record real-device `ldd`/`readelf` evidence per Knulli release

## Missing Critical Features

**Signed catalogue index:**
- Problem: index is deterministic but unsigned
- Blocks: treating device `catalog-index.json` as a trusted distribution artifact

**Crash journal:**
- Problem: no restart recovery for interrupted transactions
- Blocks: guaranteed rollback after power loss

**Package runtime sandbox:**
- Problem: installed programs are not confined
- Blocks: running untrusted upstream code safely

**Actionable RAOfflineProxy and PocketCurator:**
- Problem: script installer and mutable release fail policy
- Blocks: installing those community-approved titles through this manager

## Test Coverage Gaps

**On-device GUI and lifecycle itemization:**
- What's not tested: automated controller input, PowerVR backend, per-step Smart Pro/MagicX results
- Files: `docs/real-device-tests.md`, `internal/sdlui/`
- Risk: layout tests can pass while a device mapping or renderer fails
- Priority: High for MagicX GUI; Medium for richer Smart Pro evidence

**H700 detection files:**
- What's not tested: real Knulli H700 image paths versus fixtures
- Files: `internal/platform/platform.go`, `docs/security-model.md`
- Risk: operators must pass flags when detection is incomplete
- Priority: Medium

**Index signing and journal recovery:**
- What's not tested: absent features
- Files: `internal/catalog/catalog.go`, `internal/safefs/transaction.go`
- Risk: operators may over-trust generated artifacts and crash behavior
- Priority: High before a non-experimental release

---

*Concerns audit: 2026-09-15*
