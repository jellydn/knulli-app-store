package sdlui

import "testing"

func TestResponsiveLogicalWidth(t *testing.T) {
	for _, test := range []struct{ width, height, want int }{{640, 480, 640}, {1280, 720, 853}, {1920, 1080, 853}, {320, 240, 640}, {0, 0, 640}} {
		if got := logicalWidth(test.width, test.height); got != test.want {
			t.Fatalf("%dx%d: %d, want %d", test.width, test.height, got, test.want)
		}
	}
}
