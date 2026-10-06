# Codebase Concerns

**Analysis Date:** 2026-09-21

## Tech Debt

**`make check` silently rewrites instead of failing:**
- Issue: `check: fmt vet test` and `fmt` runs `gofmt -w` on whatever `gofmt -l` lists. A local `make check` therefore fixes formatting rather than reporting it, while CI fails on `test -z "$(gofmt -l .)"`.
- Files: `Makefile`, `.github/workflows/check.yml`
- Impact: Contributors on `make check` can still push formatting CI rejects; conversely a `make check` run mutates the working tree unexpectedly. Intentional for speed, but the divergence is a trap.
- Fix approach: Split a `fmt-check` target for the `check` dependency and keep `fmt` as the auto-fixer.

**Three functions named `wrap` / `actionName` doing ad-hoc ASCII work:**
- Issue: `wrap` is defined separately in `internal/ui/model.go` and `internal/input/session.go`; `internal/ui/model.go`'s `actionName` does byte arithmetic (`string(value[0]-'a'+'A')`).
- Files: `internal/ui/model.go`, `internal/input/session.go`
- Impact: Low — both copies are correct. But `actionName` panics-adjacent if an action string were ever empty or non-lowercase ASCII (it is guarded by an empty check, not by a case check).
- Fix approach: Nothing urgent; if a shared helper package ever appears, move `wrap` there.

**Time-sensitive status tables committed as prose:**
- Issue: `README.md` ("The catalogue has five packages", a per-package blocker table), `docs/catalogue-review-2026-09-15.md`, `docs/catalogue-review-2026-09-17.md`, and both `packaging/*/README.txt` files restate review status by hand.
- Files: `README.md`, `docs/catalogue-review-*.md`, `packaging/trimui-smart-pro/README.txt`, `packaging/magicx-zero-28/README.txt`
- Impact: The catalogue is the source of truth and can drift from all six documents silently; the weekly `catalogue-updates.yml` report is read-only and will not correct them.
- Fix approach: Generate the table from the index in a docs check, or reduce the prose to a link plus a date.

**Superseded ADR still marked accepted:**
- Issue: ADR-0006 is titled "Target Go 1.19, a static aarch64 CLI, and Knulli-owned firmware files" and ADR-0007 raises the baseline to Go 1.27. Both are listed as `accepted` in `docs/adr/README.md` with no supersession marker.
- Files: `docs/adr/0006-go119-static-cli-knulli-detection.md`, `docs/adr/0007-go127-language-baseline.md`, `docs/adr/README.md`
- Impact: A reader following ADR-0006 alone would target the wrong language version. The README even notes "New records use the next number and keep the Context / Decision / Consequences shape" but defines no status vocabulary beyond `accepted`.
- Fix approach: Add a `superseded by NNNN` status column value and mark 0006 accordingly.

**Thirteen `//lint:ignore U1000` suppressions in one file:**
- Issue: `internal/sdlui/layout.go` constants are only referenced by sdl-tagged draw code, so `staticcheck` on the untagged build reports them unused and each is suppressed by hand.
- Files: `internal/sdlui/layout.go` (lines 15, 17, 34, 38, 43, 45, 47, 49, 51, 53, 57, 59, 181)
- Impact: A new constant used only by tagged code needs a matching suppression or the untagged gate fails — a papercut, and one that could mask a genuinely dead constant if a suppression were ever added wrongly.
- Fix approach: Move the tagged-only tokens into a file that is itself sdl-tagged, or tag the constants block.

## Known Gaps

**A killed process leaves its staging directory behind:**
- Problem: `installer.stage` creates `work-*` under the manager directory, and only `stagedRelease.release` removes it. Nothing sweeps a stale `work-*` at startup, so a process that dies mid-install — killed, power lost, or quit before the cleanup runs — leaves the downloaded archive and its extracted tree on the card.
- Files: `internal/installer/stage.go` (`stage`, `stagedRelease.release`), `internal/installer/installer.go` (`apply`)
- Trigger: `sdlui.Run` cancels its context when the run returns, so quitting the GUI during a busy operation now stops extraction promptly — and can exit the process before the deferred cleanup runs.
- Impact: Up to 512 MiB of stale staging per interrupted install, on a handheld where that is a large share of internal storage. Correctness is unaffected: nothing outside `work-*` was written, so the destination and the installed state are untouched.
- Fix approach: `acquireManager` already holds the exclusive lock, so no live operation can own a `work-*` directory while it runs. Sweep stale `work-*` directories there, beside the existing `safefs.Recover` call.

