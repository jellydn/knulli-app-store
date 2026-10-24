package sdlui

const canvasHeight = 480

// Same logical-size rule as Random Game Picker 0.2.0. Both device targets use
// this layout; output aspect ratio, not a device name, determines its width.
func logicalWidth(width, height int) int {
	if height <= 0 {
		return 640
	}
	return max(640, width*canvasHeight/height)
}

// Kept as the default catalogue page size for semantic keyboard navigation.
const listRows = 5
