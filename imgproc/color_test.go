package imgproc

import (
	"fmt"
	"image/color"
	"testing"
)

func TestRotateHue(t *testing.T) {
	hsv := HSV{H: 0, S: 1, V: 1}

	hsv.RotateHue(1, "ccw")

	if hsv.H != 359 {
		t.Error("expected hue of 359, got: ", hsv.H)
	}

	hsv.RotateHue(2, "cw")

	if hsv.H != 1 {
		t.Error("expected hue of 1, got: ", hsv.H)
	}
}

func TestRGBA(t *testing.T) {
	var tests = []struct {
		hsv  HSV
		rgba color.RGBA
	}{
		{HSV{0, 0, 0}, color.RGBA{0, 0, 0, 255}},
		{HSV{0, 0, 1}, color.RGBA{255, 255, 255, 255}},
		{HSV{0, 1, 1}, color.RGBA{255, 0, 0, 255}},
		{HSV{120, 1, 1}, color.RGBA{0, 255, 0, 255}},
		{HSV{240, 1, 1}, color.RGBA{0, 0, 255, 255}},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("HSV %v -> RGBA %v", tt.hsv, tt.rgba)
		t.Run(testname, func(t *testing.T) {
			res := tt.hsv.RGBA()
			if res != tt.rgba {
				t.Errorf("got %+v, want %+v", res, tt.rgba)
			}
		})
	}
}