**Whole-file XML rewrite drops anything the parser does not model:**
- Problem: `encodeGamelist` serializes the entire parsed tree with a fresh `xml.Header`, 2-space indentation, and no preservation of comments, processing instructions, DTD, CDATA boundaries, or original whitespace. `decodeNode` ignores `xml.Comment` outright.
- Files: `internal/gamelist/xml.go` (`parseGamelist`, `decodeNode`, `encodeGamelist`)
- Trigger: any add, replace, or remove touching a gamelist file — one entry added to a large hand-curated `gamelist.xml` reflows the whole document and loses its comments.
- Impact: The semantics of the XML are preserved (element names, attributes, text), but the file's original formatting and non-element content are not. EmulationStation tolerates this; a user comparing a backup would not.
- Fix approach: Surgical text-level edit for the entry, or at minimum preserve comments by capturing `xml.Comment` tokens in `decodeNode` and re-emitting them.

**Unguarded zero-value map read for the quit chord:**
- Problem: `Session.HandleButton` tests `button == session.activeMapping()[Diagnostics]` directly. A Go map read of an absent key yields `0`, which is SDL's south face button.
- Files: `internal/input/session.go`, `internal/input/mapping.go`
- Impact: Latent. `Mapping.Validate` requires `Diagnostics` for every saved mapping, so in practice the key is always present. But `Mapping.Action` in the same package carries an explicit comment warning about exactly this trap and guards it with `ok`; this site does not.
- Fix approach: Mirror `Action`'s guard: read the button with `, ok` and bail when absent.

**Reviewed fallback mapping exists for only one device:**
- Problem: `input.Profile` returns a populated `Fallback: AutoMapping()` for `trimui-smart-pro`, but for `magicx-zero-28` it returns a profile with no `Fallback` at all (the `Evidence` string says "raw controls unavailable"). `platform.DisplayName` likewise hardcodes exactly two device names.
- Files: `internal/input/mapping.go`, `internal/platform/platform.go`
- Impact: A MagicX install carries no reviewed layout. Runtime handling is unaffected — `NewSession` gives every session SDL's canonical logical layout as its detected mapping and first launch walks the user through it — but the reviewed record the Smart Pro has does not exist here, and `TestDeviceProfilesDoNotInventMagicXFallback` fails if one is claimed without a report behind it.
- Fix approach: `docs/magicx-zero-28.md` (Recording a reviewed controller layout) now names the required evidence and the four files it must land in. Note that the confirmed MagicX diagnostic establishes the controller GUID and name, not the button layout — and `Detected` is never populated from SDL's own mapping, so `AutoMapping()` is the canonical layout rather than a per-device reading. A report must therefore confirm that this pad's buttons land where a user expects before the profile gains a `Fallback`.

**Chunked downloads skip the `Content-Length` check:**
- Problem: `download` only compares `response.ContentLength` when it is `>= 0`.
- Files: `internal/installer/download.go`
- Impact: Low — the written byte count is compared to `release.Size` and the SHA-256 is verified afterwards, so integrity is not weakened; the early check is simply unavailable for chunked responses.
- Fix approach: None required; worth a comment stating that the byte-count and hash checks are the real gate.

## Security Considerations

**The manager-state write rule has to stay symmetric:**
- Risk: `manifest.validateInstallable` rejects any package `allowed_write_paths` entry that overlaps manager state. That check used to read `under(ManagerStatePath, allowed)`, which caught a write path equal to or *above* manager state but let a path *inside* it pass — so a manifest could declare its destination as `/userdata/system/knulli-app-store/installed/<id>.json` and have the installer copy a release over a record it owns. Writing the coverage case is what surfaced it.
- Files: `internal/manifest/manifest.go` (`overlaps`, `ManagerStatePath`), `internal/manifest/manifest_test.go` (`TestOverlapsCoversBothDirections`)
- Current mitigation: The rule is now bidirectional (`overlaps`), the path has one definition shared with the installer (`installer.managerPath` reads `manifest.ManagerStatePath`), and both directions plus the shared-prefix neighbour case are pinned by tests. No manifest in `catalogue/packages/` overlaps manager state either way, so tightening broke nothing.
- Recommendations: Keep the check symmetric. A future rule that narrows it to one direction reopens the same hole, and the manager-state constant must not be re-duplicated into the installer.

