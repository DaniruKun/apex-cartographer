package imgproc

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"

	"gocv.io/x/gocv"
)

const OlympusMapImgPath = "./resources/maps/olympus.png"

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

func TrackMinimapLocationFromVideoFile(filePath string, matchFrameInterval int) {
	const frameBufferSize = 1024 // number of frames to keep in the video processing buffer

	frameBuffer := make(chan gocv.Mat, frameBufferSize)
	results := make(chan image.Point, 128)

	go frameProducer(filePath, matchFrameInterval, frameBuffer)

	go frameProcessor(frameBuffer, results)

	// Needs to be in the main thread because of the UI
	resultsPresenter(results)
}

func doResize(src *gocv.Mat, scaleFactor float64) {
	newDims := image.Point{
		X: int(float64(src.Cols()) * scaleFactor),
		Y: int(float64(src.Rows()) * scaleFactor),
	}

	gocv.Resize(*src, src, newDims, scaleFactor, scaleFactor, gocv.InterpolationLanczos4)
}

// Produces Mats read from a video file at `filePath` into a `buffer` channel
func frameProducer(filePath string, matchFrameInterval int, buffer chan<- gocv.Mat) {
	fmt.Println("Starting frame producer...")
	var frameCnt int
	frame := gocv.NewMat()
	defer frame.Close()

	video, err := gocv.VideoCaptureFile(filePath)

	if err != nil {
		fmt.Printf("Error opening video file: %s\n", filePath)
		close(buffer)
		return
	}

	defer video.Close()

	for {
		if ok := video.Read(&frame); !ok {
			fmt.Printf("Device closed: %v\n", filePath)
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
	close(buffer)
	fmt.Println("Frame producer stopped!")
}

// Processes Mats from the video frame `buffer`, performs detection, and puts the detection results into `results` channel
func frameProcessor(buffer <-chan gocv.Mat, results chan<- image.Point) {
	fmt.Println("Starting video frame processor...")

	// Prepare all of the Mats we will need
	var (
		err                  error
		candidateMinimapRect image.Rectangle
		minimapRect          image.Rectangle
		matchRect            image.Rectangle
		matchRectCenter      image.Point
		foundLoc             image.Point
	)

	const matchMethod = gocv.TmCcoeff

	// Stores the map image in memory where to match the minimap rect on
	mapImg := gocv.IMRead(OlympusMapImgPath, gocv.IMReadColor)
	defer mapImg.Close()

	croppedQuadrant := gocv.NewMat()
	defer croppedQuadrant.Close()

	grey := gocv.NewMat()
	defer grey.Close()

	template := gocv.NewMat()
	defer template.Close()

	matchRes := gocv.NewMat()
	defer matchRes.Close()

	for {
		frame, more := <-buffer

		if !more {
			fmt.Println("No more frames to process!")
			break
		}
		defer frame.Close()

		// Crop to quadrant containing the minimap
		croppedQuadrant = CropTopLeftQuadrant(&frame)

		// If the minimap has not been found yet, find its rect and set it
		if minimapRect.Min.X == 0 {
			fmt.Println("Minimap not found yet, detecting...")

			gocv.CvtColor(croppedQuadrant, &grey, gocv.ColorRGBToGray)
			gocv.Threshold(grey, &grey, 150, 255, gocv.ThresholdBinary)
			candidateMinimapRect, err = FindMinimapRect(&grey)

			if err != nil {
				fmt.Println(err)
				// Skip the frame and try with the next one
				continue
			} else {
				minimapRect = candidateMinimapRect
				fmt.Println("Minimap found at: ", minimapRect.Min.X, "x", minimapRect.Min.Y)
			}
		}

		// Create template and resize it to correct dimensions for matching
		template = croppedQuadrant.Region(minimapRect)
		doResize(&template, C)

		gocv.MatchTemplate(mapImg, template, &matchRes, matchMethod, gocv.NewMat())
		_, _, _, maxLoc := gocv.MinMaxLoc(matchRes)

		if matchMethod == gocv.TmCcoeff {
			foundLoc = maxLoc
		}

		matchRect = image.Rect(foundLoc.X, foundLoc.Y, foundLoc.X+template.Cols(), foundLoc.Y+template.Rows())
		matchRectCenter = image.Point{X: matchRect.Min.X + (matchRect.Dx() / 2), Y: matchRect.Min.Y + (matchRect.Dy() / 2)}

		results <- matchRectCenter
	}
	close(results)
	fmt.Println("Frame processor stopped!")
}

// Loops over messages in the `results` channel and presents the results in an OpenCV GUI window, and outputs to console
func resultsPresenter(results <-chan image.Point) {
	fmt.Println("Starting results presenter...")

	window := gocv.NewWindow("Map movement")
	defer window.Close()

	mapImg := gocv.IMRead(OlympusMapImgPath, gocv.IMReadColor)
	defer mapImg.Close()

	for {
		result, more := <-results

		if !more {
			fmt.Println("No more results")
			break
		}

		gocv.Circle(&mapImg, result, 3, color.RGBA{0, 255, 0, 255}, 3)
		window.IMShow(mapImg)

		fmt.Printf("Found point at: (%d, %d)\n", result.X, result.Y)

		if window.WaitKey(1) >= 0 {
			fmt.Println("User requested to stop processing...")
			os.Exit(0)
			break
		}
	}
	fmt.Println("Results presenter stopped!")
}
