package imgproc

import (
	"image/color"
	"log"
	"math"
)

type HSV struct {
	H uint32  // 0 <= H < 360
	S float64 // 0 <= S <= 1
	V float64 // 0 <= V <= 1
}

const (
	CW  = "cw"
	CCW = "ccw"
)

// Rotates the hue `H` by a number of `degrees` in the given `direction`. Direction is either `cw` or `ccw`
func (color *HSV) RotateHue(degrees uint32, direction string) {
	switch direction {
	case CW:
		if color.H == 359 {
			color.H = 0
			color.H += (degrees - 1)
		} else {
			color.H += degrees
		}

	case CCW:
		if color.H == 0 {
			color.H = 359
			color.H -= (degrees - 1)
		} else {
			color.H -= degrees
		}
	default:
		log.Fatal("unknown direction: ", direction)
	}
}

// Converts an HSV color to RGBA, where `A` is implicitly set to 255 (solid)
func (col HSV) RGBA() color.RGBA {
	var rp, gp, bp float64 // R' G' B'

	h := col.H
	c := col.V * col.S
	x := c * (1 - math.Abs(math.Mod(float64(h/60), 2)-1))
	m := col.V - c

	getPrimes := func() (rp, gp, bp float64) {
		switch {
		case h < 60:
			return c, x, 0
		case 60 <= h && h < 120:
			return x, c, 0
		case 120 <= h && h < 180:
			return 0, c, x
		case 180 <= h && h < 240:
			return 0, x, c
		case 240 <= h && h < 360:
			return x, 0, c
		case 300 <= h && h < 360:
			return c, 0, x
		default:
			return 0, 0, 0
		}
	}

	rp, gp, bp = getPrimes()

	r := uint8((rp + m) * 255)
	g := uint8((gp + m) * 255)
	b := uint8((bp + m) * 255)

	return color.RGBA{r, g, b, 255}
}
