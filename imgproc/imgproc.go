package imgproc

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

const C = 0.71 // the ratio of the size of the minimap rectange on the map to the one in the first-person view

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

// Create a loop from which frames are read and decoded from a given video file at `filePath`
// The minimap is found, then located on the full size `mapImgPath`
func TrackMinimapLocationFromVideoFile(filePath, mapImgPath string, matchFrameInterval int) {
	var (
		minimapRect image.Rectangle
		foundLoc    image.Point
	)
	const frameBufferSize = 1024 // number of frames to keep in the video processing buffer

	frameBuffer := make(chan gocv.Mat, frameBufferSize)

	window := gocv.NewWindow("Found minimap matches")
	defer window.Close()

	// Stores the map image in memory where to match the minimap rect on
	mapImg := gocv.IMRead(mapImgPath, gocv.IMReadColor)
	defer mapImg.Close()

	grey := gocv.NewMat()
	defer grey.Close()

	template := gocv.NewMat()
	defer template.Close()

	croppedQuadrant := gocv.NewMat()
	defer croppedQuadrant.Close()

	matchRes := gocv.NewMat()
	defer matchRes.Close()

	go videoFrameProducer(filePath, matchFrameInterval, frameBuffer)

	// Frame read loop
	fmt.Println("Starting video frame consumer...")
	for {
		// Read frame from frame buffer
		vf, ok := <-frameBuffer

		if !ok {
			fmt.Println("Frame buffer is closed!")
			break
		}

		defer vf.Close()

		// Crop to quadrant containing the minimap
		croppedQuadrant = CropTopLeftQuadrant(&vf)
		defer croppedQuadrant.Close()

		// If the minimap has not been found yet, find its rect and set it
		if minimapRect.Min.X == 0 {
			fmt.Println("Minimap not found yet, detecting...")

			gocv.CvtColor(croppedQuadrant, &grey, gocv.ColorRGBToGray)
			gocv.Threshold(grey, &grey, 150, 255, gocv.ThresholdBinary)
			candidateMinimapRect, err := FindMinimapRect(&grey)

			if err != nil {
				fmt.Println(err)
				continue
			} else {
				minimapRect = candidateMinimapRect
				fmt.Println("Minimap found at: ", minimapRect.Min.X, "x", minimapRect.Min.Y)
			}
		}

		// Create template mat
		template = croppedQuadrant.Region(minimapRect)

		doResize(&template, C)

		// TODO: Try out different matching methods
		method := gocv.TmCcoeff

		gocv.MatchTemplate(mapImg, template, &matchRes, method, gocv.NewMat())
		_, _, _, maxLoc := gocv.MinMaxLoc(matchRes)

		if method == gocv.TmCcoeff {
			foundLoc = maxLoc
		}

		matchRect := image.Rect(foundLoc.X, foundLoc.Y, foundLoc.X+template.Cols(), foundLoc.Y+template.Rows())

		matchRectCenter := image.Point{X: matchRect.Min.X + (matchRect.Dx() / 2), Y: matchRect.Min.Y + (matchRect.Dy() / 2)}

		fmt.Printf("Found minimap location: %d %d\n", foundLoc.X, foundLoc.Y)

		// gocv.Rectangle(&mapImg, matchRect, color.RGBA{255, 255, 0, 0}, 2)
		gocv.Circle(&mapImg, matchRectCenter, 3, color.RGBA{0, 255, 0, 255}, 3)

		window.IMShow(mapImg)
		if window.WaitKey(1) >= 0 {
			fmt.Println("Stopping processing...")
			close(frameBuffer)
			break
		}

		// var b bytes.Buffer
		// gocv.MatProfile.WriteTo(&b, 1)
		// fmt.Print(b.String())
	}
}

func doResize(src *gocv.Mat, scaleFactor float64) {
	newDims := image.Point{
		X: int(float64(src.Cols()) * scaleFactor),
		Y: int(float64(src.Rows()) * scaleFactor),
	}

	gocv.Resize(*src, src, newDims, scaleFactor, scaleFactor, gocv.InterpolationLanczos4)
}

func videoFrameProducer(filePath string, matchFrameInterval int, buffer chan gocv.Mat) {
	fmt.Println("Starting frame producer...")
	var frameCnt int
	frame := gocv.NewMat()
	defer frame.Close()

	video, err := gocv.VideoCaptureFile(filePath)

	if err != nil {
		fmt.Printf("Error opening video file: %s\n", filePath)
		return
	}

	defer video.Close()

	for {
		if ok := video.Read(&frame); !ok {
			fmt.Printf("Device closed: %v\n", filePath)
			close(buffer)
			break
		}

		if frameCnt < matchFrameInterval {
			frameCnt++
			continue
		}

		if frame.Empty() {
			continue
		}

		buffer <- frame
		frameCnt = 0 // reset frame counter
	}
}
