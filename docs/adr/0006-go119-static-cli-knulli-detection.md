# ADR 0006: Target Go 1.19, a static aarch64 CLI, and Knulli-owned firmware files

- Status: accepted
- Date: 2026-09-15

## Context

The recovery CLI must run on Knulli `aarch64` devices without a compiler or extra libraries. Current Knulli still publishes Buildroot's `ID=buildroot` in `/etc/os-release`. Device and display facts also differ across TrimUI Smart Pro and MagicX Zero 28. A toolchain or detection shortcut would produce binaries or compatibility errors that operators cannot trust.

## Decision

Use Go 1.19 as the language baseline, keep the CLI on the standard library plus `golang.org/x/image` for GUI text, and build `cmd/knulli-app` with `CGO_ENABLED=0` for linux/arm64. Detect firmware from Knulli-owned `OS_NAME="knulli"`, not `ID=buildroot`. Read the release identifier from `/usr/share/knulli/knulli.version`. Identify devices from `/boot/boot/knulli.board` values such as `trimui-smart-pro` and `magicx-zero-28`. Require explicit resolutions in manifests and validate SDL, window, display-mode, and framebuffer candidates before a compatibility check.

CI builds experimental Ports artifacts on Debian Bookworm with glibc symbols through 2.34 for the GUI binary. Operators must supply `-firmware`, `-firmware-version`, `-arch`, `-device`, and `-resolution` when detection is incomplete.

## Consequences

### Positive

- The CLI is a portable static binary for catalogue review and recovery.
- Compatibility errors can show raw value, normalized value, source, and the full detected matrix.
- Device support is explicit instead of inferred from a shared SoC family.

### Negative

- Go 1.19 and `golang.org/x/image v0.7.0` are behind current releases; upgrades need a coordinated CI and staticcheck change.
- H700 detection file names still need validation against real Knulli images.
- The GUI glibc/SDL contract is evidenced by CI and one system binary, not a published Knulli compatibility guarantee.