**Catalogue trust is per-build, not per-project:**
- Risk: `scripts/device-build.sh` generates a new ed25519 key for every build and compiles the matching public key into the binaries. As the script's own comment states, "a signed index cannot outlive the binary that embeds its key until a long-lived production key exists."
- Files: `scripts/device-build.sh`, `internal/catalog/sign.go` (`embeddedPublicKeyHex`)
- Current mitigation: The index, the signature, and both binaries ship in one zip built in one step, so the triple is internally consistent. `device-build.sh` fails the build if the derived public key is not 64 hex characters.
- Recommendations: Introduce a long-lived production signing key (stored outside the build, e.g. an Actions secret or an offline key) so a catalogue update could be shipped without rebuilding and re-signing the binaries. Until then, a user cannot verify that two releases were produced by the same project key.

**Unsigned fallback is silent on a device build:**
- Risk: `catalog.Load` calls `EmbeddedPublicKey()` and, when it returns `nil`, skips signature verification entirely with no warning.
- Files: `internal/catalog/catalog.go`, `internal/catalog/sign.go`
- Current mitigation: Correct and intentional for the desktop/local path where no key is injected. On device the key comes from `-ldflags` in `scripts/device-build.sh`, and the length check there guards the injection.
- Recommendations: Log an explicit `catalogue_unsigned` diagnostic event when no key is embedded, so a device build that lost its ldflags is visible in the log rather than indistinguishable from a healthy unsigned desktop run.

**`binary_patches` is a powerful primitive:**
- Risk: `applyBinaryPatches` rewrites bytes in a downloaded binary at a manifest-declared offset before commit. The result is verified against a manifest-declared final SHA-256, so the *output* is pinned — but the pinned value is authored by the same review that authors the patch.
- Files: `internal/installer/files.go` (`applyBinaryPatches`), `internal/manifest/manifest.go` (`BinaryPatch` validation)
- Current mitigation: Validation requires a safe relative path, a non-negative offset, equal-length non-empty `before`/`after` hex, and a 64-hex `sha256`; the patcher requires the `before` bytes to actually match at that offset; `catalogue/packages/app.romm.grout.json` uses it to disable Grout's self-updater, and the README states the transformed binary is verified.
- Recommendations: Treat any new `binary_patches` entry as equal in risk to a remote install script — the review checklist should require the reviewer to reproduce the patch independently. Consider requiring a `note`/evidence URL alongside each patch.

**Redactor is not a general secret scrubber:**
- Risk: `diagnostics.Redact` strips terminal control sequences, masks `token|password|passwd|secret|api[-_]?key|authorization` values, and removes userinfo, query, and fragment **only** from URLs that have one. A credential embedded in a URL path, or a bare opaque secret not preceded by one of those key names, passes through.
- Files: `internal/diagnostics/log.go`
- Current mitigation: All URLs the code logs are release URLs with no credentials; `Redact` runs at write time, again on export, and again over the existing log file at `diagnostics.Open`, so a format change cannot leak a token an older build wrote. Export bundles are written `0600`.
- Recommendations: Also redact long high-entropy path segments, or whitelist the known release-URL shape and redact everything else.

**POSIX-only syscalls bound the contributor matrix:**
- Risk: `syscall.Flock` (`internal/installer/stage.go`) and `syscall.Statfs` (`internal/safefs/files.go`) do not exist on Windows, and `syscall.Flock` semantics depend on the filesystem honouring advisory locks.
- Files: `internal/installer/stage.go`, `internal/safefs/files.go`
- Current mitigation: The target is Linux `aarch64`; macOS development works; CI runs `ubuntu-latest`. The `-root` flag is the documented way to run on another tree.
- Recommendations: State the POSIX requirement explicitly in `README.md` prerequisites (it currently says only Go and `libsdl2-dev`), and consider documenting that flock is advisory and assumes a single-writer card.

**`-root` points the policy engine at an arbitrary tree:**
- Risk: `-root` redirects every device path, which is what makes temp-root testing possible but also lets a user point the installer at an untrusted tree. `AGENTS.md` and the README both warn that it "must not point at an untrusted tree with symlinked path components."
- Files: `internal/safefs/guard.go`, `cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`
- Current mitigation: `NewGuard` calls `filepath.EvalSymlinks` on the root so the root itself is resolved once; `rejectSymlinkParents` then rejects a symlink at any component *below* the root before resolving a target. Package `allowed_write_paths` are validated to live under `/userdata` and to exclude `/userdata/system/knulli-app-store`.
- Recommendations: None required beyond the existing warnings; this is a documented developer flag.

