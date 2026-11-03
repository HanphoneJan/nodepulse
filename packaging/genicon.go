//go:build ignore

// Command genicon draws NodePulse's app icon without external dependencies.
// Run from the repository root: go run packaging/genicon.go
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const (
	bg      = 0x0E1220
	accent  = 0x7C9CFF
	success = 0x5DE29A
)

type point struct{ x, y float64 }

func main() {
	if err := os.MkdirAll("packaging", 0o755); err != nil {
		panic(err)
	}
	icon := render(1024, 4)
	writePNG("packaging/nodepulse-1024.png", icon)
	writePNG("gui/tray-icon.png", render(64, 8))
	writeICO("packaging/nodepulse.ico", []int{16, 24, 32, 48, 64, 128, 256})
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func writeICO(path string, sizes []int) {
	var images [][]byte
	for _, size := range sizes {
		var buf bytes.Buffer
		if err := png.Encode(&buf, render(size, 8)); err != nil {
			panic(err)
		}
		images = append(images, buf.Bytes())
	}
	var out bytes.Buffer
	_ = binary.Write(&out, binary.LittleEndian, uint16(0))
	_ = binary.Write(&out, binary.LittleEndian, uint16(1))
	_ = binary.Write(&out, binary.LittleEndian, uint16(len(sizes)))
	offset := 6 + 16*len(sizes)
	for i, size := range sizes {
		b := byte(size)
		if size >= 256 {
			b = 0
		}
		out.WriteByte(b)
		out.WriteByte(b)
		out.WriteByte(0)
		out.WriteByte(0)
		_ = binary.Write(&out, binary.LittleEndian, uint16(1))
		_ = binary.Write(&out, binary.LittleEndian, uint16(32))
		_ = binary.Write(&out, binary.LittleEndian, uint32(len(images[i])))
		_ = binary.Write(&out, binary.LittleEndian, uint32(offset))
		offset += len(images[i])
	}
	for _, data := range images {
		out.Write(data)
	}
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		panic(err)
	}
}

func render(size, supersample int) *image.RGBA {
	hi := size * supersample
	canvas := image.NewRGBA(image.Rect(0, 0, hi, hi))
	scale := float64(hi) / 256

	bgColor := rgb(bg)
	fillRoundedRect(canvas, 0, 0, float64(hi), float64(hi), 56*scale, bgColor)

	stroke := 18 * scale
	line := []point{{42, 148}, {84, 148}, {108, 92}, {138, 184}, {168, 124}, {190, 148}, {214, 148}}
	for i := 1; i < len(line); i++ {
		drawSegment(canvas, scalePoint(line[i-1], scale), scalePoint(line[i], scale), stroke/2, rgb(accent))
	}
	drawCircle(canvas, scalePoint(point{168, 124}, scale), 8*scale, rgb(success))

	// Downsample with a box filter. Supersampling gives clean edges for the
	// rounded card and thick pulse stroke.
	out := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b, a uint64
			for dy := 0; dy < supersample; dy++ {
				for dx := 0; dx < supersample; dx++ {
					cr, cg, cb, ca := canvas.At(x*supersample+dx, y*supersample+dy).RGBA()
					r += uint64(cr)
					g += uint64(cg)
					b += uint64(cb)
					a += uint64(ca)
				}
			}
			n := uint64(supersample * supersample)
			out.SetRGBA(x, y, color.RGBA{uint8(r / n >> 8), uint8(g / n >> 8), uint8(b / n >> 8), uint8(a / n >> 8)})
		}
	}
	return out
}

func scalePoint(p point, scale float64) point {
	return point{p.x * scale, p.y * scale}
}

func rgb(hex uint32) color.RGBA {
	return color.RGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255}
}

func fillRoundedRect(img *image.RGBA, x, y, w, h, r float64, c color.RGBA) {
	for py := int(y); py < int(y+h); py++ {
		for px := int(x); px < int(x+w); px++ {
			if insideRounded(px, py, x, y, w, h, r) {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

func insideRounded(px, py int, x, y, w, h, r float64) bool {
	fx, fy := float64(px)+0.5, float64(py)+0.5
	if fx < x+r && fy < y+r {
		return math.Hypot(fx-(x+r), fy-(y+r)) <= r
	}
	if fx > x+w-r && fy < y+r {
		return math.Hypot(fx-(x+w-r), fy-(y+r)) <= r
	}
	if fx < x+r && fy > y+h-r {
		return math.Hypot(fx-(x+r), fy-(y+h-r)) <= r
	}
	if fx > x+w-r && fy > y+h-r {
		return math.Hypot(fx-(x+w-r), fy-(y+h-r)) <= r
	}
	return fx >= x && fx <= x+w && fy >= y && fy <= y+h
}

func drawSegment(img *image.RGBA, a, b point, radius float64, c color.RGBA) {
	minX := int(math.Min(a.x, b.x) - radius - 1)
	maxX := int(math.Max(a.x, b.x) + radius + 1)
	minY := int(math.Min(a.y, b.y) - radius - 1)
	maxY := int(math.Max(a.y, b.y) + radius + 1)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if distanceToSegment(point{float64(x) + .5, float64(y) + .5}, a, b) <= radius {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func drawCircle(img *image.RGBA, center point, radius float64, c color.RGBA) {
	minX := int(center.x - radius - 1)
	maxX := int(center.x + radius + 1)
	minY := int(center.y - radius - 1)
	maxY := int(center.y + radius + 1)
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx, dy := float64(x)+.5-center.x, float64(y)+.5-center.y
			if dx*dx+dy*dy <= radius*radius {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func distanceToSegment(p, a, b point) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	if dx == 0 && dy == 0 {
		return math.Hypot(p.x-a.x, p.y-a.y)
	}
	t := ((p.x-a.x)*dx + (p.y-a.y)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(p.x-(a.x+t*dx), p.y-(a.y+t*dy))
}
