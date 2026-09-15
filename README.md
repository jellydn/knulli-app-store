# Knulli App Store

Knulli App Store is an early, safety-focused package manager for community Knulli utilities, themes, and integrations. The first milestone is a reviewable catalogue format and a headless installer. A device UI will come only after the package and transaction contracts are stable.

## Current scope

- Knulli devices in the H700 family
- `aarch64`
- One resolution per approved manifest; no resolution is assumed
- ZIP and `tar.gz` release archives, up to 512 MiB compressed and installed
- Writes below `/userdata` only

This project does **not** claim support for other devices. The current catalogue contains candidates only. Nothing in it is installable or marked as device-verified.

PortMaster is a featured external provider. Users open it through Knulli's official install and launch mechanism. This repository does not copy the PortMaster catalogue or implement another PortMaster installer. ROM-download sources, including EmuDrop, are outside the official catalogue policy.

## Why the installer comes first

Package operations can damage user data even when a user interface looks safe. The command-line core makes compatibility checks, download verification, extraction, rollback, and uninstall behavior testable without SDL or a device display. See [Architecture](docs/architecture.md) and [Security model](docs/security-model.md).

## Build and check

Go 1.19 or newer is required.

```sh
make check
make build
make catalogue
```

Build the initial device target without CGO:

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o build/knulli-app-aarch64 ./cmd/knulli-app
```

## Command-line use

```sh
# Review one manifest without installing it.
go run ./cmd/knulli-app validate catalogue/packages/io.github.unitreign.playtime.json

# Build a deterministic index that can be signed by release automation.
go run ./cmd/knulli-app catalogue -output build/catalog-index.json

# Install an approved local manifest. Flags fill values that detection cannot read.
sudo ./build/knulli-app install \
  -firmware knulli -firmware-version VERSION \
  -arch aarch64 -device h700 -resolution WIDTHxHEIGHT \
  package.json

sudo ./build/knulli-app repair package.json
sudo ./build/knulli-app update package-v2.json
sudo ./build/knulli-app uninstall org.example.package
```

`-root` redirects all device paths into another directory. Tests use it for a complete temporary-filesystem workflow. It is also useful for offline inspection; it must not point at an untrusted tree with symlinked path components.

The installer never runs remote install scripts. It only copies regular files from a verified archive, preserves declared configuration, and writes one optional EmulationStation menu record.

## Catalogue status

The five requested projects are present as non-actionable candidates:

| Candidate | Promotion blocker |
| --- | --- |
| PlayTime | No publisher checksum; H700 paths and behavior not tested |
| Grout | No publisher checksum for the Knulli asset; configuration and paths not tested |
| RAOfflineProxy | Knulli asset is an unrestricted install script with no publisher checksum |
| ETK Tool | Release is script-based; system-space behavior is outside the initial policy |
| PocketCurator | A checksum exists, but the repository has no declared license; paths and device behavior are not tested |

These notes record evidence reviewed on 2026-09-15. Upstream facts can change. Follow the [manifest review workflow](docs/manifest-review.md) before promotion.

## UI status

An SDL2 UI is intentionally deferred. The current catalogue has no installable package, and the exact H700 controller mapping and SDL2 runtime supplied by Knulli have not been verified. Adding an untested UI now would add a second portability surface without a safe operation to expose. The future UI boundary is documented in [Architecture](docs/architecture.md).
