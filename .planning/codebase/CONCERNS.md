# Codebase Concerns

**Analysis Date:** 2026-10-21

This map separates confirmed implementation limits from possible future work. It is not a claim that every failure path has been tested.

## Tech Debt

**Local checks differ from CI:**
- Issue: `make check` rewrites formatting and runs vet/race tests, but not Staticcheck, coverage, tagged tests, cross-builds, or walkthroughs. CI has a rejecting format check and those extra gates.
- Files: `Makefile`, `.github/workflows/check.yml`, `prek.toml`.
- Impact: A successful local shortcut is not proof of a green CI run.
- Fix approach: Use the documented full check set before delivery; changing the task contract is not required for this refresh.

**Analyzer/toolchain pins need coordinated updates:**
- Issue: Test/walkthrough CI pins Go 1.27.1 for Staticcheck v0.8.1 export-data compatibility; `.agents/setup` accepts an already installed newer Go, and device/weekly jobs use the 1.27 series.
- Files: `.agents/setup`, `.github/workflows/check.yml`, `.github/workflows/catalogue-updates.yml`, `prek.toml`.
- Impact: A developer with a newer patch version can encounter analyzer failure not seen by pinned test CI.
- Fix approach: Keep toolchain/analyzer changes coordinated. Do not silently downgrade a user's existing toolchain.

## Known Bugs

**Abandoned staging accumulation — fixed in this refresh:**
- Prior behavior: A killed process left `work-*` or `store-update-*` directories; journal recovery did not reclaim them.
- Files: `internal/installer/stage.go`, `internal/installer/selfupdate_test.go`.
- Resolution: After journal recovery, the manager lock holder removes only real staging directories with those prefixes. It leaves symlinks, originals, recovery backups, and live operations alone.
- Regression: `TestRecoveryDiscardsOnlyAbandonedStaging` fails without cleanup and checks repeated recovery, lock contention, and unrelated-data preservation.

**Incomplete glibc ABI gate — fixed in this refresh:**
- Prior behavior: Finding a GLIBC_2.34 requirement did not exclude a concurrent GLIBC_2.35 or newer requirement.
- File: `scripts/device-build.sh`.
- Resolution: Inspect the highest numeric required GLIBC version; reject missing version evidence or a maximum above 2.34. Older requirements remain allowed.
- Verification: Exercise the actual shell gate with version-info fixtures on both sides of the boundary; a real cross-build still depends on the target toolchain.

## Security Considerations

**Catalogue trust is per build:**
- Risk: Each device artifact gets a fresh signing key. It cannot accept an independently updated catalogue signed by another build's key.
- Files: `scripts/device-build.sh`, `internal/catalog/catalog.go`, `internal/catalog/sign.go`.
- Current mitigation: Ship key-bearing binaries and the catalogue/signature together. Startup recovery precedes GUI catalogue loading.
- Recommendations: A long-lived signing key/rotation policy requires an explicit operational decision, not a new secret invented during a map refresh. Unsigned local builds intentionally skip signature verification.

**Store self-update uses the GitHub HTTPS trust boundary:**
- Risk: The offer comes from the first matching official target asset in the first 20 releases, including prereleases; it is not an independent signed update feed or a monotonic version comparison.
- Files: `internal/installer/selfupdate.go`, `internal/appstore/selfupdate.go`.
- Current mitigation: Exact official tagged URL, GitHub SHA-256 digest, exact size, safe extraction, exact seven-file inventory, explicit confirmation, transactional replacement, restart requirement.
- Recommendations: Preserve this stated experimental-release policy unless product requirements change. No API retry/backoff or authentication is implemented for this path; do not confuse it with the maintenance checker.

**Reviewed transformations and write policy remain security boundaries:**
- Risk: Binary patches, file mappings, and declared write paths control executable bytes and filesystem destinations.
- Files: `internal/manifest/manifest.go`, `internal/installer/stage.go`, `internal/installer/files.go`, `internal/safefs/guard.go`.
- Current mitigation: Pinned patch input/output, safe relative mappings with collision/missing-source checks, symmetric manager-state overlap rejection, guarded `/userdata` writes, and journaled mutations.
- Recommendations: Preserve these checks and adversarial tests; do not treat relaxed device scope as relaxed archive or runtime policy.

**Redaction has a defined scope:**
- Risk: Arbitrary opaque secrets or credentials embedded in URL paths are not guaranteed to be removed.
- File: `internal/diagnostics/log.go`.
- Current mitigation: Credential-key masking, URL userinfo/query/fragment removal, terminal-control stripping, redaction on write/reopen/export, and private export modes.
- Recommendations: Do not log secrets; review exports before sharing. Heuristic entropy scrubbing is not implemented.

## Performance Bottlenecks

**Cached health is not live filesystem monitoring:**
- Problem: A full check hashes immutable managed files. Cache hits are keyed on recorded state, so outside edits can remain unseen until invalidation or service restart.
- Files: `internal/installer/status.go`, `internal/installer/cache.go`, `internal/appstore/service.go`.
- Cause: Deliberate cache tradeoff, documented in source; installer mutations invalidate the affected package.
- Improvement path: If live damage detection is required, define a recheck policy rather than claiming every catalogue read re-hashes files.

