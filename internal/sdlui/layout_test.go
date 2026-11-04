package sdlui

import "testing"

func TestResponsiveCanvasUsesNativeSmallPixelsAndAvailableAspect(t *testing.T) {
	for _, test := range []struct{ width, height, wantW, wantH int }{
		{640, 480, 640, 480}, {1280, 720, 853, 480}, {1920, 1080, 853, 480},
		{320, 240, 320, 240}, {480, 320, 480, 320}, {640, 479, 640, 479},
		{720, 720, 480, 480}, {480, 640, 360, 480}, {320, 640, 320, 640},
		{2560, 1080, 960, 480}, {0, 0, 640, 480},
	} {
		if w, h := canvasSize(test.width, test.height); w != test.wantW || h != test.wantH {
			t.Fatalf("%dx%d: %dx%d, want %dx%d", test.width, test.height, w, h, test.wantW, test.wantH)
		}
	}
}
