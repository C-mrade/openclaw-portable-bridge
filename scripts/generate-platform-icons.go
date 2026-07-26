//go:build ignore

// Generate small, deterministic platform badges used by the portable package.
// They are intentionally original and do not claim vendor endorsement.
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

type point struct{ x, y int }

func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	draw.Draw(img, image.Rect(x0, y0, x1, y1), &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func badge(base color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)
	fillCircle(img, 128, 128, 116, base)
	fillCircle(img, 128, 128, 101, color.RGBA{20, 26, 35, 255})
	return img
}

func windows() *image.RGBA {
	img := badge(color.RGBA{0, 120, 215, 255})
	blue := color.RGBA{0, 165, 255, 255}
	fillRect(img, 66, 70, 119, 121, blue)
	fillRect(img, 128, 70, 190, 121, blue)
	fillRect(img, 66, 130, 119, 186, blue)
	fillRect(img, 128, 130, 190, 186, blue)
	return img
}

func linux() *image.RGBA {
	img := badge(color.RGBA{255, 193, 7, 255})
	white := color.RGBA{245, 247, 250, 255}
	black := color.RGBA{15, 18, 23, 255}
	gold := color.RGBA{255, 183, 0, 255}
	fillCircle(img, 128, 128, 61, black)
	fillCircle(img, 128, 142, 43, white)
	fillCircle(img, 108, 105, 13, white)
	fillCircle(img, 148, 105, 13, white)
	fillCircle(img, 110, 106, 5, black)
	fillCircle(img, 146, 106, 5, black)
	fillRect(img, 113, 119, 143, 129, gold)
	fillCircle(img, 102, 187, 20, gold)
	fillCircle(img, 154, 187, 20, gold)
	return img
}

func macos() *image.RGBA {
	img := badge(color.RGBA{165, 172, 184, 255})
	silver := color.RGBA{226, 230, 236, 255}
	fillCircle(img, 112, 137, 45, silver)
	fillCircle(img, 148, 137, 45, silver)
	fillCircle(img, 130, 162, 48, silver)
	fillCircle(img, 163, 116, 17, color.RGBA{20, 26, 35, 255})
	// A simple leaf, deliberately geometric rather than a vendor logo.
	fillRect(img, 132, 68, 145, 94, silver)
	fillCircle(img, 146, 70, 13, silver)
	return img
}

func write(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	out := "assets/platform-icons"
	if len(os.Args) == 2 {
		out = os.Args[1]
	}
	for name, img := range map[string]image.Image{
		"windows.png": windows(),
		"linux.png":   linux(),
		"macos.png":   macos(),
	} {
		if err := write(filepath.Join(out, name), img); err != nil {
			panic(err)
		}
	}
}
