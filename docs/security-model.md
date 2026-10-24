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

Explicitly authorized `upstream_managed` packages remain experimental. The Store verifies their original publisher download and installation transaction, but leaves first-launch setup, runtime writes, runtime acquisition and app updates to upstream. Such updates do not inherit Store checksum approval. Health checks report changed managed bytes; Repair restores the pinned release after confirmation. A missing license is allowed only for this experimental direct-download model with a visible license warning; assets are not mirrored or redistributed. A warning is not a runtime sandbox or a license grant.

For upstream-managed experimental packages only, a missing publisher checksum may be replaced by the GitHub release asset's API digest, independently compared with the download and pinned in the reviewed manifest. The install warning must disclose this limitation. This trusts the publisher's GitHub account and GitHub, not independent publisher authentication: a compromised account can publish malicious bytes with a matching digest before review. Installation still requires the exact reviewed hash and size; it never fetches a new expected digest. This exception does not permit promotion beyond experimental.

## Store self-update

The Store card checks `api.github.com/repos/jellydn/knulli-app-store/releases` only after confirmation. It includes CI pre-releases and selects the asset for the packaged build target. A separate Update confirmation uses the pinned official release URL, byte size, and SHA-256 returned by GitHub. Missing digests, download mismatches, unsafe archive paths, unexpected file inventories, symlinks, and insufficient space reject the update.

Only the two Store binaries, catalogue/signature pair, README, launcher, and icon under `/userdata/roms/ports` can change. The updater shares the manager lock and journal/rollback implementation. The launcher runs `knulli-app recover` before starting the UI: recovery can restore a key-bearing binary and its matching catalogue before either is loaded. The service also recovers before catalogue verification. Atomic file replacement keeps the current Linux process running; the user must exit and reopen to load the new version. No package state or settings are removed.

Transactional copies, including snapshot restoration, stream to a temporary file in the destination directory, sync and close it, then rename it over the destination and sync the directory. They never truncate a running executable. A failed rollback retains the open journal and snapshots so recovery can retry; backups are removed only after restoration succeeds. This cannot recover journals already deleted by an older build.

This trusts the official GitHub repository and HTTPS API, not an independent update signing key. A compromised publisher account can publish malicious Store bytes and matching digests. The catalogue's per-build signature is not a separate signature over the Store ZIP. Custom install locations and builds without embedded target/version metadata use manual ZIP replacement.

## Installer controls

- Reject lifecycle actions on candidate manifests and reject unknown manifest fields. Candidates can retain validated release metadata without installation permission.
- Match firmware, minimum version, architecture, ABI, and required runtime libraries before download. Device and display scope are advisory; missing exact device-test evidence shows **UNTESTED**, not a blocked action.
- Accept only HTTPS, version-pinned release URLs and require exact compressed size and SHA-256.
- Limit compressed and installed size to 512 MiB and check free space before download.
- Stage extraction before package writes.
- Decode a reviewed embedded base64 `tar.gz` only after verifying the full release size and checksum. The prefix is never executed; normal archive safety and installed-size bounds still apply.
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
- Generate optional menu launchers only for the declared installed package command. Shell-quote arguments, track the launcher as a managed file, back up originals, and use the same journal and rollback as release files. Never execute launchers during Store operations.
- Before removal, perform declared read-only checks for shared hooks and service-state paths. Matches, symlinks, unreadable entries and oversized hook files block removal before mutation. Never execute upstream cleanup or delete shared hooks. Checks cover declared locations, not arbitrary runtime path overrides or every possible active process.
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
- Device detection reads the Knulli board identifier (`/boot/boot/knulli.board`), then `/etc/knulli-device`, then Batocera's board file. An unrecognized board shows `UNKNOWN DEVICE` but is not blocked. Missing or out-of-scope display metadata also does not block installation; runtime controls remain enforced.
- The GUI has one community-reported TrimUI Smart Pro test. The Grout result applies to 5.1.0.0, not the current 5.2.0.0 release. The report did not itemize lifecycle steps.
- All seven packaged apps can install when Knulli, architecture, glibc ABI, and required runtime libraries match. Optional device lists and display bounds do not restrict experimental installation. Exact package-version device evidence controls the tested badge. PocketCurator, RAOfflineProxy and retroshelf are explicitly authorized upstream-managed experimental packages. PocketCurator's deliberate ROM deletion and executable downloads, RAOfflineProxy's shared configuration changes and updater, and retroshelf's binary-driven hooks and updater remain runtime risks. They are disclosed, not removed. RAOfflineProxy and retroshelf removal checks protect documented shared hooks; users must complete cleanup first.
- Grout's reviewed staging patch replaces its only updater metadata URL with a reserved `.invalid` URL and verifies the transformed binary hash. A mismatch stops installation before any transaction write.
- Health checks use the observed destination mode after transactional copy or adoption mode normalization, and require that mode to keep the owner read, write, and execute permissions the installer set. A wider mode from a normalizing filesystem is not a health failure; a lost owner permission is. They compare the full SHA-256 of every manager-owned immutable file on every check. Unmanaged external files show their full reviewed and actual hashes for Repair instead of being treated as reviewed bytes.
- Force reinstall is offered only after a recoverable adoption ownership failure or a validated stale transaction journal. An active transaction blocks all recovery actions. Compatibility, free-space, signature, checksum, archive, path, and licensing failures stay blocked and show their exact reason; force mode cannot disable those checks.
- Before force reinstall changes a destination, the manager inventories and hashes every regular file. It preserves declared data and retains a complete timestamped backup plus original-path, hash, mode, and size records under `/userdata/system/knulli-app-store/recovery-backups/<package-id>/<timestamp>/`. A failed operation removes its incomplete retained backup after journal rollback restores the pre-operation files.
- Diagnostic export contains detected platform metadata, public catalogue package status, and redacted App Store logs. It does not scan or copy package configuration, credentials, ROMs, or user data.
- Controller controls are automatic SDL/KNULLI defaults. The app never reads, writes, migrates, or deletes legacy `controller-mappings.json` files and never writes system controller mappings. Missing GUIDs do not prevent navigation; hotplug resets held chord state.
- MagicX Zero 28 is an experimental App Store target. PlayTime, Grout, and RetSend can be tested there when all runtime checks pass. None of them has current MagicX device evidence.

Report a security issue privately to the repository owner. Do not include user data, tokens, or private catalogue URLs in a public report.
