# Codebase Concerns

**Analysis Date:** 2026-10-25

This audit uses the official Option B baseline and the local changes above it. Each prior concern was checked against current code, published v2026.10.25 metadata and the linked device reports. Possible failures are risks, not confirmed bugs. No new hardware or security penetration test was performed.

Release target: Store v2026.10.25.1. Hardware reports remain scoped to the artifacts actually tested. The earlier audit baseline was published v2026.10.25 at `14490119`; historical artifact checks below apply to that baseline.

## Tech Debt

No open defect remains from the prior check-scope and setup findings. `make check` is deliberately the fast loop; `README.md` and `TESTING.md` list the separate full gates. CI already checks hooks, dependency locks, both analyzer tag sets and device ABI. `.agents/setup` now selects exactly Go 1.27.1, matching CI and Staticcheck v0.8.1. `scripts/device-build.sh` now checks host packaging tools separately, so an existing cross-toolchain cannot hide a missing `zip`, `file` or `readelf`. These are setup fixes, not changes to package policy.

## Known Bugs

No unresolved reproducible current bug was established. The PocketCurator `$0` patch is already present and tested; its older-build failure and Repair instructions remain in `docs/real-device-tests.md`, not as current debt. Stale read-only, setup-wizard, single-card and Picker 0.1.1 guidance was corrected in the README, device guides, packaged guides and maps. The [owner's 2026-10-25 confirmation](https://ampcode.com/threads/T-01a17039-648c-7024-abe9-8785444a2b5b) (Asia/Singapore) supplies passed package evidence for seven listed versions on TrimUI Smart Pro only, Knulli AArch64 at 1280×720, superseding the older broad report. Picker 0.2.0 is now historical exact-version evidence; published Picker 0.2.1 is UNTESTED on every matrix. The other six current versions retain Smart Pro-only evidence. Review statuses remain experimental; MagicX and other matrices remain untested.

## Security Considerations

**Store self-update trusts GitHub and the publisher account:**
- Risk: An account compromise can supply malicious Store bytes and a matching GitHub digest. The bundled catalogue signature does not independently authenticate the Store ZIP.
- Files: `internal/installer/selfupdate.go`, `internal/appstore/selfupdate.go`, `docs/security-model.md`.
- Current mitigation: A confirmed check reads the official HTTPS release API. A separate Update action uses the offered URL, exact size, and SHA-256. Download verification, safe extraction, an exact seven-file inventory, a manager lock, snapshots, and journaled replacement limit installation changes. A completed update requires exit and reopen.
- Recommendations: Treat this as the stated trust boundary. No independent Store update signing key or signature feed is implemented.

**Release ordering remains limited for hash builds and ambiguous prereleases:**
- Risk: Git hashes have no numeric release order. Different prerelease suffixes of the same numeric version are deliberately not ordered. The updater includes prereleases and checks only 20 release records.
- Files: `internal/installer/selfupdate.go`.
- Current mitigation: A regression test reproduced v3 receiving an older v2 offer. Discovery now rejects equal or older tags when `version.CompareNumeric` can order them; newer numeric releases remain offered. Existing target, URL, digest, current-build and draft checks remain. Tests are in `internal/installer/selfupdate_test.go`.
- Recommendations: Hash-build and ambiguous-prerelease discovery still uses API order. A complete monotonic update policy needs release-channel/build identity requirements; this audit does not invent them or change the installer trust policy.

**Catalogue keys are generated per device build:**
- Risk: A catalogue from another build cannot be verified with the current embedded key, even if both builds come from this project.
- Files: `scripts/device-build.sh`, `internal/catalog/sign.go`, `docs/adr/0008-catalog-index-signing.md`.
- Current mitigation: Packaging ships the key-bearing GUI and signed catalogue together. The CLI's unused key can be removed by the linker. Store replacement uses one transaction. Startup recovery runs before catalogue verification.
- Recommendations: A stable key and key-rotation policy are not implemented. Unsigned development builds are not equivalent to signed device builds.

**Installed apps have no runtime sandbox:**
- Risk: Store write-path rules apply to installation, not to app execution. Apps can change files, download code, or update themselves outside the reviewed transaction.
- Files: `docs/security-model.md`, `CONTEXT.md`, `catalogue/packages/io.github.tomtombombadil.pocketcurator.json`, `catalogue/packages/io.github.misantronic.raofflineproxy.json`, `catalogue/packages/org.retroshelf.knulli.json`.
- Current mitigation: These three upstream-managed packages stay experimental and display warnings. Repair restores pinned code and keeps declared user data. PocketCurator warns about permanent ROM deletion and executable downloads. RAOfflineProxy and retroshelf declare read-only removal checks for shared integration state. The Store never runs upstream cleanup scripts.
- Recommendations: A successful Store install does not approve later downloaded code or prove safe runtime behavior. Declared removal checks do not find every active process or custom runtime path.

**Upstream evidence has explicit exceptions and limits:**
- Risk: A GitHub asset digest is not independent publisher authentication. An unverified license is not permission to redistribute or use an asset. RetSend also has documented upstream transport and receive-path limits.
- Files: `docs/security-model.md`, `catalogue/packages/io.github.misantronic.raofflineproxy.json`, `catalogue/packages/org.retroshelf.knulli.json`, `catalogue/packages/io.github.jellydn.retsend.json`, `docs/retsend-upstream-1.0.0.md`.
- Current mitigation: RAOfflineProxy discloses its digest-only evidence. Retroshelf warns that its license is unverified and uses direct upstream download. RetSend's review records missing authenticated peer pinning, restrictive key-mode controls, rooted symlink-safe receive writes, and quota controls. These are recorded upstream limits, not demonstrated Store vulnerabilities.
- Recommendations: Keep evidence exceptions separate from verified status. No package promotion is implied by this map.

**Write-path checks do not stop a hostile local race:**
- Risk: Another privileged process can change a checked directory before a later write. Root can also change installed files directly.
- Files: `internal/safefs/guard.go`, `internal/safefs/files.go`, `docs/security-model.md`.
- Current mitigation: Guards reject paths outside declared destinations and symlinks in existing path components. Transactions use atomic replacement and snapshots.
- Recommendations: This is a documented threat-model limit. No local race exploit was tested here.

**Diagnostic redaction has a limited scope:**
- Risk: Arbitrary opaque secrets or credentials inside URL paths are not guaranteed to be removed.
- Files: `internal/diagnostics/log.go`, `docs/security-model.md`.
- Current mitigation: Redaction removes common secret fields, URL userinfo, queries, fragments, and terminal controls. Logs are redacted on write, reopen, and export. Export does not scan app settings or ROMs.
- Recommendations: Do not treat exported text as a proof that all possible secrets were removed.

## Performance Bottlenecks

**Idle redraw fixed; handheld measurement remains open:**
- Confirmed cause: `internal/sdlui/run.go` drew, uploaded and presented the full Library every 30 Hz tick without a state change.
- Resolution: Input and model polling remain at 30 Hz. Rendering now follows SDL events (including window/controller changes), held navigation repeats, operation events and notice expiry. Screenshot and walkthrough runs keep full capture frames. Option B layout and installer policy are unchanged.
- Controlled desktop evidence: Three runs per binary, 2 seconds warm-up then 6 quiet seconds, dummy SDL/software renderer, 1280×720 fixture, CPU ticks from `/proc/PID/stat`. Before: 19.17%, 18.33%, 20.00% of one core (median 19.17%). After: 1.00%, 1.00%, 1.17% (median 1.00%). These are orb results, not handheld or battery results.
- Remaining work: Repeat controlled idle/active measurements on each handheld. The historical native 78.0% sample was a process average after active work, not an idle baseline. Source: `docs/real-device-tests.md`.

**Full health checks read managed file content:**
- Problem: A full health check hashes each immutable managed file. This can require substantial device I/O for large apps. No target timing was measured here.
- Files: `internal/installer/status.go`, `internal/installer/cache.go`, `internal/appstore/service.go`.
- Cause: Hash checks provide content evidence. The service caches results by installed-state digest to avoid repeating all hashes on each catalogue load.
- Improvement path: The current tradeoff is explicit: outside file edits can remain unseen until cache invalidation or a new service instance. Cached health is not live filesystem monitoring.

**Staging and rollback need spare storage:**
- Problem: Normal staging requires release bytes + 3 × declared installed bytes + 2 × pre-existing bytes before download. This can reject an install whose final files alone would fit.
- Files: `internal/installer/stage.go`, `internal/installer/selfupdate.go`.
- Cause: Verified staging, snapshots, and retained originals need temporary space. Store self-update has separate download and post-extraction space checks.
- Improvement path: No storage optimization or device performance result is claimed. These checks are safety limits, not evidence of a leak.

## Fragile Areas

**Recovery and Store replacement must stay in order:**
- Files: `internal/safefs/transaction.go`, `internal/safefs/files.go`, `internal/installer/selfupdate.go`, `internal/installer/stage.go`, `internal/appstore/service.go`, `packaging/trimui-smart-pro/Knulli App Store.sh`, `packaging/magicx-zero-28/Knulli App Store.sh`.
- Why fragile: A crash can leave only part of the binary/catalogue pair replaced. A failed restore can leave an open transaction. Corrupt journals block recovery and require inspection.
- Safe modification: Existing contracts require launcher CLI recovery before the UI loads, service recovery before catalogue verification, atomic inode replacement, and snapshots retained after failed rollback. Recovery cannot restore journals that an older build already deleted.
- Test coverage: `internal/installer/selfupdate_test.go`, `internal/safefs/guard_test.go`, and `internal/safefs/files_test.go` cover interrupted replacement, retained failed-rollback snapshots, and file replacement. These tests do not prove recovery after every hardware power failure.

**Gabagool owns global GUI state:**
- Files: `internal/sdlui/run.go`, `internal/sdlui/events.go`, `internal/input/mapping.go`, `docs/adr/0010-gabagool-knulli-frontend.md`.
- Why fragile: Gabagool initializes global SDL, font, and controller state and can exit on startup errors. Its stock input behavior differs from the Store's controls. The adapter uploads the responsive Library bitmap through SDL.
- Safe modification: The current contract permits one frontend per process. It disables toolkit input sources, rejects `INPUT_CAPTURE`, sends toolkit logs to stderr, keeps semantic mapping in `input`, and probes for a software-renderer fallback. Stock screen loops are not used.
- Test coverage: `internal/sdlui/run_test.go` checks the input contract and capture rejection. Tagged draw tests and walkthrough captures check the renderer path. The Option B TSP report now records native display, injected control, lifecycle and ES-return checks. It does not prove physical presses, audio or MagicX behavior. See `docs/real-device-tests.md`.

**Menu edits serialize the whole XML document:**
- Files: `internal/gamelist/xml.go`, `internal/gamelist/menu_test.go`.
- Why fragile: The XML model keeps elements, attributes, and text, but not comments, processing instructions, original whitespace, or CDATA boundaries. A changed entry can therefore change unrelated formatting.
- Safe modification: Exact launch-path matching and ownership checks are the contract. Filling blank metadata in an existing entry does not grant ownership. Unchanged installer-created entries alone can be removed.
- Test coverage: Tests check ownership and metadata retention. Byte-for-byte XML round trips are not promised.

**Reviewed executable transformations depend on exact bytes:**
- Files: `internal/installer/stage.go`, `catalogue/packages/app.romm.grout.json`, `catalogue/packages/io.github.tomtombombadil.pocketcurator.json`, `internal/installer/installer_test.go`.
- Why fragile: Grout's updater-URL patch and PocketCurator's launcher-path patch apply to pinned archive bytes. A later upstream update can replace a patched file, especially for an upstream-managed app.
- Safe modification: Current staging checks patch inputs, offsets, and resulting hashes before transaction writes. File mappings change installation paths after patch validation.
- Test coverage: The PocketCurator regression models PortMaster clearing `SCRIPT_DIR` and checks Install and Repair with preserved settings. It is not native lifecycle evidence.

## Scaling Limits

**Package size and catalogue processing:**
- Current capacity: The catalogue has seven experimental package manifests. `manifest.MaxPackageBytes` is 512 MiB for compressed and installed-size policy. Adoption inventory shares that byte limit.
- Limit: Larger packages are rejected. Catalogue loading and service item generation process the complete index; no paged remote catalogue is implemented.
- Scaling path: No higher size limit or catalogue redesign is proposed. Sources: `catalogue/packages/`, `internal/manifest/manifest.go`, `internal/installer/adoption.go`, `internal/appstore/service.go`.

**Logs and retained evidence:**
- Current capacity: The active diagnostic log is limited to 512 KiB, with one rotated copy.
- Limit: Older runtime evidence is discarded. Logs are not an unlimited audit history.
- Scaling path: No retention change is proposed. Source: `internal/diagnostics/log.go`.

## Dependencies at Risk

**Gabagool, Go SDL bindings, and device libraries:**
- Risk: The GUI now depends on gabagool/v2 v2.26.6, go-sdl2 v0.4.40, SDL2_image, and SDL2_ttf in addition to SDL2. No specific dependency vulnerability was established here.
- Impact: A successful desktop run does not prove the device's loader, libraries, display, or controller behavior. The backend and static CLI do not require the GUI stack.
- Migration plan: ADR 0010 fixes the current boundary; it does not authorize firmware library replacement. Device builds statically link SDL2_gfx, enforce an exact shared-library allowlist, and reject required glibc versions above 2.34. Sources: `go.mod`, `scripts/device-build.sh`, `docs/adr/0010-gabagool-knulli-frontend.md`.

**Linux filesystem APIs:**
- Risk: Manager locking and free-space checks use platform system calls. Native Windows support is not implemented.
- Impact: Portability of pure Go model tests does not establish portability of installation behavior.
- Migration plan: No platform port is proposed. Sources: `internal/installer/stage.go`, `internal/safefs/files.go`.

## Missing Critical Features

**Independent remote catalogue distribution:**
- Problem: There is no stable-key remote catalogue refresh. Store self-update replaces binaries and their bundled catalogue together.
- Blocks: A separately published index cannot be assumed to work with an existing device binary. Sources: `scripts/device-build.sh`, `internal/installer/selfupdate.go`, `docs/adr/0008-catalog-index-signing.md`.

**Download resume and dedicated operation cancellation:**
- Problem: Downloads restart rather than resume. The GUI has no dedicated user-facing Cancel action. Download and extraction use context cancellation; closing the GUI cancels its run context.
- Blocks: A user cannot pause and resume a large download or invoke a separate in-app cancellation flow. Sources: `internal/installer/download.go`, `internal/archive/archive.go`, `internal/sdlui/run.go`, `internal/ui/model.go`.

**Other firmware profiles:**
- Problem: Only KNULLI is implemented. The extension boundaries are not implementations for other firmware.
- Blocks: An ARM64 build alone cannot establish support for another CFW. Sources: `docs/adr/0010-gabagool-knulli-frontend.md`, `internal/platform/platform.go`.

## Test Coverage Gaps

**Exact device and lifecycle evidence:**
- Evidence and gaps: The owner confirmation covers the six unchanged current versions and historical Picker 0.2.0 on Smart Pro, Knulli AArch64 at 1280×720 only. Picker 0.2.1 has no hardware evidence and is UNTESTED everywhere. The confirmation does not supply an exact firmware build identifier or separate lifecycle, physical button, audio, power-loss, or Store idle-redraw results. MagicX and other matrices remain untested. The historical Option B TSP report separately covers native Picker 0.2.0 Install, Update and Repair plus preserved state, filters, injected navigation and ES return; it does not prove stable Picker gameplay/hotkey return or new-renderer Uninstall/Force reinstall. Do not infer these details from the newer package confirmation.
- Files: `docs/real-device-tests.md`, `docs/adr/0010-gabagool-knulli-frontend.md`, `CONTEXT.md`, `internal/platform/platform.go`.
- Risk: A desktop fixture or a report for another app version could be mistaken for hardware proof. Exact package version, firmware, architecture, device, and resolution control the tested badge. Missing evidence does not block an otherwise compatible experimental installation.
- Priority: High for device release claims. No new hardware run was performed for this map.

**SDL paths are outside default coverage:**
- What's not tested: The default coverage gate excludes SDL-tagged runtime and drawing. CI runs tagged race tests and offline keyboard walkthroughs separately, not an automated physical GameController or device lifecycle test.
- Files: `Makefile`, `.github/workflows/check.yml`, `internal/sdlui/run_test.go`, `scripts/desktop-walk.sh`.
- Risk: A default coverage percentage alone cannot establish GUI or device readiness.
- Priority: High when changing GUI lifetime, input, packaging, or rendering.

**Real power-loss and filesystem failure behavior:**
- What's not tested: Temp-root recovery tests do not establish every syscall failure or storage-controller durability behavior on a handheld.
- Files: `internal/safefs/guard_test.go`, `internal/safefs/files_test.go`, `internal/installer/selfupdate_test.go`.
- Risk: A passing simulated crash test is narrower evidence than a real interrupted-write device test. Failed rollback retention and corrupt-journal refusal are tested, not absent features.
- Priority: High for recovery claims. No power-loss test was performed here.

---

*Concerns audit: 2026-10-25*
