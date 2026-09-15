# ADR 0002: Use declarative reviewed manifests instead of remote install scripts

- Status: accepted
- Date: 2026-09-15

## Context

Handheld community installers often fetch a script from the network and pipe it to a shell. That model cannot prove which bytes run, which paths change, or how to repair and uninstall later. Package code also runs later with access to user files, so catalogue metadata must be reviewable before any write.

## Decision

Ship packages as JSON manifests under `catalogue/packages/` that match `schema/package-manifest-v1.schema.json`. Require HTTPS, version-pinned release URLs, exact compressed size, SHA-256, ZIP or `tar.gz` format, and an immutable asset. Keep community approval separate from technical status. Only `experimental`, `installable`, and `verified` packages are actionable. Candidate manifests must not contain release, compatibility, or install fields. The installer copies regular files from a verified archive. It never runs a remote install script.

The weekly update checker downloads GitHub release metadata only. It reports changes for manual review and never edits or approves a package.

## Consequences

### Positive

- Reviewers can inspect one file plus evidence URLs before a write occurs.
- Install, repair, update, and uninstall share the same reviewed inventory.
- Community interest cannot bypass technical review.

### Negative

- Packages that ship only a self-extracting script, a mutable `latest` asset, or undeclared writes stay blocked. RAOfflineProxy and PocketCurator are current examples.
- Catalogue-index signing and key distribution remain future release-workflow work.
- Review cost is high: every path, checksum, and device matrix must be recorded by hand.
