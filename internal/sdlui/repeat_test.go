package sdlui

// Repeat-schedule tests. Tag-free: the schedule is device-agnostic, and only the
// polling that feeds it needs SDL.

import (
	"testing"
	"time"

	storeinput "github.com/jellydn/knulli-app-store/internal/input"
)

func TestNavRepeatWaitsThenRepeatsOnTheInterval(t *testing.T) {
	var repeat NavRepeat
	start := time.Unix(0, 0)
	if dir, due := repeat.Held(storeinput.Down, start); due || dir != "" {
		t.Fatalf("the first hold repeated immediately: %q %v", dir, due)
	}
	if _, due := repeat.Held(storeinput.Down, start.Add(repeatInitialDelay-time.Millisecond)); due {
		t.Fatal("a repeat fired before the initial delay")
	}
	dir, due := repeat.Held(storeinput.Down, start.Add(repeatInitialDelay))
	if !due || dir != storeinput.Down {
		t.Fatalf("held direction reported %q %v at the initial delay", dir, due)
	}
	if _, due := repeat.Held(storeinput.Down, start.Add(repeatInitialDelay+repeatInterval-time.Millisecond)); due {
		t.Fatal("a repeat fired before the interval elapsed")
	}
	if _, due := repeat.Held(storeinput.Down, start.Add(repeatInitialDelay+repeatInterval)); !due {
		t.Fatal("held direction did not repeat on the interval")
	}
}

func TestNavRepeatStartsTheDelayOverForANewDirection(t *testing.T) {
	var repeat NavRepeat
	start := time.Unix(0, 0)
	repeat.Held(storeinput.Down, start)
	if _, due := repeat.Held(storeinput.Right, start.Add(time.Second)); due {
		t.Fatal("a new direction repeated before its own delay")
	}
	if _, due := repeat.Held(storeinput.Right, start.Add(time.Second+repeatInitialDelay)); !due {
		t.Fatal("a new direction never repeated")
	}
}

func TestNavRepeatReleaseAndEmptyDirectionForgetTheHold(t *testing.T) {
	var repeat NavRepeat
	start := time.Unix(0, 0)
	repeat.Held(storeinput.Up, start)
	repeat.Release()
	if repeat.Direction() != "" {
		t.Fatalf("release kept %q held", repeat.Direction())
	}
	if _, due := repeat.Held(storeinput.Up, start.Add(time.Hour)); due {
		t.Fatal("a released direction still repeated")
	}
	repeat.Held(storeinput.Up, start)
	if _, due := repeat.Held("", start.Add(time.Hour)); due {
		t.Fatal("an empty direction repeated")
	}
	if repeat.Direction() != "" {
		t.Fatalf("an empty direction left %q held", repeat.Direction())
	}
}
