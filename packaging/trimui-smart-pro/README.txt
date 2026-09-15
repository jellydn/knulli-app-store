KNULLI APP STORE - TRIMUI SMART PRO EXPERIMENTAL BUILD

The GUI was reported functional on one TrimUI Smart Pro. This build remains
experimental until the exact Knulli release and repeatable test evidence are
recorded. Package compatibility is reviewed separately.

INSTALL

1. On your computer, verify the ZIP against SHA256SUMS.txt.
2. Extract the ZIP directly into /userdata/roms/ports on the Knulli card.
3. Confirm these paths exist:
   /userdata/roms/ports/Knulli App Store.sh
   /userdata/roms/ports/knulli-app-store/knulli-app-ui
4. Safely eject the card, start Knulli, and refresh game lists or reboot.
5. Open Knulli App Store from the Ports system.

UPDATE

Verify the new checksum, shut down Knulli, and extract the new ZIP into the
same /userdata/roms/ports directory. Allow replacement of the old launcher
and knulli-app-store files. Manager state under /userdata/system is retained.

CONTROLS

D-pad: move through packages and actions
SDL A: select or confirm
SDL B: back or cancel

With Knulli's default Ports layout, physical B (south) maps to SDL A and
physical A (east) maps to SDL B. A per-game Xbox-layout setting reverses
those physical prompts. The app uses Knulli's SDL_GAMECONTROLLERCONFIG;
it does not guess raw event or button numbers.

CURRENT SAFETY STATE

All six bundled packages are review candidates. They are read-only and cannot
be installed. EmuDrop, PlayTime, and Grout have community approval for
catalogue inclusion, not package verification. EmuDrop downloads ROMs; users
must have the required rights and comply with local copyright law.

TROUBLESHOOTING

If the app returns to EmulationStation, inspect:
  /userdata/system/logs/knulli-app-store.log

Do not replace Knulli's SDL library with a desktop or stock TrimUI copy.
