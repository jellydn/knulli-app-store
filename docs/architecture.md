# Architecture

The installer owns policy and side effects. Every front end calls the same Go service methods and never extracts archives, edits XML, or runs shell commands itself.

```text
manifest JSON ──▶ strict validation ──▶ compatibility check
                                              │
                                              ▼
signed index                           HTTPS download
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
                                      ├─ owned gamelist.xml entry
                                      ├─ original backups
                                      └─ installed state
```

## Modules

- `internal/manifest`: v1 data contract and semantic policy not expressible cleanly in JSON Schema.
- `internal/catalog`: deterministic validation, sorting, manifest hashing, and ed25519 index signatures.
- `internal/platform`: platform detection and compatibility checks.
- `internal/diagnostics`: timestamped, redacted, size-bounded event logs and user-requested diagnostic exports.
- `internal/archive`: bounded ZIP and `tar.gz` extraction into staging.
- `internal/safefs`: allowed-path resolution, symlink rejection, atomic files, free-space checks, and rollback snapshots.
- `internal/installer`: download, lifecycle operations, installed state, backups, XML menu integration, and the committed game-list reload request.
- `internal/updatecheck`: read-only upstream release metadata checks and the JSON or Markdown review report.
- `cmd/knulli-app`: a thin installer command-line adapter.
- `cmd/knulli-app-ui`: the SDL2 GUI entry point behind the `sdl` build tag.
- `cmd/check-updates`: a thin adapter that writes the catalogue update report.

## Lifecycle invariants

An archive is fully downloaded, hashed, and staged before a destination file changes. Every destination mutation enters the transaction and a durable journal before it happens. Installed state is the last file written. A failed step restores snapshots in reverse order. The next locked start rolls back an open journal and discards a committed leftover directory.

Update and repair share the same safe application path. Update can replace the manifest version and remove stale owned files. Repair reinstalls the declared release. Both leave existing declared configuration unchanged. Uninstall removes owned files, leaves preserved configuration, and restores files that existed before first ownership. A pre-existing copy is offered as **Manage existing**; a destination directory that holds no files is absent, not a copy. Adoption inventories it without reinstalling, records exact release matches, marks uncertain files unmanaged, and backs them up before later replacement.

Menu ownership is separate from package-file ownership. The installer adds an entry only when the exact launch path is absent. It removes only one unchanged entry that it created. A shared, pre-existing, or modified entry remains as an orphan rather than being deleted. After a committed menu change, the manager asks Knulli to queue a game-list reload through `GET http://127.0.0.1:1234/reloadgames`. A successful response means accepted, not completed. If the loopback request fails, the operation remains committed and the UI reports **Restart required**.

Installed state records the mode observed after the destination copy. Health checks hash every immutable managed file and require the recorded mode to keep its owner read, write, and execute permissions; a wider mode applied by the destination filesystem is accepted, because that filesystem owns the mode bits. Runtime files outside the release inventory do not become manager-owned and do not make a package unhealthy. A failed check has a structured path, check type, expected value, and actual value for the GUI and redacted log.

## SDL2 boundary

The GUI targets the TrimUI Smart Pro 1280×720 and MagicX Zero 28 640×480 displays. `internal/ui` is a pure-Go catalogue state machine that depends on the `appstore.Backend` interface. `internal/input` gates first-run catalogue access and owns semantic mappings, assignment review, preview, controller identity, and atomic persistence. `internal/appstore` validates the catalogue and calls `installer.Manager`. `internal/sdlui` adapts SDL events and renders setup as a dedicated screen. Tests use an in-memory fake, while the production adapter keeps every package side effect in the installer.

The SDL adapter is behind the `sdl` build tag, so the CLI stays static and CGO-free. A small local cgo wrapper imports only the SDL 2.0 functions in use. The GUI discovers SDL GameControllers and consumes Knulli's generated mapping before any reviewed fallback. Saved assignments bind semantic actions to SDL GameController buttons for one device/controller identity; the app does not read raw evdev nodes.
