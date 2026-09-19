package sdlui

// Held-direction navigation repeat.
//
// A pad sends one press when a D-pad button goes down, so a catalogue longer
// than one screen would crawl if the selection only moved on presses. The
// keyboard needs none of this: the OS repeats its own keys, and Handle lets
// those repeats navigate.
//
// This file stays device-agnostic on purpose. The SDL adapter polls the pad and
// asks this whether a repeat is due, so the schedule is testable without cgo.
// The timings match RetSend's input configuration: a short delay before the
// first repeat, then a steady interval.

import (
	"time"

	storeinput "github.com/jellydn/knulli-app-store/internal/input"
)

const (
	// repeatInitialDelay is how long a direction is held before it repeats.
	repeatInitialDelay = 350 * time.Millisecond
	// repeatInterval is the gap between repeats once they have started.
	repeatInterval = 110 * time.Millisecond
)

// NavRepeat tracks one held direction. The zero value holds nothing, so a run
// starts with nothing repeating.
type NavRepeat struct {
	dir     storeinput.Action
	started time.Time
	last    time.Time
}

// Held records the direction the pad is holding and reports whether a repeat is
// due now. A different direction starts the delay over, so switching direction
// never inherits the previous one's cadence.
func (repeat *NavRepeat) Held(dir storeinput.Action, now time.Time) (storeinput.Action, bool) {
	if dir == "" {
		repeat.Release()
		return "", false
	}
	if repeat.dir != dir {
		repeat.dir, repeat.started, repeat.last = dir, now, time.Time{}
		return "", false
	}
	if repeat.last.IsZero() {
		if now.Sub(repeat.started) < repeatInitialDelay {
			return "", false
		}
		repeat.last = now
		return dir, true
	}
	if now.Sub(repeat.last) < repeatInterval {
		return "", false
	}
	repeat.last = now
	return dir, true
}

// Release forgets the held direction, so the next press starts the delay over.
func (repeat *NavRepeat) Release() {
	repeat.dir, repeat.started, repeat.last = "", time.Time{}, time.Time{}
}

// Direction returns the direction being held, or an empty action.
func (repeat *NavRepeat) Direction() storeinput.Action {
	return repeat.dir
}
