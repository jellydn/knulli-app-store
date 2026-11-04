# Domain glossary

Shared vocabulary for Knulli App Store. Use these terms in code, documentation, and review; add a term here when a new concept is named.

## Catalogue

**Package** — A utility, theme, integration, or reviewed free game offered for installation on a Knulli host. Identified by a reverse-domain id. Games include legally redistributable homebrew ROMs and game data, not commercial files or ROM-download services.

**Game launch file** — A native launcher, homebrew ROM, or game-data file that Knulli's EmulationStation opens with its configured runtime. Recorded as `install.launcher`; ROMs and data are not made executable. Compatibility architecture and ABI describe the Knulli host, not the ROM's CPU.

**Manifest** — The reviewed JSON contract for one package. A README statement is not evidence; the manifest plus its evidence URLs are.

**Candidate** — A manifest that has not completed technical review. It can retain a validated, pinned release for incremental review, but declares no compatibility or install fields and is never actionable.

**Community approval** — The record that a package's provenance was accepted by the community. Kept separate from technical status; one cannot substitute for the other.

**Technical status** — The review state that gates whether anything can be done to a package. Only statuses that passed technical review are actionable.

**Release** — One pinned, immutable version of a package: an HTTPS URL, exact byte size, SHA-256, and archive format. Mutable or aliased assets are not releases.

**Embedded archive** — A base64-encoded `tar.gz` payload at a reviewed byte offset in a pinned release. The whole release is size- and checksum-verified before decoding. The prefix is data, never an installer to execute. Normal archive path, type, byte-budget, and cancellation checks still apply.

**Catalogue index** — The deterministic, signable build of all reviewed manifests.

**External provider** — A catalogue this project features but does not own, mirror, or install from (PortMaster). Two launch paths stay visibly separate.

**Tab** — One view of the catalogue: the whole list, the packages with something to act on, or the packages installed on this device. A tab filters on a package's typed state, never on its rendered state label.

**Library browsing** — Moving through a stable package list with up/down. An outline marks the selected row. Two panes show package metadata where space permits; compact screens use one column with a fixed selected-package summary. Neighbouring rows stay still. Y opens More for the complete title, version, summary, compatibility, warnings and secondary actions. A opens confirmation for the first permitted action; an issue opens its health check first. Layout depends on drawable size, never device identity.

**Category** — All types, Games or Apps, selected with left/right while browsing. Games match the manifest's `game` type; Apps match utilities, themes and integrations, including tools with “game” in their name. A category does not grant installation permission or change technical status.

**Filtering** — Switching All, Ready and Installed status tabs with L1/R1 (Page Up/Page Down on a keyboard), within the selected category. Each category/status combination remembers its selected package. Empty views still permit changing either selector. Selectors are locked during operations and package review screens.

**Review paging** — L1/R1 (Page Up/Page Down) reads the preceding/next measured window of text in More, Health and confirmation screens without changing the category, status tab or selected action. The visible line range and total show when more text exists. All warning, recovery and compatibility text remains reachable, including long unbroken words. A still confirms and B returns; reading text does not execute an operation. Busy screens reject paging.

## Install lifecycle

**Operation** — One of the five lifecycle actions: install, adopt, update, repair, uninstall.

**Install** — First ownership: download, verify, and write the release when nothing exists at the destination.

**Adopt** (shown to users as **Manage existing**) — Take ownership of files that already exist at the destination and match the reviewed release. Content that does not match is backed up, not overwritten silently.

**External installation** — Files already present at the destination that this installer did not write. Never deleted; backed up before any overwrite. Detection counts files, not directories, so an empty skeleton left by a rolled-back write is not an external installation.

**Update** — Move an owned installation to a different reviewed release.

**Repair** — Reapply exactly the installed release. Refused when the requested release is not the installed one.

**Uninstall** — Remove owned managed files, restore originals where they exist, and release menu ownership. Preserved and unmanaged files stay.

**Outcome** — What an operation reports after committing: the GUI queues a game list refresh for exit; a one-shot CLI operation reports whether its refresh was accepted or a restart is required. The operation is committed either way.

**Interruptible** — A step that honours the caller's context, so cancelling stops it where it stands instead of at the end. Extraction is interruptible between archive entries and partway through a single entry's transfer, so a large release stops promptly. Only the staging directory has been written when it stops, so an interrupted operation needs no rollback: the staging work is discarded and the destination is left exactly as it was found.

**Notice** — A transient line reporting a finished operation, shown in the notice bar and cleared after three seconds. A failure is not a notice: it stays in the status block until it is answered.

**Originals** — Backups of files the installer replaced, kept so a failed operation can be reversed and an uninstall can restore what was there before.

**Managed file** — A file the installer owns and verifies.

**Upstream-managed package** — An explicitly warned experimental package installed by verified download and extraction. The app, not the Store, controls first-launch setup, runtime writes, runtime downloads and its own updates. Store Repair restores pinned release files and keeps declared user data. Health checks disclose upstream-modified files; they do not claim those bytes were reviewed.

**Uninstall check** — A read-only check for a shared hook reference or an existing service-state path. A match blocks removal before any package mutation and explains the required manual cleanup. The Store does not execute the cleanup or remove shared hooks.

**Preserved file** — A file in the release inventory whose contents belong to the user at runtime (databases, settings, hooks). Written once, never overwritten by a later operation.

**Unmanaged file** — An adopted file whose content did not match the reviewed release. Tracked so it is never claimed as owned, never verified, never removed.

**Installed state** — The record of what an operation left behind: manifest, files, originals, menu ownership. Written after every file change has committed, never before.

