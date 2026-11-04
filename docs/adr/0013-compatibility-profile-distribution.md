# ADR 0013: Distribute one KNULLI compatibility-profile Store ZIP

- Status: accepted; normal green-main publication authorized 2026-11-04
- Date: 2026-11-03
- Supersedes: ADR 0010's device-named packaging restriction only

## Context

Responsive layout now follows SDL drawable capability rather than device names.
The two old distributions used the same AArch64 build contract and identical
launcher behavior. Two names duplicate releases and make display identity look
like binary compatibility. Old installed updaters nevertheless filter those
names, so changing the server's filename cannot migrate them automatically.

## Decision

Build one `knulli-aarch64-glibc-sdl2` profile, named
`knulli-app-store-knulli-aarch64-glibc-sdl2-experimental-VERSION.zip`, in both CI
and tag workflows. Use one shared launcher/guide and preserve the seven-file
inventory, KNULLI firmware policy, static CLI, AArch64/glibc/SDL gates, controller
letter policy, catalogue signatures and transactional updater. Layout is not an
architecture, ABI, firmware or physical-input compatibility check.

The owner confirmed one **installable** ZIP plus a separate `SHA256SUMS.txt`
on 2026-11-03: two uploaded files, not two distributions. The catalogue and
signature stay embedded and paired with the GUI. Keep HTTPS GitHub API size/digest
verification; do not represent it as independent Store signing.

New updater code selects only the generic profile and accepts former device
target metadata as aliases. Existing binaries need verified manual seven-file
replacement unless a separately approved, demonstrably eligible bridge exists.
No duplicate device ZIPs enter new releases. Do not rewrite historical assets.
The current 20-release inventory has both old prefixes on every release, so no
automatic bridge is verified. Offline compatibility-copy preparation is not
publication or proof of migration. See [distribution and migration](../store-distribution.md).

## Consequences

- One build and launcher remove device-name routing without claiming universal
  hardware compatibility. Existing firmware/ABI and physical device tests remain.
- Archive paths and manager state do not migrate. Existing transactional rollback,
  signature checks and app/settings preservation continue unchanged.
- Collectors reject duplicate/wrong-profile ZIPs; reruns refuse mixed historical
  assets instead of deleting them. No bridge is automatically published.
- Old clients cannot all migrate in-app under the no-new-device-ZIP policy.
  Manual verification/replacement is the safe supported path. Any historical
  replacement or exceptional bridge release needs a new rollout decision.
- The owner authorized normal green-main publication on 2026-11-04, after the
  initial local-only approval. Hardware acceptance remains separate and physical
  footer/device testing is outstanding. The approved release format retains a
  separate checksum file without weakening update verification.
