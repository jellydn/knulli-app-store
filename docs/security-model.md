# Security model

## Protected assets

- User configuration and ROM metadata
- Existing files at package destinations
- EmulationStation `gamelist.xml`
- Package state and original-file backups
- The integrity of installed upstream files

## Trust boundaries

Catalogue review decides whether metadata is actionable. HTTPS provides transport security, while the reviewed SHA-256 binds the archive bytes. A checksum supplied only beside a mutable asset is not enough evidence for catalogue promotion.

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
- Show a detected external copy as the row state **EXTERNAL** with the action **Manage existing**, then inventory it before adoption. Exact release-hash matches can become manager-owned; changed and unknown files are marked unmanaged and backed up because ownership is not proven.
- Preserve declared configuration during repair, update, and uninstall.
- Track installed paths, hashes, modes, and original-file backups.
- Redact URL credentials, query strings, fragments, and common secret fields from diagnostic logs. Cap the active log at 512 KiB and retain one rotated copy.
- Parse and rewrite `gamelist.xml` as XML while retaining unknown elements. Add only an absent exact launch path. Remove only an unchanged entry that manager state proves the installer created.
- Ask Knulli's loopback `/reloadgames` endpoint to queue a live refresh after a committed menu change. Report **Restart required** if the request is not accepted; never claim that the refresh completed.

## Non-goals and residual risks

- There is no package runtime sandbox.
- Catalogue-index signing and key distribution are future release-workflow tasks.
- Root can change files below approved `/userdata` paths despite installer checks.
- Path checks do not defend against a hostile local process that races a checked directory into a symlink.
- A corrupt transaction journal blocks the next locked operation until the leftover directory is inspected.
- Empty directories can remain after rollback or uninstall.
- Device detection reads the Knulli board identifier (`/boot/boot/knulli.board`), then `/etc/knulli-device`, then Batocera's board file. An unrecognized board shows `UNKNOWN DEVICE` and is blocked until the operator supplies explicit flags.
- The GUI has one community-reported TrimUI Smart Pro test. Grout 5.1.0.0 is verified only for that declared device matrix. The report did not itemize lifecycle steps.
- PlayTime is experimental on Smart Pro and MagicX Zero 28. Its Smart Pro test does not verify its untested MagicX matrix. Grout's built-in updater is outside manager transactions and must not be used.
- Diagnostic export contains detected platform metadata, public catalogue package status, and redacted App Store logs. It does not scan or copy package configuration, credentials, ROMs, or user data.
- Controller mappings are device- and controller-scoped, validated before load/save, and replaced atomically. A missing GUID uses the controller name only within the detected Knulli device; an unidentified controller is not persisted.
- MagicX Zero 28 is an experimental App Store target. PlayTime has separate user-authorized experimental compatibility; Grout remains blocked.

Report a security issue privately to the repository owner. Do not include user data, tokens, or private catalogue URLs in a public report.
