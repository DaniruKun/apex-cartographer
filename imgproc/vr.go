package imgproc

import (
	"errors"
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

// Finds the minimap in a src binary image and returns a bounding Rectangle of it
func FindMinimapRect(src *gocv.Mat) (image.Rectangle, error) {
	const (
		minimumArea  = 10
		minimumAr    = 3
		maxMinimapAr = 2
	)
	candidateRects := []image.Rectangle{}

	contours := gocv.FindContours(*src, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	defer contours.Close()

	for i := 0; i < contours.Size(); i++ {
		contour := contours.At(i)

		area := gocv.ContourArea(contour)
		rect := gocv.BoundingRect(contour)

		width := rect.Dx()
		height := rect.Dy()
		ar := float32(width) / float32(height)

		if area > minimumArea && ar > 3 {
			candidateRects = append(candidateRects, rect)
		}
	}

	minimapMin := image.Point{X: 9999, Y: 9999}
	minimapMax := image.Point{X: 0, Y: 0}

	// Find extremes of the collection of rectangles part of the minimap
	for _, candidateRect := range candidateRects {
		if candidateRect.Min.X < minimapMin.X {
			minimapMin.X = candidateRect.Min.X
		}
		if candidateRect.Min.Y < minimapMin.Y {
			minimapMin.Y = candidateRect.Min.Y
		}
		if candidateRect.Max.X > minimapMax.X {
			minimapMax.X = candidateRect.Max.X
		}
		if candidateRect.Max.Y > minimapMax.Y {
			minimapMax.Y = candidateRect.Max.Y
		}
	}

	minimapRect := image.Rect(minimapMin.X, minimapMin.Y, minimapMax.X, minimapMax.Y)
	minimapRectAr := float32(minimapRect.Dx()) / float32(minimapRect.Dy())

	if minimapRectAr < maxMinimapAr {
		return minimapRect, nil
	} else {
		return image.Rectangle{}, errors.New("could not find minimap rectangle")
	}
}
