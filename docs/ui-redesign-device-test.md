# UI redesign device test (unpublished)

This test checks the stable package list, persistent detail pane and outlined
focus on real KNULLI hardware. It does not replace the installed Store. Use only
the ZIP for the exact device: TrimUI Smart Pro (1280×720) or MagicX Zero 28
(640×480). A result on one device does not verify the other.

## Install the test copy

1. Download the target's `knulli-store-ui-test-…zip` and `SHA256SUMS.txt` into
   one directory on your computer. Run
   `sha256sum --ignore-missing -c SHA256SUMS.txt`; continue only on `OK`.
2. Shut down KNULLI. Connect its data card to your computer.
3. Extract the test ZIP into `/userdata/roms/ports`. It adds only
   `Knulli Store UI Test.sh` and `knulli-store-ui-test/`. Do **not** extract a
   normal Store release ZIP over the installed Store for this test.
4. Safely eject and boot KNULLI. Refresh game lists from EmulationStation.
5. Launch **Knulli Store UI Test** in Ports, not **Knulli App Store**.

The launcher checks the board and architecture. `BUILD-ID` and `TARGET` beside
the test binaries identify the exact build. Keep binaries, catalogue and
signature together. No release, remote branch, firmware library or production
Store file is changed by preparing these artifacts.

## Isolation and limits

The test root is
`/userdata/system/knulli-store-ui-test/<BUILD-ID>/<TARGET>/root`.
The launcher links device-owned system paths there for **read-only platform
detection**; it does not fabricate firmware, device identity or resolution.
Its `userdata/` is a separate real directory, never a link to production data.
The unchanged Store manager directs package files, state, logs, backups and
game-list writes below that test root. Menu registration and Store self-update
are disabled by the separate launcher and nonstandard location.

This is filesystem-path isolation, not an OS sandbox. Do not launch an installed
application from its nested test directory: third-party apps can use absolute
paths and modify real data. Do not test power loss or production cleanup hooks.
Some packages can show a missing PortMaster dependency when the dependency only
exists in production userdata; do not copy live settings or create links into
production userdata to bypass this. On exit, a session with committed menu
changes can request EmulationStation's real loopback game-list refresh once;
it does not add the nested test packages to the real game list.

## Phase 1 — no package changes

Record the build ID, ZIP checksum, exact KNULLI version, board, display size,
controller name/GUID and Ports layout mode. Take a screenshot of each result.

- Open the test twice; confirm the second launch works and quit returns to ES.
- Browse with D-pad up/down and left/right. Row heights must stay fixed; focus
  must be visible. The selected package's details must match its row.
- Check long names and summaries, separate test/install status, and the last
  row. No text may overlap the detail pane, status area or footer.
- Use L1/R1 to visit All, Ready and the empty Installed filter, then return.
  Check selection memory. An empty filter must still show a way out.
- Press A once to review an install, then B. No operation must start. Selection
  must return to the same package. Start must **not** confirm.
- Press Y for More, then B; repeat and press Y to export diagnostics. Check the
  contextual footer, complete warning text and clean return to the list.
- Hold Select and press Y to quit. Repeat these controls in KNULLI default and
  Ports Xbox-layout modes using the existing ES setting; restore the original
  setting afterwards. Do not create or edit a system mapping file.
- If the device permits disconnect/reconnect, check the blocked-controller
  screen and return. Otherwise mark that case not tested.
- Leave the catalogue idle for 60 seconds and record CPU/RSS if available.
  This is a measurement, not an FPS or battery-life claim.

The production Store and its installed packages must remain unchanged. The test
should initially show no owned installed packages, even if production has some.

## Phase 2 — optional disposable lifecycle checks

Complete Phase 1 first. In the **test copy only**, install Random Game Picker
0.2.2. Do not launch it or select Play. Confirm progress locks navigation, the
completion notice clears, and the Installed filter now contains the package.

Use More → Repair and then Uninstall, with separate confirmation each time.
Check that files and state changed only below the test root. For a health/error
check, stop the test, change only a disposable managed file **inside that root**,
reopen, verify Issue opens health first, then Repair. Never damage a production
file. Cancellation before the second confirmation must not write package files.

For the repeated-refresh regression, complete install, repair, uninstall and
install again in **one session**, without launching the package. There must be
no game-list refresh during these operations. Quit once: the diagnostic log
must contain exactly one `gamelist_refresh_accepted` or
`gamelist_refresh_unavailable` event for that session, not one per operation.
An idle/cancel-only session must contain neither event. A refused refresh must
not undo the committed installation; restart ES manually if required.

Update and two-stage recovery have desktop evidence; they need separately
prepared old-version/error fixtures before a native test. Do not alter an
installed-state JSON or a signed catalogue by hand to force either state.

## Return results and remove the test

Logs are below `<test-root>/userdata/system/logs/`; diagnostic exports are below
`<test-root>/userdata/system/knulli-app-store/diagnostics/`. The launcher also
keeps `<test-home>/launcher.log`. Inspect exports for private data before sharing.
Report each case as pass, fail or not tested, with screenshots, build ID,
checksum, exact firmware and control mode. A UI test does not grant a package
device-tested badge or establish gameplay compatibility.

Close the test and shut down KNULLI before removal. On the connected card, remove
only `Knulli Store UI Test.sh`, `knulli-store-ui-test/` and this build's directory
below `/userdata/system/knulli-store-ui-test/`. Keep the production Store,
`/userdata/system/knulli-app-store`, installed apps, settings and ROMs. Boot and
refresh game lists again.
