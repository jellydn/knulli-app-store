# Architecture decision records

Accepted decisions for Knulli App Store. New records use the next number and keep the Context / Decision / Consequences shape.

| Number | Title | Status |
| --- | --- | --- |
| [0001](0001-safe-installer-before-ui.md) | Build a transactional installer before the UI | accepted |
| [0002](0002-declarative-reviewed-catalogue.md) | Use declarative reviewed manifests instead of remote install scripts | accepted |
| [0003](0003-userdata-transactional-writes.md) | Constrain package writes to `/userdata` with transactional rollback | accepted |
| [0004](0004-optional-sdl2-semantic-input.md) | Keep the SDL2 GUI optional behind a build tag and semantic controller mapping | accepted |
| [0005](0005-portmaster-external-provider.md) | Treat PortMaster as an external provider, not a copied catalogue | accepted |
| [0006](0006-go119-static-cli-knulli-detection.md) | Target Go 1.19, a static aarch64 CLI, and Knulli-owned firmware files | accepted |
| [0007](0007-go127-language-baseline.md) | Raise the language baseline to Go 1.27 | accepted |
| [0008](0008-catalog-index-signing.md) | Sign the device catalogue index with ed25519 | accepted |
| [0009](0009-advisory-device-compatibility.md) | Allow untested devices and displays | accepted |
| [0010](0010-gabagool-knulli-frontend.md) | Use a gabagool-backed Go frontend, KNULLI first | accepted |
| [0011](0011-reviewed-free-games.md) | Admit reviewed free games and homebrew ROMs | accepted |
| [0012](0012-reviewed-game-checksums.md) | Pin reviewed publisher game downloads without a publisher checksum | accepted |
| [0013](0013-compatibility-profile-distribution.md) | Distribute one KNULLI compatibility-profile Store ZIP | accepted |

ADR 0007 supersedes ADR 0006's Go 1.19 baseline. ADR 0009 supersedes its device/display installation gate. ADR 0006's static CLI and firmware detection decisions still apply.

ADR 0010 supersedes only ADR 0004's renderer choice, not its backend, build-tag or input policy.

ADR 0011 supersedes only ADR 0005's package-type restriction. PortMaster remains external-only.

ADR 0012 adds an owner-approved checksum exception for reviewed experimental games, not a rights or compatibility exception.

ADR 0013 supersedes ADR 0010's device-named packaging restriction only; responsive layout and firmware/ABI/input constraints remain.
