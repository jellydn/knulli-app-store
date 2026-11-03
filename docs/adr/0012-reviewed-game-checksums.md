# ADR 0012: Pin reviewed publisher game downloads without a publisher checksum

- Status: accepted by the owner in the [game catalogue thread](https://ampcode.com/threads/T-01a19e8e-05d2-75f5-81dd-46a4794ed7b3).
- Date: 2026-11-03
- Supersedes only ADR 0011's publisher-checksum requirement for explicitly reviewed experimental games.

## Context

Some homebrew publishers provide versioned playable archives but no checksum. The owner approved a narrow exception after the first source review added no games beyond Freedoom. A checksum calculated by this Store detects later replacement bytes, but does not independently authenticate the first download.

## Decision

An experimental game may use its original publisher's version-pinned HTTPS ZIP or tar.gz without a publisher checksum. Reviewers must download the archive, record its exact size and SHA-256, inspect the inventory, link the publisher's download page and rights evidence, and add `store-checksum` evidence for the exact release URL. The install warning must disclose the missing publisher checksum. Non-GitHub URLs must contain the reviewed version in their path, with no credentials, query or fragment. GitHub downloads must still belong to the declared publisher repository.

Trust is placed in the publisher's HTTPS site and download host at review time. The signed catalogue pins the reviewed bytes; installation never downloads a new expected checksum. This exception cannot promote a game beyond experimental, authenticate a publisher, or waive code and asset rights, runtime compatibility, archive safety, lifecycle or save-retention review.

Keep original notices. Where a publisher archive lacks licence text, reproduce the required licence notices and attribution in the package's review notes, distributed in the signed catalogue and retained in the installed-state manifest. Include full-licence and exact corresponding-source links where required. Do not repackage or mirror the archive. Files remain the publisher's unmodified files, except existing reviewed path mappings.

Mutable aliases, expiring download links, source-only snapshots, Nursery and PortMaster assets, commercial files and unclear asset rights remain excluded. The existing checksum rules for other package types do not change.

## Consequences

- More reviewed homebrew games can be installed without weakening byte verification or filesystem boundaries.
- Initial publisher compromise is not detected by a Store-calculated checksum; the warning must not imply otherwise.
- Updates need a new human review and pinned checksum. The automatic checker must not treat this evidence as a publisher checksum.
- No physical-device tested badge is granted by this exception.