**Space and sequential work:**
- Problem: Normal staging reserves release bytes + 3 × installed bytes + 2 × existing bytes; status/index work is sequential.
- Files: `internal/installer/stage.go`, `internal/appstore/service.go`.
- Cause: Rollback safety and a small catalogue.
- Improvement path: Measure on target hardware before weakening safety margins or adding parallelism.

## Fragile Areas

**Recovery and self-replacement:**
- Files: `internal/safefs/transaction.go`, `internal/safefs/files.go`, `internal/installer/selfupdate.go`, `packaging/magicx-zero-28/Knulli App Store.sh`.
- Why fragile: A partially replaced key-bearing binary must not load the wrong catalogue; rollback may itself fail.
- Safe modification: Keep launcher CLI recovery before GUI loading, durable snapshots, atomic inode replacement, and installed-state-last ordering for package commits.
- Test coverage: Open-journal recovery, failed rollback retention, running executable replacement, and Store-before-catalogue recovery have cases in `internal/safefs/guard_test.go`, `internal/installer/selfupdate_test.go`, and `internal/appstore/service_test.go`. Do not claim every I/O failure is covered.

**Whole-document XML serialization:**
- Files: `internal/gamelist/xml.go`, `internal/gamelist/menu_test.go`.
- Why fragile: Edits reserialize the document. Comments, processing instructions, original whitespace, and CDATA boundaries are not preserved, although modeled elements/attributes and existing metadata values are.
- Safe modification: Keep exact-path matching, fill only blank metadata on unowned entries, and never claim/remove those entries. Formatting preservation would need a separate parser/editing contract, not an incidental rewrite.
- Test coverage: Ownership and metadata preservation are tested; byte-for-byte XML round trips are not the contract.

**SDL and controller boundary:**
- Files: `internal/input/mapping.go`, `internal/sdlui/events.go`, `internal/sdlui/run.go`, `internal/sdlui/draw.go`.
- Why fragile: Layout bindings and labels must agree without double-swapping events; desktop SDL is not device SDL.
- Safe modification: Test Knulli/Xbox/desktop mappings and real hardware. Calibration/persisted overrides no longer exist and must not be cited as a fallback.
- Test coverage: Portable routing/input cases plus tagged draw tests and desktop walkthroughs, not an automated hardware test.

## Scaling Limits

- **Catalogue:** Six manifests (four experimental, two candidate) plus an optional synthetic Store card. `Items` derives all entries before filtering; index loading is whole-file (`catalogue/packages/`, `internal/appstore/service.go`).
- **Package bytes:** `manifest.MaxPackageBytes` is the shared 512 MiB cap for compressed release, installed-size policy, and adoption inventory; archive extraction receives an explicit budget (`internal/manifest/manifest.go`, `internal/archive/archive.go`, `internal/installer/adoption.go`). The cap is not duplicated independent policy.
- **Logs:** 512 KiB active plus one rotation; older evidence is discarded (`internal/diagnostics/log.go`).
- **Update lookup:** Store checks only 20 release records, not all release history (`internal/installer/selfupdate.go`).

## Dependencies at Risk

- **`golang.org/x/image v0.47.0`:** Only direct module dependency; affects GUI scaling/text. Renovate tracks updates (`go.mod`, `renovate.json`, `internal/sdlui/draw.go`). No specific vulnerability is established by this refresh.
- **System ABI:** SDL2 and glibc on the device are outside the repository. The cross-build gate constrains required GLIBC versions but does not prove target controller behavior (`scripts/device-build.sh`, `docs/real-device-tests.md`).
- **POSIX syscalls:** Flock/statfs limit native portability; no Windows implementation exists (`internal/installer/stage.go`, `internal/safefs/files.go`).

## Missing Critical Features

- No standalone remote catalogue refresh with a stable project signing key. Store self-update now supplies a new bundled catalogue with its binaries, so “no update mechanism” is no longer accurate (`internal/installer/selfupdate.go`, `scripts/device-build.sh`).
- No resumable download or dedicated user-facing cancel action. Extraction honors context, and quitting cancels the GUI run (`internal/installer/download.go`, `internal/archive/archive.go`, `internal/sdlui/run.go`).
- No automated device E2E. Real GameController, `/userdata`, display, and SDL-loader evidence must come from hardware reports (`CONTEXT.md`, `docs/real-device-tests.md`).

## Test Coverage Gaps

- Default coverage does not include SDL-tagged drawing/runtime or the GUI entry point; CI separately runs tagged tests and walkthroughs (`Makefile`, `.github/workflows/check.yml`).
- `cmd/knulli-app-ui/main.go` has no dedicated tests for flags or canonical-location self-update wiring. Service/installer update behavior is tested below that boundary (`internal/appstore/service_test.go`, `internal/installer/selfupdate_test.go`).
- Exact syscall failures and real power-loss durability are not all established by temp-root tests. Recovery tests do check open journals and retained failed-rollback snapshots (`internal/safefs/guard_test.go`).
- Percentages from the previous map were stale and internally inconsistent. Recompute with `make cover` instead of copying them forward (`TESTING.md`).

---

*Concerns audit: 2026-10-21*
