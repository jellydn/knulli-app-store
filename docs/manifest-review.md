# Manifest review workflow

## States

- `candidate`: read-only metadata with an optional validated, pinned release. Compatibility and install instructions are rejected. Review can retain checksums and archive sizes without granting installation permission.
- `experimental`: actionable on any device when Knulli, architecture, ABI, and runtime dependencies pass. Device identity and display size do not block installation. Device lists, resolutions and broad bounds are optional; do not invent them to pass validation. An explicit warning is required; a minimum firmware version may be omitted when no evidence supports one.
- `installable`: two reviewers have checked all evidence and the package can be used by the installer.
- `verified`: `installable` plus linked real-device test evidence for every declared device and resolution.

`review.approval` is separate from these technical states. Community approval permits catalogue inclusion, but it does not provide compatibility evidence, enable installation, or grant a verified badge.

`real-device-test` evidence records the tester, date, exact package version, firmware, architecture, device, resolution, and passed result. Missing exact evidence shows **UNTESTED**, even if the release is verified on another device. It does not block Install, Manage existing, Update, or Repair. Candidate entries still need an actual installation definition; an untested badge cannot supply a missing archive or install layout.

## Promotion checklist

1. Confirm the repository belongs to the project or documented maintainer.
2. Read the repository license file and record a valid SPDX expression. Only an authorized upstream-managed experimental direct download may omit an unverified license, with a visible license warning and no asset redistribution.
3. Select a version-tagged release URL. Reject branch archives and `latest` aliases. Never execute a remote installer. A release containing a base64 `tar.gz` payload can use `release.base64_offset`: the exact byte offset after its marker line. Verify the full release's size and checksum, and inspect the decoded archive. No script prefix is executed.
4. Obtain the SHA-256 from the publisher through a channel separate from the downloaded asset. Record compressed and installed byte sizes.
5. Inspect the complete archive. It must contain regular files only and one declared launcher.
6. Review source or packaged behavior for network use and filesystem writes.
7. List the narrowest `/userdata` destination, menu file, preserved configuration, and allowed write paths.
8. Establish the Knulli version evidence, architecture, dynamic ABI, minimum glibc version, and runtime libraries. Missing runtime facts must block with the exact reason. Record device scope and resolutions as advisory evidence, not an installation allowlist.
9. Have a second reviewer reproduce the checksum, archive inspection, and manifest validation before `installable` status. An authorized `experimental` review can be completed by one maintainer; it must still pass release, runtime and lifecycle safety controls.
10. Use `experimental` for an authorized, clearly warned hardware test when mandatory release and path controls pass but compatibility evidence is incomplete. Promote to `installable` after review. Add `real-device-test` evidence and promote to `verified` only after the exact matrix passes on hardware.

For ROM-download software, record a visible copyright warning and review every provider and self-update path. Approval does not establish that a ROM download is lawful. The user must have the required rights and comply with local law.

`install.upstream_managed` is an explicit experimental exception to Store-managed runtime updates. Review the original archive and Store write paths, disclose first-launch setup and app-controlled updates/downloads, and keep the package experimental. Runtime writes are not sandboxed. Store Repair returns managed files to the pinned release. Declare read-only `uninstall_checks` for external hooks or service state and verify that a blocked removal changes no package files, menu entries or installed state. Never execute upstream install/uninstall scripts. See [upstream-managed installation](how-to-add-a-package.md#upstream-managed-installation).

## Build the index

`knulli-app catalogue` sorts packages by ID and hashes the canonical manifest representation. It omits timestamps so identical inputs produce identical bytes. Device CI signs the exact index bytes with ed25519 and ships `catalog-index.json.sig`. The public key is compiled into that device binary. Local `make catalogue` stays unsigned for review.

The binary release asset remains upstream. A reviewed same-length binary patch is allowed only when the manifest pins the path, offset, exact source bytes, exact replacement bytes, and final SHA-256. Source or result mismatch stops before a transaction. Use this only to remove a proven unsafe runtime path when a normal upstream setting does not exist.
