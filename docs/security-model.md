# Security model

## Protected assets

- User configuration and ROM metadata
- Existing files at package destinations
- EmulationStation `gamelist.xml`
- Package state and original-file backups
- The integrity of installed upstream files

## Trust boundaries

Catalogue review decides whether metadata is actionable. HTTPS provides transport security, while the reviewed SHA-256 binds the archive bytes. Device builds also verify an ed25519 signature over `catalog-index.json`. A checksum supplied only beside a mutable asset is not enough evidence for catalogue promotion.

Upstream package code runs later when the user launches it. The installer does not sandbox that runtime. The `network` field tells users and future policy code whether the installed application needs network access; it does not grant or enforce an operating-system permission.

## Installer controls

- Reject candidate manifests and unknown manifest fields.
- Match firmware, minimum version, architecture, approved device, and resolution before download.
- Accept only HTTPS, version-pinned release URLs and require exact compressed size and SHA-256.
- Limit compressed and installed size to 512 MiB and check free space before download.
- Stage extraction before package writes.
- Reject absolute paths, traversal, duplicate archive names, links, devices, pipes, and other non-regular entries.
- Install archive files only below one declared destination.
- Permit menu changes only at a declared path below `/userdata`.
- Reject symlinks in every existing write-path component.
- Keep manager state outside package write policy.
- Serialize operations with a filesystem lock.
- Atomically replace regular files and installed state.
- Snapshot every changed file, persist a journal before destination mutation, and roll back a failed operation in reverse order. Recover an open journal on the next locked start.
- Keep a persistent backup when installation overwrites a file the package did not own.
- Show a detected external copy as the row state **EXTERNAL** with the action **Manage existing**, then inventory it before adoption. Detection and adoption both require at least one file at the destination, so an empty directory is never offered for adoption. Exact release-hash matches can become manager-owned; changed and unknown files are marked unmanaged and backed up because ownership is not proven.
- Preserve declared configuration during repair, update, and uninstall.
- Track installed paths, hashes, modes, and original-file backups.
- Persist the requested operation, detected install type, and safe retry target below manager state. Revalidate the current package state before showing a retry; never convert an absent fresh install into adoption.
- Redact URL credentials, query strings, fragments, and common secret fields from diagnostic logs. Cap the active log at 512 KiB and retain one rotated copy.
- Parse and rewrite `gamelist.xml` as XML while retaining unknown elements. Add only an absent exact launch path. Remove only an unchanged entry that manager state proves the installer created.
- Ask Knulli's loopback `/reloadgames` endpoint to queue a live refresh after a committed menu change. Report **Restart required** if the request is not accepted; never claim that the refresh completed.

## Non-goals and residual risks

- There is no package runtime sandbox.
- Device artifacts authenticate `catalog-index.json` with an ed25519 sidecar and a public key compiled into that build. A replaced index without a matching signature fails to load. A long-lived production key is still required if the index is shipped without a new binary.
- Root can change files below approved `/userdata` paths despite installer checks.
- Path checks do not defend against a hostile local process that races a checked directory into a symlink.
- A corrupt transaction journal blocks the next locked operation until the leftover directory is inspected.
- Empty directories can remain after rollback or uninstall. They are inert: external detection counts files, so a leftover skeleton does not block a fresh install or offer adoption.
- Device detection reads the Knulli board identifier (`/boot/boot/knulli.board`), then `/etc/knulli-device`, then Batocera's board file. An unrecognized board shows `UNKNOWN DEVICE` and is blocked until the operator supplies explicit flags.
- The GUI has one community-reported TrimUI Smart Pro test. The Grout result applies to 5.1.0.0, not the current 5.2.0.0 release. The report did not itemize lifecycle steps.
- PlayTime, Grout, and RetSend are broad experimental packages only when Knulli, AArch64, glibc, the required runtime libraries, a device identity, and reviewed display bounds match. Exact package-version device evidence controls the tested badge.
- Grout's reviewed staging patch replaces its only updater metadata URL with a reserved `.invalid` URL and verifies the transformed binary hash. A mismatch stops installation before any transaction write.
- Health checks use the observed destination mode after transactional copy or adoption mode normalization, and require that mode to keep the owner read, write, and execute permissions the installer set. A wider mode from a normalizing filesystem is not a health failure; a lost owner permission is. They compare the full SHA-256 of every manager-owned immutable file on every check. Unmanaged external files show their full reviewed and actual hashes for Repair instead of being treated as reviewed bytes.
- Force reinstall is offered only after a recoverable adoption ownership failure or a validated stale transaction journal. An active transaction blocks all recovery actions. Compatibility, free-space, signature, checksum, archive, path, and licensing failures stay blocked and show their exact reason; force mode cannot disable those checks.
- Before force reinstall changes a destination, the manager inventories and hashes every regular file. It preserves declared data and retains a complete timestamped backup plus original-path, hash, mode, and size records under `/userdata/system/knulli-app-store/recovery-backups/<package-id>/<timestamp>/`. A failed operation removes its incomplete retained backup after journal rollback restores the pre-operation files.
- Diagnostic export contains detected platform metadata, public catalogue package status, and redacted App Store logs. It does not scan or copy package configuration, credentials, ROMs, or user data.
- Controller mappings are device- and controller-scoped, validated before load/save, and replaced atomically. A missing GUID uses the controller name only within the detected Knulli device; an unidentified controller is not persisted.
- MagicX Zero 28 is an experimental App Store target. PlayTime, Grout, and RetSend can be tested there when all runtime checks pass. None of them has current MagicX device evidence.

Report a security issue privately to the repository owner. Do not include user data, tokens, or private catalogue URLs in a public report.