## Performance Bottlenecks

**Whole-catalogue health verification:**
- Problem: `Manager.Status` hashes every immutable managed file of a package and compares permission bits. `Service.Items` calls it for every catalogue entry on every load.
- Files: `internal/installer/status.go`, `internal/installer/cache.go`, `internal/appstore/service.go`
- Cause: Content hashes are deliberately trusted over size and mtime, which is what `TestHealthCheckHashesContentWhenSizeAndTimeMatch` pins down.
- Improvement path: Already mitigated by `StatusCache`, keyed on the SHA-256 of the canonical installed-state encoding, plus explicit `Invalidate` on every mutation. The remaining cost is one full re-hash of the operated package per operation, plus a first-read re-hash after any state change. A per-file mtime fast path would weaken the guarantee and should be avoided.

**Free-space estimate is deliberately conservative:**
- Problem: `stage` requires `release.Size + InstalledSize*3 + existingBytes*2`.
- Files: `internal/installer/stage.go`
- Cause: Staging needs the compressed archive, the extracted tree, the destination copy, and rollback snapshots simultaneously.
- Improvement path: Correct as written. On a nearly-full SD card it will refuse an install that might technically fit — the error message reports both required and available bytes, which is the right behaviour for a safety-first installer.

**Sequential catalogue work:**
- Problem: `Service.Items` iterates packages one at a time; downloads are single-stream.
- Files: `internal/appstore/service.go`, `internal/installer/download.go`
- Cause: Simplicity and deterministic logging; the catalogue is 5 packages.
- Improvement path: Not worth parallelizing at this size. Revisit if the catalogue reaches the low hundreds.

**State digest recomputed per status read:**
- Problem: `stateIdentity` re-marshals the whole installed-state JSON on every `Status` call to key the cache.
- Files: `internal/installer/cache.go`
- Cause: The digest must cover the full record so a forgotten invalidation cannot serve a stale result.
- Improvement path: Acceptable — the record is small (manifest plus file list). Only revisit if a package ever holds thousands of files.

## Fragile Areas

**`internal/installer` — every side effect in one package:**
- Files: `internal/installer/*.go` (~1,400 production lines across 12 files)
- Why fragile: It owns the lock, the transaction, the journal, downloads, extraction, adoption, health checks, retry records, and menu ownership. A change to any of those can affect all of them, and the ordering invariants (journal before mutation, installed state last, rollback in reverse) are load-bearing.
- Safe modification: Keep the file-per-concern split; add behavior next to the concern it touches; re-read the package doc comment at the top of `installer.go` before restructuring. Never introduce a destination mutation outside a `safefs.Transaction`.
- Test coverage: Good (79.4%, 46 test functions, plus a 1,421-line end-to-end test), but the rollback and journal-recovery *failure* branches are the thinnest.

**`internal/sdlui/draw.go` — largest file, least directly verifyable:**
- Files: `internal/sdlui/draw.go` (730 lines), `internal/sdlui/layout.go` (204), `internal/sdlui/run.go` (498)
- Why fragile: Rendering correctness cannot be asserted as easily as logic, and the whole file is sdl-tagged so it is outside the default `go test` run and outside `make cover`.
- Safe modification: Change geometry in `layout.go` only, keep `draw.go` free of literals, and re-run `make walkthrough` so every screen state is re-rendered to PNG evidence.
- Test coverage: `draw_test.go` (788 lines) and `layout_test.go`; 91.8% for the package under `-tags sdl`. Screens are additionally captured by CI's `gui-walkthrough` artifact.

**`internal/safefs` journal recovery:**
- Files: `internal/safefs/transaction.go`, `internal/safefs/guard.go`, `internal/safefs/files.go`
- Why fragile: It is the code that runs when everything else has already gone wrong, on a card that may have been yanked mid-write. A bug here either loses user files or leaves the journal unparseable. Coverage rose from 64.1% to 84.3%, but what remains uncovered is still this kind of code: the `Sync`/`Close`/`RemoveAll` failure branches, which need an injected I/O fault to reach.
- Safe modification: Any journal format change needs both a forward and a backward recovery test; `snapshotsFromRecord` already validates that every journal path stays inside the root and that backup names are single path components — preserve those checks.
- Test coverage gaps: Rollback ordering across many snapshots, and the failure-of-failure paths (a rollback that itself cannot complete). The journal rejection paths are now pinned: an unknown status stops both `Recover` and `PendingForPath`, and a backup name containing a separator is refused.

