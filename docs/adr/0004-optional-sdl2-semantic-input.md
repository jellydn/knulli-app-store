# ADR 0004: Keep the SDL2 GUI optional behind a build tag and semantic controller mapping

- Status: accepted
- Date: 2026-09-15
- Renderer choice superseded by [ADR 0010](0010-gabagool-knulli-frontend.md).
  The optional build tag, backend and semantic input boundaries still apply.

## Context

The product needs a controller-only device UI, but SDL2 requires CGO and a device-specific shared library. Handheld A/B button layouts also differ. A GUI that extracts archives or assumes one physical button map would couple safety policy to presentation and break across devices.

## Decision

Keep installer policy in `internal/installer` and catalogue actions in `internal/appstore.Backend`. Put the SDL adapter in `internal/sdlui` behind the `sdl` build tag, with a small local cgo wrapper that imports only the SDL 2.0 functions in use. Keep `internal/ui` as a pure-Go state machine and `internal/input` as semantic actions (`confirm`, `back`, and the rest) bound to SDL GameController buttons for one device/controller identity.

Prefer Knulli's generated `SDL_GAMECONTROLLERCONFIG`. Do not read raw evdev nodes or assume a universal physical A/B layout. Updated 2026-10-19 at the owner's request: use automatic SDL/KNULLI controls and open the catalogue immediately, without setup, calibration, or mapping persistence. The launcher selects Knulli's default ES-to-SDL letter policy or Ports Xbox mode from its runtime flag. Bindings compensate once; events are unchanged. ES A confirms, B backs out, Y exports diagnostics, and Select+Y quits. Legacy custom mapping files are ignored and untouched. The CLI stays CGO-free and statically cross-compiled.

## Consequences

### Positive

- Safety tests and the recovery CLI run without SDL2.
- Device artifacts can still link `libSDL2-2.0.so.0` for the GUI.
- Automatic controls remove first-run friction and stale custom-mapping overrides.

### Negative

- GUI builds need a cross compiler, SDL2 headers, and a glibc contract (CI currently assumes symbols through 2.34).
- The GUI remains experimental: one community-tested TrimUI Smart Pro, incomplete MagicX coverage.
- Custom cgo wrappers and the `sdl` build tag add maintenance cost for the optional GUI.
