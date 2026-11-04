package sdlui

const canvasHeight = 480

// Short outputs use native pixels so text is never a downsampled desktop UI.
// Larger outputs retain the established scale without imposing a 4:3 minimum.
func canvasSize(width, height int) (int, int) {
	if width <= 0 || height <= 0 {
		return 640, canvasHeight
	}
	if height < canvasHeight {
		return min(width, 960), height
	}
	if width*canvasHeight/height < 320 {
		return 320, height * 320 / width
	}
	return min(960, width*canvasHeight/height), canvasHeight
}

// Kept as the default catalogue page size for semantic keyboard navigation.
const listRows = 5
