# ADR 0010: Use a gabagool-backed Go frontend, KNULLI first

- Status: accepted
- Date: 2026-10-24
- Supersedes: only ADR 0004's local C wrapper / renderer choice. Its optional
  `sdl` build tag, backend boundary and semantic input decisions remain in force.

## Context

The requested Go rewrite already exists. The inspected baseline is
[`2540b237`](https://github.com/jellydn/knulli-app-store/commit/2540b23779907ef73d28500a72ccbb44f802e444).
Rewriting its installer would put existing archive validation, ownership,
rollback, recovery and self-update behavior at risk without a product benefit.
The owner instead authorized a frontend migration that keeps the one-card flow.

We researched [Grout at `15b02e26`](https://github.com/rommapp/grout/tree/15b02e269980d05768807686ad9ecbbd0879229c).
It pins gabagool/v2 v2.26.6 and separates typed screen input/output/actions,
application state, routing, transitions and firmware-specific behavior. Its
shared footers name controller actions consistently. Those boundaries are
useful here; its RomM client, download manager and multi-CFW implementations are
not Store installation policy and must not be copied into it.

## Decision

Keep Go for both the shared service and the frontend. Go provides a static
recovery CLI, standard-library hashing/signatures/HTTP and testable concurrent
operations. Keeping one language avoids a second binding layer between the
transactional installer and presentation. This is not a new language migration.

Pin gabagool/v2 v2.26.6, the version inspected in Grout. Use its public advanced
window/renderer API (`Init`, `GetWindow`, `Present`, `Close`) and go-sdl2 event
types instead of the local C wrapper. Keep the existing custom card canvas,
selection, confirmations, health detours, notices and footer vocabulary. This
is a toolkit-backed custom screen, **not** adoption of Grout's list layout or
gabagool's stock screen components.

Stock components run blocking event loops and add Start-to-confirm. Using them
now would introduce another owner for controls, confirmation and asynchronous
operation progress. The Store instead uses one loop and disables the toolkit's
input processor. Raw SDL GameController events reach the existing semantic
mapper unchanged. Reject `INPUT_CAPTURE` before toolkit initialization: its
developer wizard writes assignments and is not part of Store behavior. Send
framework logs to stderr rather than creating a toolkit log beside the app.

Keep these ownership boundaries:

| Boundary | Owner and contract |
| --- | --- |
| Catalogue / discovery | `manifest`, `catalog`, and `appstore.Service.Items`: reviewed, signed metadata and permitted actions. No rendering or toolkit dependency. |
| Install and update operations | `installer` and `appstore.Backend.Execute`: pinned bytes, staging, journaling, ownership, preserved data and recovery. The frontend never extracts, shells out or mutates package files. |
| KNULLI firmware profile | `platform` plus the KNULLI Ports launchers: firmware evidence, paths, runtime compatibility, display evidence and layout context. Menu refresh stays in the existing installer integration. |
| Input mapping | `input`: KNULLI versus Ports Xbox letter policy, semantic actions, source identity, hotplug and quit chord. `sdlui` only translates SDL event types. No raw evdev controls or custom assignment persistence. |
| UI state and transitions | `ui.Model`: catalogue selection/filter memory, health/actions/confirmation states, asynchronous progress and completion. `sdlui.Handle` routes semantic actions; drawing cannot execute operations. |
| Presentation and lifetime | `sdlui`: one-card bitmap, letterboxing, framework window/renderer, event pump and capture. Only this optional adapter imports gabagool/SDL. |

These are extension seams, not a generic firmware framework. A future CFW needs
its own reviewed profile and integration tests; none is implemented or claimed
here. The only packaged targets remain KNULLI TrimUI Smart Pro and MagicX Zero
28. Neither an ARM64 build nor a desktop fixture is device compatibility proof.

Desktop walkthroughs capture the actual renderer backbuffer before presenting,
not only the source bitmap. Keep headless layout tests, race tests, adversarial
lifecycle tests and signed ARM64 packaging checks. Keep the existing glibc
ceiling and replace the two-library count with an exact SDL2 / SDL2_image /
SDL2_ttf / libc / libm / AArch64 loader allowlist required by the toolkit. Grout bundles SDL2_gfx; we
link it statically in device builds rather than assume the firmware has it or
install a shared library. Desktop builds need its development package too.

## Alternatives considered

- **Keep the local C wrapper:** smallest dependency footprint, but retains a
  private SDL binding and does not meet the approved toolkit migration.
- **Replace everything with Grout's stock list/router/screens:** more shared
  widgets, but loses the chosen card flow and adds input/transition policy.
  Grout's downloader must not replace Store verification and transactions.
- **Rewrite in C++/SDL, Python, or a web UI:** possible, but duplicates the
  working Go service or requires a bridge/runtime. It does not improve the
  verified installation contract. C++ in the game picker is not evidence that
  the Store needs a second language.
- **Implement several firmware profiles now:** no authorized device evidence
  or integration contract. Keep KNULLI first instead of speculative interfaces.

## Consequences

### Positive

- The local C wrapper is removed; framework code owns window and renderer
  lifetime while the tested card and semantic controls remain unchanged.
- Catalogue and installer code remain independent of presentation. The CLI
  still builds with `CGO_ENABLED=0`; no backend policy is replaced.
- Captures now exercise the real texture upload, viewport and render path.

### Negative

- The GUI has a larger Go dependency graph and now links SDL2_image and
  SDL2_ttf as well as SDL2 and libc. Do not replace firmware libraries to make
  it run. Missing libraries require a compatible firmware, not a bypass.
- Gabagool initializes global SDL/font/controller state and can exit on startup
  errors. Run one frontend per process. Check both dummy/software and real
  display paths; an initialization-only unit test is insufficient.
- The custom card renderer remains application code. This migration does not
  promise stock widgets, a new visual design, or other-CFW support.
- The new binary needs physical KNULLI control, launch/return and display
  retesting. Earlier device reports do not verify this toolkit build.
