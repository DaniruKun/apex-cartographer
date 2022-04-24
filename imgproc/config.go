package imgproc

type Config struct {
	FrameInterval int    // Number of frames to skip in between detection attempts
	MapName       string // Apex Legends map, currently only `olympus`
	Debug         bool   // Toggles debug mode
	ShowGUI       bool   // Show GUI with live visuals or not
	SaveImg       bool   // Save the final result image
}
