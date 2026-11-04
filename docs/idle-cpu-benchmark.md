# Store CPU interval benchmark

This is the repeatable measurement procedure for [issue #73](https://github.com/jellydn/knulli-app-store/issues/73).
It does not grant device-test evidence or close that issue. Physical controls,
audible restoration and Store-installed Picker gameplay remain #74 and #75.

## Existing change, not a new optimization

The original continuously drawn Store is commit
`14490119b819f524a8a59919b949359092b3b014`. Commit
`e5c14f011c9b113b0d356aa321cac018b53d153d` already added dirty rendering:
normal runs poll input/model at 30 Hz, but draw/upload/present only after SDL
events, held navigation, model changes or notice expiry. Do not apply another
redraw optimization without profiling the current code.

The historical desktop experiment reported three six-second intervals after
two seconds of warm-up: 19.17/18.33/20.00% before and 1.00/1.00/1.17% after,
normalized to one CPU core, with dummy/software SDL at 1280×720. Its temporary
Python helper was not retained. These short historical host samples are not
the required controlled handheld comparison. The original handheld `ps` 78%
sample was a lifetime average after interaction, not idle CPU or battery use.

## Read-only interval sampler

`scripts/measure-process.sh` needs Linux `/proc`, POSIX `sh`, `awk`, `sleep`
and `getconf` with `CLK_TCK` and `PAGESIZE`. Check these tools on the device;
do not replace firmware libraries or install tools without owner approval.
The script reads an existing PID. It does not launch, stop, signal or alter
the Store, controller, accounts, saves or platform files. Redirecting its
output writes only to the path you select.

```sh
# PID is the actual GUI PID, not its launcher, CLI or sampler shell.
sh scripts/measure-process.sh "$PID" 60 3 5 > build/idle-library.tsv
```

Arguments are PID, interval seconds (default 60), repeat count (3), and warm-up
seconds (5). One invocation emits three consecutive intervals after one warm-up.
Keep the Store open and do not suspend the device during the measurement.
With an already approved SSH session, the same reviewed local script can read
the device process while the report stays on the host:

```sh
# Replace DEVICE and PID with the approved host and actual GUI PID.
ssh DEVICE 'sh -s -- PID 60 3 5' < scripts/measure-process.sh > build/device-idle.tsv
```

Each TSV row retains both uptime endpoints, user+system CPU tick totals,
queried ticks/second, process start-time identity and ending resident KiB.
CPU percent is 100 × (end ticks − start ticks) / ticks per second / elapsed
seconds. It is not divided by the number of cores: one fully used core is
100%. RSS is the ending resident-page count × queried page size / 1024,
not peak memory or proof of bounded growth. Uptime has hundredth-second
resolution; use long intervals, not this tool for input latency.

Process death, zombie state or a changed start-time identity fails the command.
Do not use partial stdout as a successful run if its exit status is nonzero.
The sampler excludes child-process CPU; it measures the GUI process only.
Avoid recording command lines or environment dumps: they can expose secrets.

## Before/after protocol

1. Obtain owner approval for both exact builds and the scratch package/root.
   Record Store source and binary SHA-256, catalogue SHA-256, public ZIP hash
   if used, exact firmware build, device identity/source, architecture/ABI,
   drawable resolution, SDL versions, renderer/backend and launch layout.
   A platform override or fixture file is not detected hardware evidence.
2. Use the same catalogue, selected package, screen, input/layout, brightness,
   governor/power state and renderer for both builds. Record background work,
   network activity and temperature where available. Do not change firmware
   settings merely to improve a result. Alternate build order across repeats
   if drift is observed. Never mix a signed public catalogue with another GUI.
3. Warm up and record at least three 60-second no-input intervals in Library,
   More/confirmation and an empty filter. Run normal interactive mode: **no
   `-keys`, `-shot-dir` or `-screenshot`**. Those evidence modes deliberately
   draw every frame and cannot measure dirty-render idle behavior.
4. In a separate active case, record the same paced input sequence and timing
   for both builds, including expected and observed selection. CPU sampling
   alone cannot detect lost events or measure physical latency. Record
   filtering/focus, confirmation/cancel and Select+Y/ES return independently.
5. Exercise progress, completion, failure and three-second notice expiry with
   no follow-up input. Check held/released navigation, controller disconnect/
   reconnect and expose/resize where supported. Record RSS over repeated
   operation/browse cycles; a single ending sample is not a leak test.
6. Publish raw TSVs and per-repeat results, range and median, plus matched
   profiles of drawing, allocations, upload and pacing where available.
   State the achieved CPU reduction and observed variation. Do not compare
   unlike renderers, catalogues or hardware as a causal before/after result.
   Battery, FPS and physical-input claims need their own measurements.

## Owner evidence still required

| Matrix | Required next run | Status |
| --- | --- | --- |
| TSP, KNULLI, AArch64, 1280×720 | Matched controlled before/after idle and active runs; asynchronous states and physical responsiveness | Not run here |
| MagicX, KNULLI, AArch64, 640×480 | Same benchmark on the actual handheld | Not run here |
| Linux desktop, 1280×720 and 640×480 | Normal-mode intervals plus software/dummy walkthrough and capture regressions | Host checks only; never a device substitute |

Use the [device report checklist](real-device-tests.md) and
[desktop walkthrough](desktop-verification.md). An owner must physically press
buttons, listen before/after ES return, and supply legal local game data for
Picker's normal-menu Play/hotkey round trip. No device access, physical-input
result, audio result, gameplay result or new tested badge is claimed here.