**`cmd/knulli-app` argument handling:**
- Files: `cmd/knulli-app/main.go` (248 lines), `cmd/knulli-app/main_test.go` (780 lines)
- Why fragile: The incomplete-detection gate is a four-condition disjunction — `Firmware`, `Device`, and `Resolution` empty, or a package naming a `MinimumVersion` with no detected version — and it is the only thing stopping an install against a half-detected platform. It sat at 4.7% coverage, so a dropped condition would have shipped silently.
- Safe modification: Exercise the CLI through `run(arguments)` and `runApply(operation, arguments, applyDeps)` directly with a temp root rather than shelling out; both are separated from `main` for exactly that, and `applyDeps` lets a test point a download at a loopback release server and a refresh at a loopback port instead of editing the code under test.
- Test coverage gaps: only `main`'s `os.Exit` shim, which needs a subprocess this repo does not otherwise use, and the two `writeNewFile` cleanup branches for a failing `Write`/`Close`, which need an injected I/O fault. Each of the four gate conditions now has its own case, verified by dropping each condition and watching its case fail.

**`internal/manifest.Validate` — the single policy gate:**
- Files: `internal/manifest/manifest.go` (414 lines), `internal/manifest/manifest_test.go` (608 lines)
- Why fragile: Every rule that decides whether a package is actionable lives here, and it is 65.9% covered. `hasVerifiedMatrixEvidence` — the strictest rule, requiring real-device-test evidence for every declared device × resolution — has no `verified` package in the catalogue to exercise it, so its happy path is untested against real data.
- Safe modification: Add a test table entry for every new rule; keep the accumulate-sort-join error shape so a reviewer sees all problems at once.
- Test coverage gaps: `hasVerifiedMatrixEvidence`'s positive path, `immutableReleaseURL` corners (e.g. a path that is a prefix match but not a release download), and the `binary_patches` validation surface.

**`cmd/knulli-app-ui` has no tests of its own:**
- Files: `cmd/knulli-app-ui/main.go` (97 lines, entirely sdl-tagged)
- Why fragile: Flag wiring, the default catalogue path next to the executable, and the `-shot-dir`-requires-`-keys` guard are untested. All real logic was deliberately pushed into `internal/sdlui`, so the untested surface is thin — but the `-shot-dir`/`-keys` invariant is only enforced there.
- Safe modification: Keep the file to wiring only; move any new decision into `internal/sdlui`.

**Exported injection seams on production types:**
- Files: `internal/installer/installer.go` (`Client`, `RefreshClient`, `RefreshURL`, `Now`, `AvailableBytes`), `internal/updatecheck/check.go` (`BaseURL`, `Token`, `Sleep`, `MaxRetries`)
- Why fragile: These exist only for tests, so they are public API that no production caller sets. A refactor could remove one and silently break a test's ability to simulate a failure mode.
- Safe modification: Change them only alongside the tests that depend on them, and keep the zero-value behavior production-safe (nil client, nil clock, default URL) — which the current code does.

## Scaling Limits

**Catalogue size:**
- Current capacity: 5 manifested packages; the index is a single JSON file parsed in full at every GUI start and on every `check-updates` run.
- Limit: `Service.Items` is O(packages) with a `Status` read and a destination walk each, and `Model` renders a fixed window of 5 rows (`listRows = (listBottom - listTop) / rowHeight = 5`). Interface usability degrades well before the code does.
- Scaling path: Tabs already narrow the list. The next lever is lazy per-row status, since `StatusCache` already keys on state identity.

**Size caps:**
- Current capacity: 512 MiB compressed and 512 MiB installed per release; 512 MiB per adoption inventory.
- Limit: Enforced independently in three places (see Tech Debt).
- Scaling path: Centralize the constant, then raise it only with a matching free-space and staging-timeout review.

**Logs:**
- Current capacity: 512 KiB active plus one 512 KiB rotation; diagnostic exports are unbounded text bundles written `0600`.
- Limit: A long install on a chatty device rotates rather than grows, so growth is bounded — but old evidence is discarded one rotation back.
- Scaling path: Already adequate. A second rotation would be a deliberate trade against SD-card churn.

