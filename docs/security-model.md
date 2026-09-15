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
- Match firmware, minimum version, architecture, H700 device family, and resolution before download.
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
- Snapshot every changed file and roll back a failed operation in reverse order.
- Keep a persistent backup when installation overwrites a file the package did not own.
- Preserve declared configuration during repair, update, and uninstall.
- Track installed paths, hashes, modes, and original-file backups.
- Parse and rewrite `gamelist.xml` as XML while retaining unknown elements.

## Non-goals and residual risks

- There is no package runtime sandbox.
- Catalogue-index signing and key distribution are future release-workflow tasks.
- Root can change files below approved `/userdata` paths despite installer checks.
- Path checks do not defend against a hostile local process that races a checked directory into a symlink.
- Power loss is not yet journal-recovered across process restarts. Atomic file replacement limits corruption, and synchronous failures roll back.
- Empty directories can remain after rollback or uninstall.
- H700 detection file names need validation against real Knulli images. Operators must supply explicit flags when detection is incomplete.
- The GUI has one community-reported TrimUI Smart Pro test. Package-specific real-device testing has not been completed, and no current package has a verified badge.
- EmuDrop is an approved catalogue exception, not an approved installer payload. Its copyright risk, missing license, and launch-time updater keep it non-installable.

Report a security issue privately to the repository owner. Do not include user data, tokens, or private catalogue URLs in a public report.
