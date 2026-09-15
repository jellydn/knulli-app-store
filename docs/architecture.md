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
- `internal/archive`: bounded ZIP and `tar.gz` extraction into staging.
- `internal/safefs`: allowed-path resolution, symlink rejection, atomic files, free-space checks, and rollback snapshots.
- `internal/installer`: download, lifecycle operations, installed state, backups, and XML menu integration.
- `cmd/knulli-app`: a thin command-line adapter.

## Lifecycle invariants

An archive is fully downloaded, hashed, and staged before a destination file changes. Every destination mutation enters the transaction before it happens. Installed state is the last file written. A failed step restores snapshots in reverse order.

Update and repair share the same safe application path. Update can replace the manifest version and remove stale owned files. Repair reinstalls the declared release. Both leave existing declared configuration unchanged. Uninstall removes owned files, leaves preserved configuration, and restores files that existed before first ownership.

## Future SDL2 boundary

The first UI target is one H700 reference resolution and controller-only input. The UI should be a separate command that depends on an interface with list, install, update, repair, and uninstall methods. The production adapter will use `installer.Manager`; tests will use an in-memory fake. SDL2 and controller mappings must stay behind that adapter so the CLI can remain a static, CGO-free recovery tool.

The SDL2 scaffold is deferred until the Knulli-provided SDL ABI and H700 controller mapping are confirmed on hardware. This avoids selecting a binding that weakens the current static build.
