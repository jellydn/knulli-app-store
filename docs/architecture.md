# Architecture

The installer owns policy and side effects. A future UI will call the same Go service methods and will not extract archives, edit XML, or run shell commands itself.

```text
manifest JSON ──▶ strict validation ──▶ compatibility check
                                              │
                                              ▼
signed index (future)                  HTTPS download
                                              │
                                              ▼
                                      size + SHA-256
                                              │
                                              ▼
                                      safe staging area
                                              │
                                              ▼
                                      write transaction
                                      ├─ package files
                                      ├─ gamelist.xml
                                      ├─ original backups
                                      └─ installed state
```

## Modules

- `internal/manifest`: v1 data contract and semantic policy not expressible cleanly in JSON Schema.
- `internal/catalog`: deterministic validation, sorting, and manifest hashing for a signable index.
- `internal/platform`: platform detection and compatibility checks.
- `internal/diagnostics`: timestamped, redacted, size-bounded event logs and user-requested diagnostic exports.
- `internal/archive`: bounded ZIP and `tar.gz` extraction into staging.
- `internal/safefs`: allowed-path resolution, symlink rejection, atomic files, free-space checks, and rollback snapshots.
- `internal/installer`: download, lifecycle operations, installed state, backups, and XML menu integration.
- `cmd/knulli-app`: a thin command-line adapter.

## Lifecycle invariants

An archive is fully downloaded, hashed, and staged before a destination file changes. Every destination mutation enters the transaction before it happens. Installed state is the last file written. A failed step restores snapshots in reverse order.

Update and repair share the same safe application path. Update can replace the manifest version and remove stale owned files. Repair reinstalls the declared release. Both leave existing declared configuration unchanged. Uninstall removes owned files, leaves preserved configuration, and restores files that existed before first ownership.

## SDL2 boundary

The GUI targets the TrimUI Smart Pro 1280×720 and MagicX Zero 28 640×480 displays. `internal/ui` is a pure-Go state machine that depends on the `appstore.Backend` interface. `internal/input` owns semantic mappings, calibration, controller identity, and atomic persistence. `internal/appstore` validates the catalogue and calls `installer.Manager`. `internal/sdlui` adapts SDL events and rendering. Tests use an in-memory fake, while the production adapter keeps every package side effect in the installer.

The SDL adapter is behind the `sdl` build tag, so the CLI stays static and CGO-free. A small local cgo wrapper imports only the SDL 2.0 functions in use. The GUI discovers SDL GameControllers and consumes Knulli's generated mapping before any reviewed fallback. Saved assignments bind semantic actions to SDL GameController buttons for one device/controller identity; the app does not read raw evdev nodes.
