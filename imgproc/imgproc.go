package imgproc

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"

	"github.com/DaniruKun/apex-cartographer/utils"

	"gocv.io/x/gocv"
)

const C = 0.71 // the ratio of the size of the minimap rectange on the map to the one in the first-person view

var ColorGreen = color.RGBA{0, 255, 0, 255}

var config Config // global config for the processing session

// Run recognition and result recording from a local video file at `filePath`
// The `userConfig` is later used in all related functions
func RunTrackingFromFile(filePath string, userConfig Config) {
	const frameBufferSize = 1024 // number of frames to keep in the video processing buffer
	config = userConfig

	frameBuffer := make(chan gocv.Mat, frameBufferSize)
	results := make(chan image.Point, 128)

	go frameProducer(filePath, config.FrameInterval, frameBuffer)

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
func frameProducer(filePath string, frameInterval int, buffer chan<- gocv.Mat) {
	fmt.Println("Starting frame producer...")

	frame := gocv.NewMat()
	defer frame.Close()

	video, err := gocv.VideoCaptureFile(filePath)

	if err != nil {
		fmt.Printf("Error opening video file: %s\n", filePath)
		close(buffer)
		return
	}

	defer video.Close()

	w := video.Get(3)
	h := video.Get(4)
	fps := video.Get(5)
	frameCntTotal := video.Get(7)
	currFrameNo := 0

	fmt.Printf("Reading video file %s\nFPS: %f\nDimensions: %f x %f\n", filePath, fps, w, h)

	for {
		video.Grab(frameInterval)
		if ok := video.Read(&frame); !ok {
			fmt.Printf("Device closed: %v\n", filePath)
			break
		}

		if frame.Empty() {
			continue
		}

		currFrameNo += frameInterval

		buffer <- frame
		fmt.Printf("Read frame %d of %d | ", currFrameNo, int(frameCntTotal))
	}
	close(buffer)
	fmt.Println("Frame producer stopped!")
	os.Exit(0)
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

	mapFilePath, err := utils.GetMapPath(config.MapName)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Stores the map image in memory where to match the minimap rect on
	mapImg := gocv.IMRead(mapFilePath, gocv.IMReadColor)
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
			frame.Close()
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

			if config.Debug {
				gocv.Rectangle(&croppedQuadrant, candidateMinimapRect, ColorGreen, 4)
				gocv.IMWrite(filepath.Join("debug", "minimap-detection-frame-debug.png"), frame)
				gocv.IMWrite(filepath.Join("debug", "minimap-detection-grey-debug.png"), grey)
				gocv.IMWrite(filepath.Join("debug", "minimap-detection-candidate-rect-debug.png"), croppedQuadrant)
			}

			if err != nil {
				fmt.Println(err)
				// Skip the frame and try with the next one
				continue
			} else {
				minimapRect = candidateMinimapRect
				fmt.Printf("Minimap found at: (%d, %d)\n", minimapRect.Min.X, minimapRect.Min.Y)
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

// Loops over messages in the `results` channel and presents the results
func resultsPresenter(results <-chan image.Point) {
	fmt.Println("Starting results presenter...")

	mapFilePath, err := utils.GetMapPath(config.MapName)
	if err != nil {
		log.Fatal(err)
		return
	}

	mapImg := gocv.IMRead(mapFilePath, gocv.IMReadColor)
	defer mapImg.Close()

	markerColor := HSV{120, 1, 1}

	window := gocv.NewWindow("Map movement")
	defer window.Close()

	for {
		result, more := <-results

		if !more {
			fmt.Println("No more results")
			break
		}

		if config.Debug {
			fmt.Printf("Found point at: (%d, %d)\n", result.X, result.Y)
		}

		markerColor.RotateHue(5, "cw")

		gocv.Circle(&mapImg, result, 2, markerColor.RGBA(), 2)

		if config.SaveImg {
			resultFileName := fmt.Sprintf("%s-route.png", config.MapName)
			gocv.IMWrite(resultFileName, mapImg)
		}

		if config.ShowGUI {
			window.IMShow(mapImg)

			if window.WaitKey(1) >= 0 {
				fmt.Println("User requested to stop processing...")
				break
			}
		}
	}

	fmt.Println("Results presenter stopped!")
	os.Exit(0)
}
