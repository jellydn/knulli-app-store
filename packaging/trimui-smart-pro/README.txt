KNULLI APP STORE - TRIMUI SMART PRO EXPERIMENTAL BUILD

This build has not passed a real-device test. It targets Linux aarch64,
the TrimUI Smart Pro 1280x720 display, and Knulli's SDL2 runtime.

INSTALL

1. On your computer, verify the ZIP against SHA256SUMS.txt.
2. Extract the ZIP directly into /userdata/roms/ports on the Knulli card.
3. Confirm these paths exist:
   /userdata/roms/ports/Knulli App Store.sh
   /userdata/roms/ports/knulli-app-store/knulli-app-ui
4. Safely eject the card, start Knulli, and refresh game lists or reboot.
5. Open Knulli App Store from the Ports system.

CONTROLS

D-pad: move through packages and actions
SDL A: select or confirm
SDL B: back or cancel

With Knulli's default Ports layout, physical B (south) maps to SDL A and
physical A (east) maps to SDL B. A per-game Xbox-layout setting reverses
those physical prompts. The app uses Knulli's SDL_GAMECONTROLLERCONFIG;
it does not guess raw event or button numbers.

CURRENT SAFETY STATE

All bundled packages are review candidates. They are read-only and cannot
be installed. A package becomes actionable only after its manifest passes
the repository review and compatibility policy.

TROUBLESHOOTING

If the app returns to EmulationStation, inspect:
  /userdata/system/logs/knulli-app-store.log

Do not replace Knulli's SDL library with a desktop or stock TrimUI copy.