## Dependencies at Risk

**`golang.org/x/image v0.46.0` (the only direct module dependency):**
- Risk: Low. It is a first-party Go module with a stable API; the project uses only `draw.NearestNeighbor` scaling and font raster drawing.
- Impact: A breaking change or an unmaintained release would affect `internal/sdlui/run.go` (canvas scaling) and `internal/sdlui/draw.go` (text), i.e. the GUI only — the CLI, catalogue, and installer would be unaffected.
- Migration plan: The scaling path is ~20 lines and could be replaced with `image/draw` or a hand-written nearest-neighbour scaler if needed. Renovate (`config:recommended`) tracks bumps.

**Device SDL2 ABI is not pinned by this repo:**
- Risk: `scripts/device-build.sh` asserts the GUI needs exactly two shared libraries and no glibc symbol newer than `GLIBC_2.34`, and greps for `libSDL2-2.0.so.0`. It does not pin the SDL2 minor version the Knulli image ships.
- Impact: A Knulli image that ships an SDL2 with a different controller-database version could change which `SDL_GAMECONTROLLERCONFIG` mappings are produced, altering detected bindings. `packaging/trimui-smart-pro/README.txt` explicitly warns users not to replace Knulli's SDL copy.
- Migration plan: The reviewed fallback mapping plus the first-run setup flow already absorb an unexpected SDL mapping. Recording the SDL2 version in the diagnostic export would make a future report actionable.

**`staticcheck@v0.8.1` pinned via `go run`:**
- Risk: None for reproducibility (it is pinned), but the pin lives in two places — `prek.toml` (two hooks) and `.github/workflows/check.yml` — and `go run` fetches on first use, so an offline pre-push fails.
- Impact: A local/CI analyzer-version skew if only one is bumped.
- Migration plan: Move the version into a `Makefile` variable and have both the workflow and the hook call it.

**Catalogue manifests as an external contract:**
- Risk: `schema/package-manifest-v1.schema.json` plus `manifest.Validate()` are versioned `v1`. Adding a required field breaks every existing manifest at load time.
- Impact: `catalog.Load` re-validates every entry, so a tightened rule fails the whole index — which is the intended fail-closed behaviour, but it means a policy change is a breaking change.
- Migration plan: `installer.migrateLegacyManifest` is the existing precedent for a backward-compatible field addition (it backfills `ABIs` and `Dependencies` when absent). Follow that pattern, or introduce `v2` with a dual-read path.

## Missing Critical Features

**No long-lived catalogue signing key:**
- Problem: Every device build mints a fresh ed25519 key (`scripts/device-build.sh`). There is no production key, no rotation policy, and no way to publish a catalogue update that an already-installed binary will accept.
- Blocks: Shipping a new or corrected package without cutting a whole new release; a user cannot distinguish two releases as the same publisher.

**No way to resume an interrupted download:**
- Problem: Extraction honours the operation context, but an interrupted download starts from zero and no partial archive is cached.
- Files: `internal/installer/download.go`, `internal/installer/cache.go`
- Blocks: Recovering partial progress across a network drop. An interrupted install still starts over. There is also no user-facing cancel affordance: cancellation is reachable only by quitting the GUI, whose run context cancels on return.

**No GUI retry for a contended manager lock:**
- Problem: `acquireLock` is `LOCK_EX|LOCK_NB` and returns the error string `"another package operation is active"` immediately. `RecoveryStatus` recognises that exact string to set `Active: true`, so the special case is handled — but only for the status path.
- Blocks: Two concurrent callers (e.g. the CLI and the GUI) cannot wait for each other; the loser sees a raw error.

**Catalogue is baked in at build time:**
- Problem: `cmd/knulli-app-ui` defaults `-catalog` to `catalog-index.json` next to the executable; nothing fetches a newer index.
- Blocks: Delivering a new catalogue package without a new release. This is partly by design (the index is signed together with the binary), but it is the practical consequence of the ephemeral-key decision above.

**No automated device E2E:**
- Problem: `CONTEXT.md` states plainly that a walkthrough "never exercises a real GameController, a real `/userdata` write, or the kernel's SDL loading."
- Blocks: Automated verification of controller detection, on-device write semantics, and SDL loading. `docs/real-device-tests.md` and manifest `review.evidence` entries are the only record, and the README notes full MagicX GUI testing is still incomplete.

