//go:build linux

package scripts_test

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestProcessSamplerReportsIntervalsAndLeavesTargetRunning(t *testing.T) {
	// A whitespace split of /proc/PID/stat misreads this process name.
	name := filepath.Join(t.TempDir(), "busy )\nworker")
	data, err := os.ReadFile("/bin/sh")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, data, 0755); err != nil {
		t.Fatal(err)
	}
	target := exec.CommandContext(t.Context(), name, "-c", "while :; do :; done")
	if err := target.Start(); err != nil {
		if os.IsPermission(err) {
			t.Skipf("temporary executable is blocked (for example, a noexec mount): %v", err)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { target.Process.Kill(); target.Wait() })
	pid := strconv.Itoa(target.Process.Pid)
	command := exec.CommandContext(t.Context(), "sh", "measure-process.sh", pid, "1", "2", "0")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("sampler: %v\n%s", err, output)
	}
	rows := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(rows) != 3 || !strings.Contains(rows[0], "cpu_percent_one_core") {
		t.Fatalf("expected header and two intervals: %s", output)
	}
	var identity, previousEnd float64
	for index, row := range rows[1:] {
		fields := strings.Split(row, "\t")
		if len(fields) != 11 || fields[0] != strconv.Itoa(index+1) || fields[1] != pid {
			t.Fatalf("unexpected sample: %s", row)
		}
		number := func(column int) float64 {
			t.Helper()
			value, err := strconv.ParseFloat(fields[column], 64)
			if err != nil {
				t.Fatal(err)
			}
			return value
		}
		elapsed := number(3) - number(2)
		if elapsed < 1 || math.Abs(number(8)-elapsed) > 0.001 || number(10) <= 0 {
			t.Fatalf("invalid elapsed time or RSS: %s", row)
		}
		delta := number(5) - number(4)
		if delta <= 0 || number(6) <= 0 || math.Abs(number(9)-100*delta/number(6)/elapsed) > 0.001 {
			t.Fatalf("CPU must use this interval, normalized to one core: %s", row)
		}
		if index > 0 && (number(7) != identity || number(4) < previousEnd) {
			t.Fatalf("identity or counter changed between intervals: %s", row)
		}
		identity, previousEnd = number(7), number(5)
	}
	if _, err := os.ReadFile("/proc/" + pid + "/stat"); err != nil {
		t.Fatalf("sampler must leave the target running: %v", err)
	}
}

func TestProcessSamplerRejectsInvalidArgumentsAndMissingProcess(t *testing.T) {
	for _, args := range [][]string{
		nil, {"0"}, {"-1"}, {"self"}, {"1", "0"}, {"1", "1", "0"},
		{"1", "1", "1", "-1"}, {"1", "1", "1", "0", "extra"}, {"999999999", "1", "1", "0"},
		{"1", "", "1", "0"}, {"1", "1", "", "0"}, {"1", "1", "1", ""},
		{"1", "1", "99999999999999999999", "0"},
	} {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		command := exec.CommandContext(ctx, "sh", append([]string{"measure-process.sh"}, args...)...)
		command.WaitDelay = time.Second
		output, err := command.CombinedOutput()
		timedOut := ctx.Err() != nil
		cancel()
		if timedOut {
			t.Fatalf("must reject %q without starting an interval", args)
		}
		if err == nil || strings.Contains(string(output), "cpu_percent_one_core") {
			t.Fatalf("must reject %q before emitting samples: %v\n%s", args, err, output)
		}
	}
}
