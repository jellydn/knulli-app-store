# ADR 0009: Allow untested devices and displays

- Status: accepted
- Date: 2026-10-19
- Supersedes the device/display installation gate in ADR 0006, not its detection rules.

## Context

The owner requested that device constraints be relaxed: packaged apps should remain available on untested hardware, with a warning instead of a device allowlist. Missing reports are not proof that an app cannot run.

## Decision

Device identity, declared devices, exact resolutions, and display bounds do not block Install, Manage existing, Update, Repair, or recovery. Unknown device/display values remain unknown; no values or evidence are invented. The UI, CLI, and installer share this policy.

Only an exact passed report for the package version, firmware, architecture, device, and resolution grants a tested badge. Other packaged apps show **UNTESTED**, including releases verified on a different device. Existing confirmation and package warnings remain visible.

Firmware, architecture, ABI/glibc, and runtime dependencies remain required. Signature, size, checksum, archive safety, `/userdata` containment, and transactional ownership are unchanged. Candidate entries without an install definition cannot be installed by removing a device gate. ADR 0002's declarative package contract remains in force.

## Consequences

- Random Game Picker and the other packaged apps are available on unlisted boards and displays outside the recorded scope.
- Installation does not prove launch, layout, or controller behavior. Small displays and unknown devices may need troubleshooting.
- Detection and exact evidence remain useful for diagnostics and tested badges without denying an installation solely for missing device coverage.
- PocketCurator and RAOfflineProxy still need package integration, not device approval; neither currently declares an actionable archive/install layout.
