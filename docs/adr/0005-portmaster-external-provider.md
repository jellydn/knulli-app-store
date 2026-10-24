# ADR 0005: Treat PortMaster as an external provider, not a copied catalogue

- Status: accepted
- Date: 2026-09-15

## Context

PortMaster is the established way to install game ports on Knulli. Copying that catalogue would duplicate an upstream installer, mix ROM-adjacent content into this review process, and create two sources of truth for the same packages.

## Decision

Record PortMaster in `catalogue/providers/portmaster.json` as a featured external provider with `catalogue_policy: external-only`. Users open it through Knulli's official install and launch mechanism. This repository does not mirror the PortMaster catalogue, implement another PortMaster installer, or accept ROM-download sources into the official package list.

This App Store reviews utilities, themes, and integrations that can meet the manifest and write-path policy.

## Consequences

### Positive

- Review effort stays on packages this installer can own, hash, and roll back.
- Users still have a clear path to PortMaster without a second unofficial installer.
- ROM-download sources stay outside official catalogue policy.

### Negative

- The GUI cannot install PortMaster games itself.
- Users must understand two launch paths: App Store packages versus Knulli's PortMaster integration.
- Featured-provider presentation has to stay honest that this project does not own that catalogue.
