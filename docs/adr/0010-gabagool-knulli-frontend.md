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
The owner authorized a frontend migration, then expanded its scope to a console
layout based on the public Random Game Picker v0.2.0 implementation.

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
types instead of the local C wrapper. Use the owner-selected Library design:
a package list, visible filters and contextual button glyphs. The original
expanded selected row was replaced in the 2026-10-27 development revision below.
Keep the released picker's shared responsive
canvas: logical height 480, width max(640, output width × 480 / output height).
Use its DejaVu Sans 18/26 px scale, palette and contextual footer proportions.
Keep selection, confirmations, health detours, notices and
semantic footer vocabulary. This
is a toolkit-backed custom screen, **not** adoption of Grout's list layout or
gabagool's stock screen components.

### 2026-10-27 presentation revision (unreleased)

Use Zitch as the lead reference: its [official handheld screenshot and UI source](https://github.com/itchio/zitch/tree/0c296f6faf76787e5a7c868f2abba20a8962f282)
establish outlined controller focus, tabbed browsing, separate package pages and
persistent input hints. Adapt these patterns to a stable text list and persistent
detail pane: the Store has no reviewed covers, so a cover carousel adds no value.
Keep the Store's own navy/mint palette and licensed font; no reference assets or
code are copied. [RetroShelf's public site](https://www.retroshelf.org/) and
[KNULLI guide](https://www.retroshelf.org/help/plugins/knulli) inform quiet spacing
and clear status copy, not handheld navigation: its linked source returned 404.
Test evidence and installation state remain separate. Rendering modules own the
library, chrome, and package review; the input model and installer do not change.
Verify the same layout at 1280×720 and 640×480 with both input sources. Desktop
fixtures do not establish physical-controller or native-device results.

### 2026-11-03 capability-based presentation revision (approved, unreleased)

This revision replaces the fixed minimum 640×480 canvas and always-present
detail pane above. Keep standard outputs at a 480-high logical scale, but let
aspect ratio determine width without a 640 minimum; cap content at 960 logical
pixels on wide outputs. Below 480 output pixels high, use native pixels rather
than downsampling. Very tall outputs retain at least 320 logical pixels wide
and expand logical height instead of squeezing controls. Use 14/18 body/title sizes on short or narrow canvases and
the existing 18/26 sizes elsewhere. The initial UI target minimum is 320×240;
320×200, portrait and ultrawide cases are additional stress cases, not physical
device support claims.

Use two panes only when the canvas is at least 640 wide and 480 high. Otherwise
use a stable single-column list, a fixed selected-package summary and Y for
complete details. Derive compact row capacity from available height. Keep
reviewed offline artwork, contained with its aspect ratio intact; omit
secondary art on compact screens before reducing essential text. The footer
revision below supersedes the initial two-row hint budget.

L1/R1 (Page Up/Page Down) still filters status while browsing. On More, Health
and both confirmation screens it pages the complete measured review text,
including warnings, compatibility and recovery information. Show the visible
line range and total. Keep action selection independent from text paging,
preserve the two-step recovery confirmation, and never execute on a paging key.
The pure-Go model owns the review offset and clamps it to measured capacity.

The SDL renderer output, not the board name or a resolution override, determines
the drawable boundary. Rebuild size-dependent canvas/texture resources when
output changes; preserve the model, worker, input policy and selected action.
Retain the last valid surface during a transient invalid size. Log drawable
layout changes separately from detected platform evidence. Do not infer
rotation from portrait dimensions or rewrite exact device-test evidence on
resize. Windowed verification permits interactive resizing.

Source review of [Grout at 48b229cc](https://github.com/rommapp/grout/tree/48b229cc04002681335ba66547a267d680c152e3)
supports measured row capacity, contained artwork, firmware quirks and separate
architecture builds. Its stock widgets do not control this custom renderer.
[retroshelf's public Linux repository at 4646af74](https://github.com/retroshelf-org/linux/tree/4646af742d1f89a32158d18f5575ff58c544daa5)
contains release metadata only. Its linked implementation source still returns
404; no claim about its internal responsive UI is made. The owner approved
proceeding without that source. Painter concepts informed structure only;
actual SDL renders and navigation tests determine acceptance.

### 2026-11-03 measured footer revision (approved, unreleased)

Device testing reported overlapping hints, without an exact failing capture.
Recheck [Grout's actual footer call site](https://github.com/rommapp/grout/blob/48b229cc04002681335ba66547a267d680c152e3/ui/game_details.go)
and [gabagool v2.26.6 footer](https://github.com/BrandonKowalski/gabagool/blob/v2.26.6/pkg/gabagool/footer.go).
Grout supplies contextual button/help pairs. Gabagool measures key and help
widths, gives long keys wider chips, and groups pills left/right. It does not
wrap colliding groups; its automatic grouping can omit hints beyond four.
Adopt measured chips and contextual help, not its item limit, scaling policy,
blocking input loop or Start confirmation.

Use one footer for compact and standard rendering. Measure actual glyph ink
and advances, retain an eight-pixel group gap, and wrap whole groups into the
required rows without hiding or abbreviating actions. Reserve measured footer
and status bounds before composing/clipping the body. Library row capacity and
review page capacity use the remaining space. Keep trust/install state at the
320×240 minimum. Below-target 320×200 stress reviews may omit optional heading
and action chrome before losing warning text or the page range. All complete
details remain reachable; controller mapping, filtering, confirmation and quit
policy stay unchanged. Use actual drawable-derived geometry, including
1280×720 and all compact/aspect-ratio/resize cases, never a device-name patch.

No architecture/ABI packaging or legacy update-channel migration was included
in this presentation phase. [ADR 0013](0013-compatibility-profile-distribution.md)
subsequently authorizes one shared distribution and documents legacy limits.
Keep firmware/ABI/glibc/SDL requirements, catalogue
signing, controller letter policy and installer boundaries. Responsive output
does not imply a universal binary. Desktop evidence is not hardware evidence;
the changed build needs new physical display/controller and ES return checks
before any separately authorized release.

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
| Presentation and lifetime | `sdlui`: shared responsive console canvas, framework window/renderer, event pump and capture. Only this optional adapter imports gabagool/SDL. |

These are extension seams, not a generic firmware framework. A future CFW needs
its own reviewed profile and integration tests; none is implemented or claimed
here. The initial packaged targets were KNULLI TrimUI Smart Pro and MagicX Zero
28; ADR 0013 replaces those names with a shared compatibility profile. Neither
an ARM64 build nor a desktop fixture is device compatibility proof.

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
  lifetime while the tested semantic controls remain unchanged. The Library
  keeps neighbouring packages visible; separate centered review screens leave
  room for compatibility, health, warnings and focused actions.
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
- The custom console renderer remains application code. This migration does not
  promise stock widgets or other-CFW support. Manifests do not declare reviewed
  art, so browsing uses the local mark instead of runtime artwork downloads.
- The new binary needs physical KNULLI control, launch/return and display
  retesting. Earlier device reports do not verify this toolkit build.