**Health** — The check that managed files still match what was installed, in content and in the mode the filesystem actually applied. Every immutable managed file is hashed on each check. A wider mode applied by the destination filesystem is accepted; only a lost owner read, write, or execute permission fails. Preserved and unmanaged files are exempt.

## Writes

**Allowed write paths** — The only destinations an operation may touch. All writes stay below `/userdata`.

**Transaction** — A reversible group of writes. Snapshots restore in reverse order when any step fails.

**Menu ownership** — The installer owns a `gamelist.xml` entry only when it created that exact entry and it is unchanged. For an existing exact launcher path, registration and Repair can fill blank name, description, and image fields without taking ownership. Existing nonblank values, play history, and custom fields remain. Shared or modified entries are never replaced or deleted.

**Game list refresh** — The loopback request asking Knulli to reload menus after a committed menu change. The GUI combines all such changes into one request on exit, after any active operation finishes. An unchanged, cancelled or failed-only session sends none. One-shot CLI commands request it immediately. Accepted means queued, not completed; a refusal is reported as restart required while the operation stays committed.

## Platform

**Platform detection** — Reading firmware, version, architecture, device, and resolution from Knulli-owned files. Every field records its raw value, its source, and its normalized value.

**Compatibility** — The match between a package's firmware, architecture, ABI, and runtime requirements and the detected platform. A failure must show the raw value, the normalized value, the source, and the full detected matrix. Device lists and display bounds record reviewed scope, not installation permission.

**Device** — A specific handheld identity read from a Knulli board file, never inferred from a shared SoC family. An unknown identity does not block installation.

**Resolution** — An explicit WIDTHxHEIGHT from a real source. No resolution is ever assumed. Missing resolution or a size outside a manifest's reviewed scope does not block installation.

**Drawable size** — The current SDL renderer output in pixels. It determines layout and can change during a session. It is separate from device identity, CPU/ABI, input mapping and firmware requirements. Resizing preserves selection, review focus and running operations; a transient invalid size retains the last valid surface and never creates device-test evidence. The initial minimum UI target is 320×240, not proof of physical device or binary compatibility.

**Store build profile** — `knulli-aarch64-glibc-sdl2`, the shared Store distribution contract for KNULLI, Linux AArch64, glibc and system SDL2/image/ttf, with SDL2_gfx static. It is not a device name, layout selector or universal compatibility claim. One installable ZIP replaces the former device-named distributions at unchanged paths; its checksum sidecar is separate.

**Legacy bridge** — A separately approved compatibility delivery that an old device-prefix updater can discover and verify before moving to the shared profile. New code accepting legacy metadata does not patch already-installed binaries. Current release inventory and old version/first-asset rules prevent a verified automatic bridge; manual verified seven-file replacement is the supported migration. Local compatibility-copy preparation is not publication or proof of eligibility.

**Untested** — No passed device report matches the exact package version, firmware, architecture, device, and resolution. The UI shows UNTESTED and still offers installation when the package and runtime checks pass; evidence for another device never grants a tested badge.

## Interface

**Semantic action** — One of the nine controls the GUI exposes: up, down, left, right, page up, page down, confirm, back, and diagnostics. An action is named independently of any physical button, so a device layout never changes what an action means. Exit is not one of them: quitting is the Select chord, so no single press carries it.

**Binding** — The logical SDL GameController button carrying a semantic action. Automatic defaults use ES A for confirm, B for back, Y for More on a card and diagnostics inside More, D-pad navigation, and L1/R1 filtering. Empty catalogues and blocked-controller screens still export diagnostics directly. No custom assignments are loaded or saved.

**Controller layout** — The launcher-selected letter policy: `STORE_CONTROLLER_LAYOUT=knulli` accounts for Knulli's ES a/b/x/y → SDL b/a/y/x mapping; `xbox` accounts for Ports' Xbox-layout flag and uses direct SDL letters. An unset value uses normal desktop SDL. Events are never swapped again. Footer letters follow the selected ES layout, not guessed raw indices.

**Verb** — The word an action carries in a footer hint, for example `Confirm` for the confirm action. Directional actions share `Navigate` and paging actions share `Page`.

**Header** — The page identity and current category/status intersection, for example `Library` and `GAMES / INSTALLED`. Show the current values, not a second set of controller instructions or focusable filter controls. Keep both values visible even when no package matches. The footer owns the controls; selection memory and filter locking are unchanged. Below the 320×240 target, optional review identity can yield to reachable text while the filter state stays visible.

**Footer hint** — A measured button chip and complete action label, for example `A Manage existing` or `L1/R1 Read`. Whole groups wrap into as many rows as required, with safe margins and a minimum gap; actions are not dropped or shortened to force a fit. Footer and status height are reserved before body layout. The 320×240 target retains selected-package trust/install state and complete reachable reviews. Face labels use the selected ES letters and keyboard runs name their keys. Quit is always Select+Y (Tab/F5+Y on a keyboard), never a lone face button. Status filtering applies only while browsing; Read pages review text.

**Keyboard binding** — A fixed desktop key: arrows browse, Page Up/Page Down filters while browsing and pages text while reviewing, Enter confirms, Escape backs out, Y opens More (Y there exports diagnostics), and held Tab (or F5) plus Y quits. Keyboard input requires no setup or persistence.

## Verification

**Fixture root** — A throwaway directory holding the Knulli files platform detection reads, written for desktop verification only. Installs and logs are written below it, so it stands in for a device root without a device and is never a real device root.

**Walkthrough** — A scripted key sequence a desktop run replays through the real binary. It records the opening frame, each post-key frame, and operation completions. A walkthrough is evidence that a flow still works, not a substitute for a device run: it never exercises a real GameController, a real `/userdata` write, or the kernel's SDL loading.
