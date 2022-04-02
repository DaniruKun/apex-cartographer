/*
Copyright © 2022 Daniils Petrovs <thedanpetrov@gmail.com>

*/
package cmd

import (
	"fmt"
	"os"

	"image/color"

	"github.com/DaniruKun/apex-cartographer/imgproc"
	"github.com/spf13/cobra"
	"gocv.io/x/gocv"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "apex-cartographer",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running cartographer")
		filePath, _ := cmd.Flags().GetString("file")

		if filePath == "" {
			webcam, _ := gocv.VideoCaptureDevice(2)

			window := gocv.NewWindow("Live Recording")
			img := gocv.NewMat()
			for {
				webcam.Read(&img)
				window.IMShow(img)
				if window.WaitKey(1) >= 0 {
					break
				}
			}
		} else {
			video, err := gocv.VideoCaptureFile(filePath)
			if err != nil {
				fmt.Printf("Error opening video file: %s\n", filePath)
				return
			}

			defer video.Close()

			window := gocv.NewWindow("Video Recording")
			defer window.Close()

			img := gocv.NewMat()
			defer img.Close()

			grey := gocv.NewMat()
			defer grey.Close()

			const (
				MinimumArea = 10
				MinimumAr   = 3
			)

			var (
				width, height int
				ar            float32
			)
			// Frame read loop
			for {
				if ok := video.Read(&img); !ok {
					fmt.Printf("Device closed: %v\n", filePath)
					return
				}

				if img.Empty() {
					continue
				}

				croppedQuadrant := imgproc.CropTopLeftQuadrant(&img)
				defer croppedQuadrant.Close()

				gocv.CvtColor(croppedQuadrant, &grey, gocv.ColorRGBToGray)
				gocv.Threshold(grey, &grey, 150, 255, gocv.ThresholdBinary)
				contours := gocv.FindContours(grey, gocv.RetrievalExternal, gocv.ChainApproxSimple)

				for i := 0; i < contours.Size(); i++ {
					contour := contours.At(i)

					area := gocv.ContourArea(contour)
					if area < MinimumArea {
						continue
					}

					rect := gocv.BoundingRect(contour)

					width = rect.Dx()
					height = rect.Dy()
					ar = float32(width) / float32(height)

					if height > width || ar < 3 {
						continue
					}

					// Draw the found success rectangle
					gocv.Rectangle(&croppedQuadrant, rect, color.RGBA{0, 0, 255, 0}, 2)
				}

				contours.Close()

				window.IMShow(croppedQuadrant)
				if window.WaitKey(1) >= 0 {
					fmt.Println("Stopping processing...")
					break
				}
			}
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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.apex-cartographer.yaml)")
	rootCmd.Flags().StringP("file", "f", "", "Video file to run cartographer on")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
