# ADR 0007: Raise the language baseline to Go 1.27

- Status: accepted
- Date: 2026-09-15
- Supersedes: Go 1.19 toolchain version in [ADR 0006](0006-go119-static-cli-knulli-detection.md)

## Context

CI already ran Go 1.27.x and staticcheck v0.8.1. `go.mod` still declared Go 1.19. SDL packages kept legacy `// +build sdl` lines. `golang.org/x/image` stayed on v0.7.0. That split missed later toolchain fixes and blocked a current x/image release.

## Decision

Set `go 1.27` in `go.mod`. Keep CI on `1.27.x` and Debian Bookworm `golang:1.27-bookworm`. Drop `// +build sdl` lines. Upgrade `golang.org/x/image` with the same baseline. Keep the static CGO-free aarch64 CLI and Knulli-owned firmware detection from ADR 0006.

## Consequences

### Positive

- Language, CI images, staticcheck, and GUI image code share one supported floor.
- SDL packages no longer carry pre-Go 1.17 build tags.

### Negative

- Developers need Go 1.27 or later.
- A later Go bump still needs a coordinated staticcheck and container change.
