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
