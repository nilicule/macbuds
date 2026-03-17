//go:build ignore

// Run with: go run gen_icons.go
// Generates assets/icon_none.png, assets/icon_connected.png, assets/icon_disconnected.png

package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const size = 44

func setPixel(img *image.RGBA, x, y int, c color.RGBA) {
	if x >= 0 && x < size && y >= 0 && y < size {
		img.SetRGBA(x, y, c)
	}
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			if dx*dx+dy*dy <= float64(r*r)+0.5 {
				setPixel(img, x, y, c)
			}
		}
	}
}

// drawRingArc draws a thick arc between startDeg and endDeg (in standard math coords,
// y-axis pointing down in image space: 270° = top of image).
func drawRingArc(img *image.RGBA, cx, cy, r, thickness int, startDeg, endDeg float64, c color.RGBA) {
	innerR := r - thickness/2
	outerR := r + thickness/2
	steps := int((endDeg-startDeg)*float64(r)*math.Pi/180) * 4
	if steps < 2000 {
		steps = 2000
	}
	for i := 0; i <= steps; i++ {
		deg := startDeg + float64(i)/float64(steps)*(endDeg-startDeg)
		rad := deg * math.Pi / 180
		cosA := math.Cos(rad)
		sinA := math.Sin(rad)
		for r2 := innerR; r2 <= outerR; r2++ {
			x := cx + int(math.Round(cosA*float64(r2)))
			y := cy + int(math.Round(sinA*float64(r2)))
			setPixel(img, x, y, c)
		}
	}
}

func drawHeadphones(img *image.RGBA, c color.RGBA) {
	// Headband: arc from 180° to 360° (through 270° = top of image in y-down coords)
	// Center at (22, 24), radius 13 → endpoints at (9, 24) and (35, 24)
	drawRingArc(img, 22, 24, 13, 5, 180, 360, c)

	// Earcups: solid circles at arc endpoints, slightly below
	fillCircle(img, 9, 30, 8, c)
	fillCircle(img, 35, 30, 8, c)
}

func saveIcon(filename string, img *image.RGBA) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func main() {
	if err := os.MkdirAll("assets", 0755); err != nil {
		panic(err)
	}

	// No device selected – neutral dark gray
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	drawHeadphones(img, color.RGBA{80, 80, 80, 200})
	saveIcon("assets/icon_none.png", img)

	// Connected – macOS system green
	img = image.NewRGBA(image.Rect(0, 0, size, size))
	drawHeadphones(img, color.RGBA{52, 199, 89, 255})
	saveIcon("assets/icon_connected.png", img)

	// Disconnected – macOS system red
	img = image.NewRGBA(image.Rect(0, 0, size, size))
	drawHeadphones(img, color.RGBA{255, 59, 48, 255})
	saveIcon("assets/icon_disconnected.png", img)

	println("Icons generated: assets/icon_none.png, assets/icon_connected.png, assets/icon_disconnected.png")
}
