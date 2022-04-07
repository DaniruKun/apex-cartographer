/*
Copyright © 2022 Daniils Petrovs <thedanpetrov@gmail.com>

*/
package cmd

import (
	"fmt"
	"image"
	"os"

	"image/color"

	"github.com/DaniruKun/apex-cartographer/imgproc"
	"github.com/spf13/cobra"
	"gocv.io/x/gocv"
)

var minimapRect image.Rectangle

const OlympusMapImgPath = "./resources/olympus-map-1-apex.png"
const TemplateMatchFrameInterval = 60 // the number of frames to skip minimap matching on, lower -> more precise

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "apex-cartographer",
	Short: "Apex Cartographer",
	Long:  `An app that analyses Apex Legends gameplay to infer the player's position on the map.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running cartographer")
		filePath, _ := cmd.Flags().GetString("file")

		video, err := gocv.VideoCaptureFile(filePath)
		if err != nil {
			fmt.Printf("Error opening video file: %s\n", filePath)
			return
		}

		defer video.Close()

		window := gocv.NewWindow("Found minimap matches")
		defer window.Close()

		img := gocv.NewMat()
		defer img.Close()

		// Stores the map image in memory where to match the minimap rect on
		mapImg := gocv.IMRead(OlympusMapImgPath, gocv.IMReadUnchanged)
		defer mapImg.Close()

		grey := gocv.NewMat()
		defer grey.Close()

		var frameCnt int

		// Frame read loop
		for {

			if ok := video.Read(&img); !ok {
				fmt.Printf("Device closed: %v\n", filePath)
				return
			}

			if frameCnt < TemplateMatchFrameInterval {
				frameCnt++
				continue
			}

			if img.Empty() {
				continue
			}

			croppedQuadrant := imgproc.CropTopLeftQuadrant(&img)
			defer croppedQuadrant.Close()

			if minimapRect.Min.X == 0 {
				// If the minimap has not been found yet, find its rect and set it
				fmt.Println("Minimap not found yet, detecting...")

				gocv.CvtColor(croppedQuadrant, &grey, gocv.ColorRGBToGray)
				gocv.Threshold(grey, &grey, 150, 255, gocv.ThresholdBinary)
				candidateMinimapRect, err := imgproc.FindMinimapRect(&grey)

				if err != nil {
					fmt.Println(err)
				} else {
					minimapRect = candidateMinimapRect
					fmt.Println("Minimap found at: ", minimapRect.Min.X, "x", minimapRect.Min.Y)
				}
			}

			if minimapRect.Min.X != 0 {
				// Perform matching only when the minimap rect has been previously detected

				// Create template mat
				template := croppedQuadrant.Region(minimapRect)
				defer template.Close()

				// TODO: Dynamically infer scaling factor and new size
				scaleFactor := 0.5

				newWidth := template.Cols() / 2
				newHeight := template.Cols() / 2

				gocv.Resize(template, &template, image.Point{X: newWidth, Y: newHeight}, scaleFactor, scaleFactor, gocv.InterpolationLanczos4)

				matchRes := gocv.NewMat()
				defer matchRes.Close()

				// TODO: Try out different matching methods
				method := gocv.TmCcoeff

				gocv.MatchTemplate(mapImg, template, &matchRes, method, gocv.NewMat())
				_, _, _, maxLoc := gocv.MinMaxLoc(matchRes)

				var foundLoc image.Point

				if method == gocv.TmCcoeff {
					foundLoc = maxLoc
				}

				matchRect := image.Rect(foundLoc.X, foundLoc.Y, foundLoc.X+template.Cols(), foundLoc.Y+template.Rows())

				fmt.Printf("Found minimap location: %d %d\n", foundLoc.X, foundLoc.Y)

				gocv.Rectangle(&mapImg, matchRect, color.RGBA{255, 255, 0, 0}, 2)
			}

			window.IMShow(mapImg)
			if window.WaitKey(1) >= 0 {
				fmt.Println("Stopping processing...")
				break
			}
			frameCnt = 0 // reset frame counter
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.apex-cartographer.yaml)")
	rootCmd.Flags().StringP("file", "f", "", "Video file to run cartographer on")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
