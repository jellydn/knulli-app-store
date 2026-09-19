package ui

import (
	"testing"
	"time"
)

// A notice expires on its own: reading the queue after its moment has passed is
// what removes it, which is why the frame loop keeps no timer of its own.
func TestToastsExpireOnReadAfterTheTTL(t *testing.T) {
	start := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	var toasts Toasts
	toasts.Push("Install completed", start)

	if got := toasts.Live(start); len(got) != 1 || got[0] != "Install completed" {
		t.Fatalf("a fresh notice reads %v", got)
	}
	if got := toasts.Live(start.Add(ToastTTL - time.Millisecond)); len(got) != 1 {
		t.Fatalf("a notice expired early: %v", got)
	}
	if got := toasts.Live(start.Add(ToastTTL)); len(got) != 0 {
		t.Fatalf("a notice outlived its moment: %v", got)
	}
	// Expiry is final: nothing brings the notice back on a later read.
	if got := toasts.Live(start); len(got) != 0 {
		t.Fatalf("an expired notice returned: %v", got)
	}
}

// Notices keep the order they arrived in, and a burst expires one at a time
// rather than all at once.
func TestToastsKeepTheirOrderAndExpireOneByOne(t *testing.T) {
	start := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	var toasts Toasts
	toasts.Push("Install completed", start)
	toasts.Push("Game list refresh requested", start.Add(time.Second))

	got := toasts.Live(start.Add(time.Second))
	if len(got) != 2 || got[0] != "Install completed" || got[1] != "Game list refresh requested" {
		t.Fatalf("notices read %v", got)
	}
	got = toasts.Live(start.Add(ToastTTL))
	if len(got) != 1 || got[0] != "Game list refresh requested" {
		t.Fatalf("the older notice did not expire alone: %v", got)
	}
}

// An empty notice is nothing to say, so it never takes a line from a notice
// that does.
func TestEmptyNoticeIsNotQueued(t *testing.T) {
	var toasts Toasts
	toasts.Push("", time.Now())
	if got := toasts.Live(time.Now()); len(got) != 0 {
		t.Fatalf("an empty notice reads %v", got)
	}
}