## Test Coverage Gaps

Measured 2026-09-21 with `go test -covermode=atomic ./...`; total 84.2% against a 70% floor. The three lowest packages were raised in one pass: `internal/safefs` from 64.1% to 84.3%, `internal/manifest` from 65.9% to 98.4%, and `cmd/knulli-app` from 4.7% to 83.8%.

**`cmd/knulli-app` (83.8%, raised from 4.7%):**
- What's not tested: `main` itself, a four-line `os.Exit` shim reachable only by spawning a subprocess, and the `writeNewFile` `Write`/`Close` cleanup branches, which need an injected I/O fault. The `run` dispatch, every subcommand's flag parsing, `printGameListOutcome`, and all four conditions of the platform-incomplete gate now have cases.
- Files: `cmd/knulli-app/main.go`, `cmd/knulli-app/main_test.go`
- Risk: Low. This was the highest-value gap in the repo before the pass: the gate is a four-way disjunction, and nothing asserted its conditions. Removing any single condition now fails a named subtest.
- Priority: Low

**`internal/manifest` (98.4%, raised from 65.9%):**
- What's not tested: only the `json.Marshal` error returns in `Decode` and `Canonical`, which are unreachable for these types. Every validation rule, the symmetric manager-state check, `immutableReleaseURL`, `binary_patches`, `display_bounds`, `Canonical`, and `Load` now have cases.
- Files: `internal/manifest/manifest.go`, `internal/manifest/manifest_test.go`
- Risk: Low. This was the highest-value gap in the repo before the pass, because the rule gating the `verified` claim had no case for a second declared resolution or a mismatched evidence version.
- Priority: Low

**`internal/safefs` (84.3%, raised from 64.1%):**
- What's not tested: the `Sync`/`Close`/`RemoveAll` error returns and the `AvailableBytes` "no existing parent" branch, all of which need an injected syscall fault. `SHA256`, `AvailableBytes`, `Virtual`, the transaction `Copy`/`Chmod`/`Remove`/`Commit` API, the closed-transaction guards, and both journal rejection paths are now covered.
- Files: `internal/safefs/transaction.go`, `internal/safefs/files.go`, `internal/safefs/guard_test.go`, `internal/safefs/files_test.go`
- Risk: Medium. This code runs after something else already failed, so what remains untested is the failure-of-failure surface: a rollback that cannot itself complete, and an unwritable journal parent.
- Priority: Medium

**`internal/catalog` (67.5%):**
- What's not tested: `Write` error paths (temp-create, sync, rename), `writeSignature` failures, and `Load` rejection of an unsigned index when a key *is* embedded.
- Files: `internal/catalog/catalog.go`, `internal/catalog/sign.go`
- Risk: A broken signature-loading path would surface only on a device, where the key is embedded and the desktop tests never reach it.
- Priority: Medium

**`cmd/knulli-app-ui` (0 test functions):**
- What's not tested: flag wiring, the default catalogue path derivation, and the `-shot-dir` requires `-keys` guard.
- Files: `cmd/knulli-app-ui/main.go`
- Risk: Low — the file is deliberately wiring only, and `internal/sdlui` (91.8%) holds the logic. The `-shot-dir` guard is the one real invariant.
- Priority: Low

**SDL-tagged rendering paths:**
- What's not tested in the default run: `internal/sdlui/draw.go` and `run.go` are outside `go test ./...` and outside `make cover` entirely; they run only under `go test -tags sdl ./...`. CI does run that, but the coverage gate never sees them.
- Files: `internal/sdlui/draw.go`, `internal/sdlui/run.go`
- Risk: A layout regression is caught by tests and by walkthrough evidence, so the practical risk is confined to on-device rendering differences.
- Priority: Medium

**Health-check and recovery failure branches:**
- What's not tested: The "rollback also failed" composition path, some `recoveryBackupDirectory` failure paths, and the `AdoptionConflictError` trigger from an existing backup.
- Files: `internal/installer/commit.go`, `internal/installer/recovery.go`, `internal/installer/installer_test.go`
- Risk: These are the paths that must not themselves fail. They are partially covered by focused tests (`commit_test.go`, `cache_test.go`, `lifecycle_test.go`) but the failure-of-failure cases are the thinnest.
- Priority: Medium

---

*Concerns audit: 2026-09-21*
