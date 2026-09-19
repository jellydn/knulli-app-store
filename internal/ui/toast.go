package ui

import "time"

// ToastTTL is how long a notice stays on screen. It is long enough to read one
// sentence and short enough that a finished operation leaves no banner behind.
const ToastTTL = 3 * time.Second

// ToastLimit is how many notices the queue keeps. A notice normally expires
// before the next one arrives, since the manager serialises operations, so the
// real depth is one with a short burst on top; four keeps a burst intact without
// retaining a report the reader will never reach. The bound applies at Push
// rather than at Live, so the queue cannot grow while nothing is reading it.
const ToastLimit = 4

// toast is one transient notice: a line that reports what just happened and
// asks nothing of the user. Anything the user has to act on belongs in the
// status block instead, because that one stays until it is answered.
type toast struct {
	text string
	at   time.Time
}

// Toasts is the queue behind the notice bar. Notices are timestamped rather
// than cancelled, so the queue never needs a timer: a notice expires when
// someone reads the queue after its moment has passed.
type Toasts struct {
	items []toast
}

// Push adds a notice, stamped with the moment it arrived.
func (toasts *Toasts) Push(text string, at time.Time) {
	if text == "" {
		return
	}
	toasts.items = append(toasts.items, toast{text: text, at: at})
	if len(toasts.items) > ToastLimit {
		// Give up the oldest notice: it is also the one closest to its own
		// expiry, so it is the one the user has had longest to read. The slice is
		// shifted in place rather than reallocated, so a burst cannot grow it.
		copy(toasts.items, toasts.items[len(toasts.items)-ToastLimit:])
		toasts.items = toasts.items[:ToastLimit]
	}
}

// Live returns the notices that are still on screen at a moment, oldest first,
// and drops the expired ones. Reading is what expires them, which is why a
// notice disappears on its own without any other event arriving: the frame loop
// reads the queue every frame.
func (toasts *Toasts) Live(at time.Time) []string {
	live := toasts.items[:0]
	for _, notice := range toasts.items {
		if at.Sub(notice.at) < ToastTTL {
			live = append(live, notice)
		}
	}
	toasts.items = live
	texts := make([]string, 0, len(live))
	for _, notice := range live {
		texts = append(texts, notice.text)
	}
	return texts
}
