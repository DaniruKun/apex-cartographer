package imgproc

import (
	"image"

	"gocv.io/x/gocv"
)

// Creates and returns a new Mat from the top left quadrant of the src Mat
func CropTopLeftQuadrant(mat *gocv.Mat) gocv.Mat {
	srcWidth := mat.Cols()
	srcHeight := mat.Rows()

	rect := image.Rect(0, 0, srcWidth/2, srcHeight/2)
	cropped := mat.Region(rect)
	return cropped
}
