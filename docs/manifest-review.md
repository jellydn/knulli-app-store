# Manifest review workflow

## States

- `candidate`: informative metadata only. The schema rejects release and install instructions in this state.
- `experimental`: actionable only for the declared test matrix. It requires an explicit warning and may omit a minimum firmware version when no evidence supports one.
- `installable`: two reviewers have checked all evidence and the package can be used by the installer.
- `verified`: `installable` plus linked real-device test evidence for every declared device and resolution.

`review.approval` is separate from these technical states. Community approval permits catalogue inclusion, but it does not provide compatibility evidence, enable installation, or grant a verified badge.

`real-device-test` evidence records the tester, date, exact package version, firmware, architecture, device, resolution, and passed result. A mixed device matrix can show a tested device without marking an untested device as verified.

## Promotion checklist

1. Confirm the repository belongs to the project or documented maintainer.
2. Read the repository license file and record a valid SPDX expression.
3. Select a version-tagged release URL. Reject branch archives, `latest` aliases, and remote installer scripts.
4. Obtain the SHA-256 from the publisher through a channel separate from the downloaded asset. Record compressed and installed byte sizes.
5. Inspect the complete archive. It must contain regular files only and one declared launcher.
6. Review source or packaged behavior for network use and filesystem writes.
7. List the narrowest `/userdata` destination, menu file, preserved configuration, and allowed write paths.
8. Establish the minimum Knulli version and test `aarch64`, every declared device ID, and each listed resolution.
9. Have a second reviewer reproduce the checksum, archive inspection, and manifest validation.
10. Use `experimental` for an authorized, clearly warned hardware test when mandatory release and path controls pass but compatibility evidence is incomplete. Promote to `installable` after review. Add `real-device-test` evidence and promote to `verified` only after the exact matrix passes on hardware.

For ROM-download software, record a visible copyright warning and review every provider and self-update path. Approval does not establish that a ROM download is lawful. The user must have the required rights and comply with local law.

## Build the index

`knulli-app catalogue` sorts packages by ID and hashes the canonical manifest representation. It omits timestamps so identical inputs produce identical bytes. Release automation can sign `build/catalog-index.json` with a separately managed key. Signature transport and key rotation are intentionally not part of manifest v1.

The binary release asset remains upstream. The generated index contains metadata and hashes only.
