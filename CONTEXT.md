# Domain glossary

Shared vocabulary for Knulli App Store. Use these terms in code, documentation, and review; add a term here when a new concept is named.

## Catalogue

**Package** — A utility, theme, or integration offered for installation on a reviewed Knulli device. Identified by a reverse-domain id.

**Manifest** — The reviewed JSON contract for one package. A README statement is not evidence; the manifest plus its evidence URLs are.

**Candidate** — A manifest that has not completed technical review. It declares no release, compatibility, or install fields, and it is never actionable.

**Community approval** — The record that a package's provenance was accepted by the community. Kept separate from technical status; one cannot substitute for the other.

**Technical status** — The review state that gates whether anything can be done to a package. Only statuses that passed technical review are actionable.

**Release** — One pinned, immutable version of a package: an HTTPS URL, exact byte size, SHA-256, and archive format. Mutable or aliased assets are not releases.

**Catalogue index** — The deterministic, signable build of all reviewed manifests.

**External provider** — A catalogue this project features but does not own, mirror, or install from (PortMaster). Two launch paths stay visibly separate.

## Install lifecycle

**Operation** — One of the five lifecycle actions: install, adopt, update, repair, uninstall.

**Install** — First ownership: download, verify, and write the release when nothing exists at the destination.

**Adopt** (shown to users as **Manage existing**) — Take ownership of files that already exist at the destination and match the reviewed release. Content that does not match is backed up, not overwritten silently.

**External installation** — Files already present at the destination that this installer did not write. Never deleted; backed up before any overwrite. Detection counts files, not directories, so an empty skeleton left by a rolled-back write is not an external installation.

**Update** — Move an owned installation to a different reviewed release.

**Repair** — Reapply exactly the installed release. Refused when the requested release is not the installed one.

**Uninstall** — Remove owned managed files, restore originals where they exist, and release menu ownership. Preserved and unmanaged files stay.

**Outcome** — What an operation reports after committing: whether the game list refresh was accepted, or a restart is required. The operation is committed either way.

**Originals** — Backups of files the installer replaced, kept so a failed operation can be reversed and an uninstall can restore what was there before.

**Managed file** — A file the installer owns and verifies.

**Preserved file** — A file in the release inventory whose contents belong to the user at runtime (databases, settings, hooks). Written once, never overwritten by a later operation.

**Unmanaged file** — An adopted file whose content did not match the reviewed release. Tracked so it is never claimed as owned, never verified, never removed.

**Installed state** — The record of what an operation left behind: manifest, files, originals, menu ownership. Written after every file change has committed, never before.

**Health** — The check that managed files still match what was installed, in content and in the mode the filesystem actually applied. Every immutable managed file is hashed on each check. A wider mode applied by the destination filesystem is accepted; only a lost owner read, write, or execute permission fails. Preserved and unmanaged files are exempt.

## Writes

**Allowed write paths** — The only destinations an operation may touch. All writes stay below `/userdata`.

**Transaction** — A reversible group of writes. Snapshots restore in reverse order when any step fails.

**Menu ownership** — The installer owns a `gamelist.xml` entry only when it created that exact entry and it is unchanged. Shared or modified entries are never edited or deleted.

**Game list refresh** — The loopback request asking Knulli to reload menus after a committed menu change. Accepted means queued, not completed; a refusal is reported as restart required while the operation stays committed.

## Platform

**Platform detection** — Reading firmware, version, architecture, device, and resolution from Knulli-owned files. Every field records its raw value, its source, and its normalized value.

**Compatibility** — The match between a package's declared matrix and the detected platform. A failure must show the raw value, the normalized value, the source, and the full detected matrix.

**Device** — A specific handheld identity read from a Knulli board file. Device support is explicit per device, never inferred from a shared SoC family.

**Resolution** — An explicit WIDTHxHEIGHT from a real source. No resolution is ever assumed; every actionable manifest names the resolutions it supports.

## Interface

**Semantic action** — One of the eight controls the GUI exposes: up, down, left, right, confirm, back, diagnostics, and exit. An action is named independently of any physical button, so a device layout never changes what an action means.

**Binding** — The physical SDL GameController button that carries one semantic action for one device and controller identity. Bindings are detected, tested, and saved per identity; no physical position is assumed.

**Verb** — The word an action carries in a footer hint, for example `Confirm` for the confirm action. Directional actions share the `Navigate` footer verb, while setup prompts and mapping summaries keep the direction visible.

**Footer hint** — The single line below every screen's panel. It names each available action beside its binding, for example `Confirm (A)  Back (B)  Settings (Y)`, or gives an actionless calibration instruction. Outside the footer, mapping summaries identify saved bindings but do not instruct the user to press them.

**Keyboard binding** — A binding carried by a desktop keyboard key rather than a pad button, used when no GameController exists so a development machine can reach every screen. It is a binding set like any other, with its own identity, so a desktop run never reads or overwrites a mapping saved for a handheld controller.

## Verification

**Fixture root** — A throwaway directory holding the Knulli files platform detection reads, written for desktop verification only. Installs, logs, and mappings are written below it, so it stands in for a device root without a device and is never a real device root.

**Walkthrough** — A scripted key sequence a desktop run replays through the real binary. It records the opening frame, each post-key frame, and operation completions. A walkthrough is evidence that a flow still works, not a substitute for a device run: it never exercises a real GameController, a real `/userdata` write, or the kernel's SDL loading.
