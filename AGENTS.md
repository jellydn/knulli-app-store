# AGENTS.md

Go 1.27 module `github.com/jellydn/knulli-app-store`. Safe package manager for Knulli handhelds (TrimUI Smart Pro 1280×720, MagicX Zero 28 640×480).

## Commands

```sh
make check      # fmt + vet (both tag sets) + go test -race ./... — fast local loop
make build      # CGO-free CLI -> build/knulli-app
make catalogue  # deterministic index -> build/catalog-index.json
git diff --exit-code -- build/catalog-index.json  # required after catalogue-affecting changes
make cover      # coverage gate: every prod-Go package needs a test file AND total >= 70%
go test -tags sdl ./...  # SDL GUI compile + layout tests; needs libsdl2-dev (brew install sdl2 / apt install libsdl2-dev)
make gui        # desktop GUI with keyboard input + scratch fixture root (no device)
make walkthrough            # offline GUI evidence; add WALK_FLAGS=--install for network-backed lifecycle flows
```

- Local-only `prek.toml` mirrors CI (`check.yml`): fast hooks on pre-commit, `go test -race` + `staticcheck` (both tag sets) on pre-push. Run the pre-push set before opening a PR.
- `build/` is gitignored — never commit it; review the generated index locally instead. Device CI signs `catalog-index.json` and bakes the public key into that build.
- Device cross-compile: `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o build/knulli-app-aarch64 ./cmd/knulli-app`. Full per-device path is `scripts/device-build.sh <target> <version> dist`.

## Architecture

- Installer owns policy and side effects. Frontends (`cmd/knulli-app`, `cmd/knulli-app-ui`) call `installer.Manager` / `appstore.Backend` and never extract archives, edit XML, or shell out.
- Module map: `manifest` (contract + policy) → `catalog` (deterministic index, ed25519 sig) → `platform` (detection/compat) → `archive` (bounded ZIP/tar.gz staging) → `safefs` (path policy, snapshots) → `installer` (lifecycle, state, gamelist) → `ui` (pure-Go state machine) / `input` (mappings) / `sdlui` (SDL render, behind `sdl` build tag so CLI stays static).
- Vocabulary in `CONTEXT.md` is normative (adopt = Manage existing, external = files not dirs, outcome = accepted-not-completed). Use its terms in code and docs.

## Hard constraints (do not break)

- Writes stay below `/userdata` only; reject symlinks in existing write-path components. Use `-root` flag to redirect into a temp fixture root in tests.
- Never run a remote install script; only copy regular files from a verified archive. Reject absolute paths, traversal, links, devices, non-regular entries.
- Release = pinned HTTPS URL + exact byte size + SHA-256; archives ≤ 512 MiB compressed and installed. Mutable/aliased assets are not releases.
- Transaction order: download → verify → stage → journal → mutate → installed-state-last. Roll back in reverse on failure.
- Compatibility failures must report raw value, normalized value, source, and full detected matrix.
- Device identity is per-device (never infer from SoC family); resolution is explicit WIDTHxHEIGHT, never assumed. Knulli identity = `OS_NAME="knulli"` in `/etc/os-release`, version from `/usr/share/knulli/knulli.version`.
- Menu ownership: add only an absent exact launch path, remove only an unchanged entry this installer created. Game-list reload via loopback `GET 127.0.0.1:1234/reloadgames` means queued, not completed.

## Catalogue contributions

- One file per package in `catalogue/packages/`. Statuses: `candidate` (not actionable, no release/compat/install fields) → `installable` (review checklist passed) → `verified` (linked report on exact device/firmware/arch/resolution).
- Required evidence: license, immutable asset, publisher checksum, Knulli compat, install paths, preserved files, network use, every write path. A README claim is not evidence. No shell-piped installers, no unrestricted writes, no ROM-download sources.
- Read-only checker: `go run ./cmd/check-updates -output build/catalogue-update-report` writes `.json`+`.md` for manual review; never edits manifests. `GITHUB_TOKEN` optional locally, never in reports.

## Code conventions

- New filesystem behavior needs a temp-root integration test; new archive behavior needs an adversarial fixture where an unsafe impl would differ.
- Comments explain design reasons the code cannot show, not what the code does.
- Keep `internal/installer` independent of presentation code; GUI tests use an in-memory fake backend.
