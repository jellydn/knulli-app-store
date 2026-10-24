# How to add a package

## What

Add one reviewed manifest under `catalogue/packages/`. Give the package a stable reverse-domain ID and link its upstream project, license, maintainer, and evidence. A package stays `candidate` until every mandatory technical field is safe. Community approval does not make it installable or verified.

## Why

Package code runs with access to user files. A complete review protects ROMs, saves, configuration, storage, and the Knulli menu. Exact release metadata also makes an install repeatable and repairable.

## How

1. Open an issue with the project link, maintainer, purpose, target devices, and known risks.
2. Select a version-pinned release asset. Do not use `latest` or a branch archive. The Store never runs a remote installer. A self-extracting release can declare `release.base64_offset` for a reviewed base64 `tar.gz` payload; the full asset is size- and hash-verified before data-only decoding.
3. Record its exact compressed size, extracted size, SHA-256, supported ZIP or `tar.gz` format, license SPDX expression, and evidence URLs.
4. Inspect every archive entry. Reject absolute paths, traversal, links, devices, pipes, duplicate names, and undeclared launchers.
5. Record architecture, ABI, minimum glibc symbols, and every dynamic dependency. Experimental packages need no device or display claims. Add only evidence-based optional lists or bounds. Missing exact device reports show UNTESTED, not blocked. A shared chip does not prove compatibility.
6. Declare the narrowest install destination and allowed `/userdata` writes. List user data and configuration in preserve paths. List each launcher or binary that needs execute mode and explain why.
7. Review runtime writes, network use, self-updaters, uninstall behavior, menu integration, and rollback. Store operations stay inside checksum and transaction controls. An authorized upstream-managed experimental package can leave runtime setup and updates to the app, with the warnings and cleanup checks below.
8. Use the existing manifest, compatibility, archive and lifecycle tests. Add a focused regression only for new behavior they do not cover. Validate the actual release in a disposable fixture root; never run its installer or launcher there. Check install, repair, update, uninstall, preserved data and rollback when their layout changes.
9. Run:

   ```sh
   gofmt -w .
   go vet ./...
   go vet -tags sdl ./...
   go test -race ./...
   go test -tags sdl ./...
   make catalogue
   make catalogue-check
   ```

   `make catalogue-check` builds the index twice and requires the two builds to be byte-identical. Use it instead of a `git diff` on the generated index: `build/` is gitignored, so git reports that path as clean whatever it contains.

10. Open a pull request with the security review, archive inventory, exact commands, checksums, dependency output, screenshots at each target size, and this checklist.
11. Use `experimental` for a clear, authorized hardware test after all mandatory release, runtime and lifecycle controls pass. One maintainer can complete this review; a second review is required for `installable`. Record tester, date, firmware, device, architecture, resolution, package version, and exact lifecycle results when tests occur. Use `verified` only for the exact matrix proven on real hardware.

A `candidate` can retain validated `release` metadata without `compatibility` or `install`. This lets later review reuse pinned bytes without exposing any lifecycle action. To promote it, add the reviewed runtime and installation definitions and an experimental warning. Do not treat the candidate's checksum as proof that its runtime is safe.

## Upstream-managed installation

Use `install.upstream_managed: true` only with `experimental` status. Download directly from the publisher, verify the pinned bytes and extract without running installer scripts. Keep a private package destination, not the entire shared Tools or Ports directory. Document first-launch setup, runtime writes, runtime downloads and the app's updater: none is sandboxed or verified by the Store. Explain that Store Repair restores pinned code; users must close the app first. Preserve settings and never claim runtime-created files as Store-owned.

When a license cannot be verified, an upstream-managed experimental manifest may omit `license` only if its visible install warning discloses the license uncertainty. Do not mirror or redistribute that asset, invent an SPDX identifier, or promote it beyond experimental.

Declare `install.uninstall_checks` for shared hooks and active-service state. Each check has an absolute `path` below `/userdata` and a cleanup `message`. Without `contains`, an existing path blocks removal. With `contains`, the Store searches the file or directory tree for that case-insensitive text. A match, symlink, unreadable entry or file over 1 MiB blocks removal before package mutation. Checks are read-only; never run an upstream uninstall script or delete a shared hook. Check documented hook locations and warn about custom upstream path overrides. Retest these checks when upstream changes its hook layout.

For an update, repeat the full release, archive, dependency, security, lifecycle, and device review. Never copy values from the weekly report without verification. For removal or deprecation, remove the catalogue and product references, regenerate the index, and retain only generic orphan-state handling for old manager state.

## EmulationStation menu metadata

Declare `install.menu` with its shared `gamelist`, exact relative launcher `path`, `name`, `description`, and optional relative `image` path. Verify the image exists in the pinned archive. Menu paths and image paths are relative to the gamelist directory. Do not package a shared `gamelist.xml` over the user's file.

When the package must live outside Tools or Ports, `install.menu_command` can name the installed package launcher plus arguments. The Store writes a shell-quoted `exec` wrapper at the declared menu path, which must be separately allowed and outside the package destination. It never executes the wrapper during installation. The wrapper is tracked, backed up, rolled back, health-checked and removed like other managed files. Adoption preserves an existing wrapper that does not match.

Keep launchers at the system root or one directory below it. Knulli's default limited-depth scanner can ignore deeper launchers even when a gamelist entry exists. `install.file_mappings` can map extracted file paths to paths below the package destination. Mappings run after stripping and verified patches, before executable modes are applied. Missing sources, unsafe paths, and duplicate output paths are rejected. Review and patch any launcher-relative binary paths when relocating a launcher; move its matching `.keys` file too. Updates and Repair track the resulting paths, so obsolete managed files can be removed without claiming a shared directory.

The Store adds only an absent exact launcher path. It replaces or removes an entry only while its recorded metadata, including its image, is unchanged. For an existing exact path, it fills blank name, description, and image fields without taking ownership. Existing nonblank metadata, play history, attributes, and custom fields remain. Run Repair to fill an installed app's missing metadata; reopen the Store to fill its own. Refresh the game lists or restart EmulationStation if needed. No manual gamelist deletion is needed.
