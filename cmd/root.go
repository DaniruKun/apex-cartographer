/*
Copyright © 2022 Daniils Petrovs <thedanpetrov@gmail.com>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/DaniruKun/apex-cartographer/imgproc"
	"github.com/spf13/cobra"
)

const OlympusMapImgPath = "./resources/maps/olympus.png"
const TemplateMatchFrameInterval = 60 // the number of frames to skip minimap matching on, lower -> more precise

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "apex-cartographer",
	Short: "Apex Cartographer",
	Long:  `An app that analyses Apex Legends gameplay to infer the player's position on the map.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running cartographer")
		filePath, _ := cmd.Flags().GetString("file")

		imgproc.TrackMinimapLocationFromVideoFile(filePath, OlympusMapImgPath, TemplateMatchFrameInterval)
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
