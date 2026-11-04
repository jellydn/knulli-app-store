#!/bin/sh
# Read-only Linux process intervals; never launch, signal or change the target.
# Usage: sh scripts/measure-process.sh PID [SECONDS [REPEATS [WARMUP]]]
set -eu

if [ "$#" -lt 1 ] || [ "$#" -gt 4 ]; then
  echo "usage: $0 PID [SECONDS=60 [REPEATS=3 [WARMUP=5]]]" >&2
  exit 2
fi
pid=$1
seconds=${2:-60}
repeats=${3:-3}
warmup=${4:-5}
for value in "$pid" "$seconds" "$repeats" "$warmup"; do
  case "$value" in
    ''|*[!0-9]*) echo "arguments must be non-negative integers" >&2; exit 2 ;;
  esac
done
if [ "$pid" -eq 0 ] || [ "$seconds" -eq 0 ] || [ "$repeats" -eq 0 ]; then
  echo "PID, SECONDS and REPEATS must be positive" >&2
  exit 2
fi

# Query the host rather than assume a tick rate or memory page size.
ticks=$(getconf CLK_TCK)
page_size=$(getconf PAGESIZE)

snapshot() {
  # comm can contain spaces and parentheses. Fields after its final ')' start
  # at field 3 of proc_pid_stat(5), not at field 2 of a whitespace split.
  awk '
    NR == 1 {
      sub(/^.*\) /, "")
      if (NF < 22 || $1 == "Z" || $1 == "X") exit 1
      ticks = $12 + $13
      started = $20
      rss = $22
      next
    }
    NR == 2 { printf "%.0f %.0f %.0f %s\n", ticks, started, rss, $1 }
  ' "/proc/$pid/stat" /proc/uptime
}

initial=$(snapshot) || { echo "cannot sample live PID $pid" >&2; exit 1; }
set -- $initial
identity=$2
sleep "$warmup"
printf 'sample\tpid\tstart_uptime_s\tend_uptime_s\tstart_cpu_ticks\tend_cpu_ticks\tticks_per_second\tstart_time_ticks\telapsed_s\tcpu_percent_one_core\trss_kib\n'
sample=1
while [ "$sample" -le "$repeats" ]; do
  before=$(snapshot) || { echo "PID $pid exited before sample $sample" >&2; exit 1; }
  sleep "$seconds"
  after=$(snapshot) || { echo "PID $pid exited during sample $sample" >&2; exit 1; }
  awk -v before="$before" -v after="$after" -v identity="$identity" \
    -v sample="$sample" -v pid="$pid" -v hz="$ticks" -v page="$page_size" 'BEGIN {
      split(before, b); split(after, a)
      elapsed = a[4] - b[4]
      if (b[2] != identity || a[2] != identity || elapsed <= 0 || a[1] < b[1]) {
        print "process identity changed or invalid interval" > "/dev/stderr"
        exit 1
      }
      printf "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%.2f\t%.3f\t%.0f\n", \
        sample, pid, b[4], a[4], b[1], a[1], hz, identity, elapsed, \
        100 * (a[1] - b[1]) / hz / elapsed, a[3] * page / 1024
    }'
  sample=$((sample + 1))
done
