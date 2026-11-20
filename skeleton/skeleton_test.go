package skeleton

import (
	"image"
	"image/color"
	"testing"

	"github.com/ernyoke/imger/utils"
)

// -------------------------------------------------------------------------------------------------------
// Test helpers
// -------------------------------------------------------------------------------------------------------

// newBinaryImage returns a w x h black image with the foreground pixels set to the given value.
func newBinaryImage(w, h int, foreground uint8, pixels []image.Point) *image.Gray {
	var background uint8 = utils.MinUint8
	if foreground == utils.MinUint8 {
		background = utils.MaxUint8
	}
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetGray(x, y, color.Gray{Y: background})
		}
	}
	for _, p := range pixels {
		img.SetGray(p.X, p.Y, color.Gray{Y: foreground})
	}
	return img
}

// rectPixels returns the pixels of the filled rectangle [x0, x1] x [y0, y1].
func rectPixels(x0, y0, x1, y1 int) []image.Point {
	var pixels []image.Point
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			pixels = append(pixels, image.Point{X: x, Y: y})
		}
	}
	return pixels
}

// ringPixels returns the pixels of a filled ring centered at (cx, cy).
func ringPixels(cx, cy int, innerR, outerR float64, w, h int) []image.Point {
	var pixels []image.Point
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := float64(x-cx), float64(y-cy)
			d2 := dx*dx + dy*dy
			if d2 >= innerR*innerR && d2 <= outerR*outerR {
				pixels = append(pixels, image.Point{X: x, Y: y})
			}
		}
	}
	return pixels
}

// foregroundPixels collects the foreground pixel positions of a binary image.
func foregroundPixels(img *image.Gray, foreground uint8) []image.Point {
	var pixels []image.Point
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.GrayAt(x, y).Y == foreground {
				pixels = append(pixels, image.Point{X: x, Y: y})
			}
		}
	}
	return pixels
}

// neighbourCount counts the 8-connected foreground neighbours of (x, y).
func neighbourCount(img *image.Gray, foreground uint8, x, y int) int {
	count := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if !image.Pt(nx, ny).In(img.Bounds()) {
				continue
			}
			if img.GrayAt(nx, ny).Y == foreground {
				count++
			}
		}
	}
	return count
}

// endpoints returns the foreground pixels with exactly one foreground neighbour.
func endpoints(img *image.Gray, foreground uint8) []image.Point {
	var result []image.Point
	for _, p := range foregroundPixels(img, foreground) {
		if neighbourCount(img, foreground, p.X, p.Y) == 1 {
			result = append(result, p)
		}
	}
	return result
}

// junctions returns the foreground pixels with three or more foreground neighbours.
func junctions(img *image.Gray, foreground uint8) []image.Point {
	var result []image.Point
	for _, p := range foregroundPixels(img, foreground) {
		if neighbourCount(img, foreground, p.X, p.Y) >= 3 {
			result = append(result, p)
		}
	}
	return result
}

// countComponents counts the 8-connected foreground components using flood fill.
func countComponents(img *image.Gray, foreground uint8) int {
	bounds := img.Bounds()
	size := bounds.Size()
	w, h := size.X, size.Y
	offset := bounds.Min
	visited := make([]bool, w*h)
	at := func(x, y int) bool {
		return img.GrayAt(x+offset.X, y+offset.Y).Y == foreground
	}
	components := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !at(x, y) || visited[y*w+x] {
				continue
			}
			components++
			queue := []image.Point{{X: x, Y: y}}
			visited[y*w+x] = true
			for len(queue) > 0 {
				p := queue[0]
				queue = queue[1:]
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := p.X+dx, p.Y+dy
						if nx < 0 || ny < 0 || nx >= w || ny >= h {
							continue
						}
						if at(nx, ny) && !visited[ny*w+nx] {
							visited[ny*w+nx] = true
							queue = append(queue, image.Point{X: nx, Y: ny})
						}
					}
				}
			}
		}
	}
	return components
}

// nearPoint reports whether some point of points lies within radius of (x, y).
func nearPoint(points []image.Point, x, y, radius int) bool {
	for _, p := range points {
		dx, dy := p.X-x, p.Y-y
		if dx*dx+dy*dy <= radius*radius {
			return true
		}
	}
	return false
}

func assertSkeletonEqual(t *testing.T, expected, actual *image.Gray) {
	t.Helper()
	if !expected.Bounds().Eq(actual.Bounds()) {
		t.Fatalf("bounds mismatch: expected %v, got %v", expected.Bounds(), actual.Bounds())
	}
	bounds := expected.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if expected.GrayAt(x, y).Y != actual.GrayAt(x, y).Y {
				t.Fatalf("pixel mismatch at (%d, %d): expected %d, got %d",
					x, y, expected.GrayAt(x, y).Y, actual.GrayAt(x, y).Y)
			}
		}
	}
}

// -------------------------------------------------------------------------------------------------------
// Fixed shape tests
// -------------------------------------------------------------------------------------------------------

// A 3 pixel thick horizontal bar must collapse into a 1 pixel wide horizontal line
// with exactly 2 endpoints and a single connected component.
func Test_Skeletonize_StraightLine(t *testing.T) {
	foreground := utils.MaxUint8
	img := newBinaryImage(21, 7, foreground, rectPixels(2, 2, 18, 4))

	skeleton := Skeletonize(img, foreground)

	if !skeleton.Bounds().Eq(img.Bounds()) {
		t.Fatalf("expected bounds %v, got %v", img.Bounds(), skeleton.Bounds())
	}
	if components := countComponents(skeleton, foreground); components != 1 {
		t.Fatalf("expected 1 connected component, got %d", components)
	}
	eps := endpoints(skeleton, foreground)
	if len(eps) != 2 {
		t.Fatalf("expected 2 endpoints, got %d (%v)", len(eps), eps)
	}
	if !nearPoint(eps, 2, 3, 2) || !nearPoint(eps, 18, 3, 2) {
		t.Fatalf("endpoints not preserved near bar ends: %v", eps)
	}
	// One pixel width: every column contains at most one skeleton pixel.
	for x := 0; x < 21; x++ {
		count := 0
		for y := 0; y < 7; y++ {
			if skeleton.GrayAt(x, y).Y == foreground {
				count++
			}
		}
		if count > 1 {
			t.Fatalf("skeleton is wider than 1 pixel at column %d", x)
		}
	}
	// The skeleton must keep the horizontal extent of the bar.
	pixels := foregroundPixels(skeleton, foreground)
	minX, maxX := 21, 0
	for _, p := range pixels {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
	}
	if maxX-minX < 12 {
		t.Fatalf("skeleton is too short: extent [%d, %d]", minX, maxX)
	}
}

// A T shaped region must collapse into a skeleton with 3 endpoints and a junction
// near the center of the original shape.
func Test_Skeletonize_TShape(t *testing.T) {
	foreground := utils.MaxUint8
	pixels := append(rectPixels(2, 1, 18, 3), rectPixels(9, 4, 11, 13)...)
	img := newBinaryImage(21, 15, foreground, pixels)

	skeleton := Skeletonize(img, foreground)

	if components := countComponents(skeleton, foreground); components != 1 {
		t.Fatalf("expected 1 connected component, got %d", components)
	}
	eps := endpoints(skeleton, foreground)
	if len(eps) != 3 {
		t.Fatalf("expected 3 endpoints, got %d (%v)", len(eps), eps)
	}
	for _, tip := range []image.Point{{X: 2, Y: 2}, {X: 18, Y: 2}, {X: 10, Y: 13}} {
		if !nearPoint(eps, tip.X, tip.Y, 2) {
			t.Fatalf("no endpoint near expected tip %v: %v", tip, eps)
		}
	}
	js := junctions(skeleton, foreground)
	if len(js) == 0 {
		t.Fatal("expected at least one junction point")
	}
	if !nearPoint(js, 10, 3, 3) {
		t.Fatalf("junction not preserved near (10, 3): %v", js)
	}
}

// A cross shaped region must collapse into a skeleton with 4 endpoints and a
// junction near the center.
func Test_Skeletonize_Cross(t *testing.T) {
	foreground := utils.MaxUint8
	pixels := append(rectPixels(2, 9, 18, 11), rectPixels(9, 2, 11, 18)...)
	img := newBinaryImage(21, 21, foreground, pixels)

	skeleton := Skeletonize(img, foreground)

	if components := countComponents(skeleton, foreground); components != 1 {
		t.Fatalf("expected 1 connected component, got %d", components)
	}
	eps := endpoints(skeleton, foreground)
	if len(eps) != 4 {
		t.Fatalf("expected 4 endpoints, got %d (%v)", len(eps), eps)
	}
	for _, tip := range []image.Point{{X: 2, Y: 10}, {X: 18, Y: 10}, {X: 10, Y: 2}, {X: 10, Y: 18}} {
		if !nearPoint(eps, tip.X, tip.Y, 2) {
			t.Fatalf("no endpoint near expected tip %v: %v", tip, eps)
		}
	}
	js := junctions(skeleton, foreground)
	if len(js) == 0 {
		t.Fatal("expected at least one junction point")
	}
	if !nearPoint(js, 10, 10, 3) {
		t.Fatalf("junction not preserved near (10, 10): %v", js)
	}
}

// A thick closed ring must collapse into a 1 pixel wide closed loop: a single
// connected component without any endpoint.
func Test_Skeletonize_Ring(t *testing.T) {
	foreground := utils.MaxUint8
	img := newBinaryImage(31, 31, foreground, ringPixels(15, 15, 6, 8, 31, 31))

	skeleton := Skeletonize(img, foreground)

	if components := countComponents(skeleton, foreground); components != 1 {
		t.Fatalf("expected 1 connected component, got %d", components)
	}
	if eps := endpoints(skeleton, foreground); len(eps) != 0 {
		t.Fatalf("expected no endpoints on a closed ring, got %v", eps)
	}
	pixels := foregroundPixels(skeleton, foreground)
	if len(pixels) == 0 {
		t.Fatal("skeleton of a ring must not be empty")
	}
	// Every pixel of a closed loop has at least 2 neighbours.
	for _, p := range pixels {
		if n := neighbourCount(skeleton, foreground, p.X, p.Y); n < 2 {
			t.Fatalf("skeleton of a ring is broken at %v (neighbours: %d)", p, n)
		}
	}
}

// -------------------------------------------------------------------------------------------------------
// Robustness tests
// -------------------------------------------------------------------------------------------------------

// The skeleton of an image with a non zero origin must keep the origin and match the
// translated skeleton of the same image at the zero origin.
func Test_Skeletonize_NonZeroOrigin(t *testing.T) {
	foreground := utils.MaxUint8
	pixels := append(rectPixels(2, 9, 18, 11), rectPixels(9, 2, 11, 18)...)
	base := newBinaryImage(21, 21, foreground, pixels)

	// Embed the shape into a larger canvas and crop it with a non zero origin.
	canvas := image.NewGray(image.Rect(0, 0, 41, 41))
	for y := 0; y < 41; y++ {
		for x := 0; x < 41; x++ {
			canvas.SetGray(x, y, color.Gray{Y: utils.MinUint8})
		}
	}
	for y := 0; y < 21; y++ {
		for x := 0; x < 21; x++ {
			canvas.SetGray(x+10, y+10, base.GrayAt(x, y))
		}
	}
	cropped := canvas.SubImage(image.Rect(10, 10, 31, 31)).(*image.Gray)

	skeletonZero := Skeletonize(base, foreground)
	skeletonShifted := Skeletonize(cropped, foreground)

	if !skeletonShifted.Bounds().Eq(cropped.Bounds()) {
		t.Fatalf("expected bounds %v, got %v", cropped.Bounds(), skeletonShifted.Bounds())
	}
	bounds := skeletonZero.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if skeletonZero.GrayAt(x, y).Y != skeletonShifted.GrayAt(x+10, y+10).Y {
				t.Fatalf("skeleton differs at (%d, %d) for non zero origin", x, y)
			}
		}
	}
}

// Shapes touching the image edges must be thinned like interior shapes.
func Test_Skeletonize_ShapeTouchingEdges(t *testing.T) {
	foreground := utils.MaxUint8
	// Thick bar touching the left and right image edges.
	img := newBinaryImage(20, 9, foreground, rectPixels(0, 3, 19, 5))

	skeleton := Skeletonize(img, foreground)

	if components := countComponents(skeleton, foreground); components != 1 {
		t.Fatalf("expected 1 connected component, got %d", components)
	}
	eps := endpoints(skeleton, foreground)
	if len(eps) != 2 {
		t.Fatalf("expected 2 endpoints, got %d (%v)", len(eps), eps)
	}
	if !nearPoint(eps, 0, 4, 2) || !nearPoint(eps, 19, 4, 2) {
		t.Fatalf("endpoints not preserved at image edges: %v", eps)
	}
}

// An empty image must return an empty result without panicking.
func Test_Skeletonize_EmptyImage(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 0, 0))
	skeleton := Skeletonize(img, utils.MaxUint8)
	if !skeleton.Bounds().Eq(img.Bounds()) {
		t.Fatalf("expected bounds %v, got %v", img.Bounds(), skeleton.Bounds())
	}
}

// An image without any foreground pixel stays empty.
func Test_Skeletonize_NoForeground(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	skeleton := Skeletonize(img, utils.MaxUint8)
	if pixels := foregroundPixels(skeleton, utils.MaxUint8); len(pixels) != 0 {
		t.Fatalf("expected empty skeleton, got %d pixels", len(pixels))
	}
}

// A single foreground pixel is already a skeleton and must be preserved.
func Test_Skeletonize_SinglePixel(t *testing.T) {
	foreground := utils.MaxUint8
	img := newBinaryImage(5, 5, foreground, []image.Point{{X: 2, Y: 2}})

	skeleton := Skeletonize(img, foreground)

	pixels := foregroundPixels(skeleton, foreground)
	if len(pixels) != 1 || pixels[0] != image.Pt(2, 2) {
		t.Fatalf("expected single pixel at (2, 2), got %v", pixels)
	}
}

// A one pixel wide line is already a skeleton and must remain unchanged.
func Test_Skeletonize_SinglePixelLine(t *testing.T) {
	foreground := utils.MaxUint8
	var pixels []image.Point
	for x := 1; x < 19; x++ {
		pixels = append(pixels, image.Point{X: x, Y: 3})
	}
	img := newBinaryImage(20, 7, foreground, pixels)

	skeleton := Skeletonize(img, foreground)

	assertSkeletonEqual(t, img, skeleton)
}

// A thick solid block must collapse into a small connected skeleton and the result
// must be stable: skeletonizing it again changes nothing.
func Test_Skeletonize_SolidBlock(t *testing.T) {
	foreground := utils.MaxUint8
	img := newBinaryImage(40, 40, foreground, rectPixels(5, 5, 34, 34))

	skeleton := Skeletonize(img, foreground)

	if components := countComponents(skeleton, foreground); components != 1 {
		t.Fatalf("expected 1 connected component, got %d", components)
	}
	pixels := foregroundPixels(skeleton, foreground)
	if len(pixels) == 0 {
		t.Fatal("skeleton of a solid block must not be empty")
	}
	if len(pixels) > 30*30/4 {
		t.Fatalf("skeleton of a solid block should be thin, got %d pixels", len(pixels))
	}
	assertSkeletonEqual(t, skeleton, Skeletonize(skeleton, foreground))
}

// Black foreground on white background must work as well.
func Test_Skeletonize_BlackForeground(t *testing.T) {
	var foreground uint8 = utils.MinUint8
	img := newBinaryImage(21, 7, foreground, rectPixels(2, 2, 18, 4))

	skeleton := Skeletonize(img, foreground)

	if components := countComponents(skeleton, foreground); components != 1 {
		t.Fatalf("expected 1 connected component, got %d", components)
	}
	if eps := endpoints(skeleton, foreground); len(eps) != 2 {
		t.Fatalf("expected 2 endpoints, got %d (%v)", len(eps), eps)
	}
	// Background of the result must be white.
	if skeleton.GrayAt(0, 0).Y != utils.MaxUint8 {
		t.Fatalf("expected white background, got %d", skeleton.GrayAt(0, 0).Y)
	}
}

// Skeletonization must be deterministic: repeated runs produce identical results.
func Test_Skeletonize_Deterministic(t *testing.T) {
	foreground := utils.MaxUint8
	pixels := append(rectPixels(2, 1, 18, 3), rectPixels(9, 4, 11, 13)...)
	img := newBinaryImage(21, 15, foreground, pixels)

	first := Skeletonize(img, foreground)
	second := Skeletonize(img, foreground)
	assertSkeletonEqual(t, first, second)
}

// A huge solid image must terminate within the iteration bound with a stable result.
func Test_Skeletonize_LargeImage(t *testing.T) {
	foreground := utils.MaxUint8
	size := 1500
	img := image.NewGray(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetGray(x, y, color.Gray{Y: foreground})
		}
	}

	skeleton := Skeletonize(img, foreground)

	if components := countComponents(skeleton, foreground); components != 1 {
		t.Fatalf("expected 1 connected component, got %d", components)
	}
	pixels := foregroundPixels(skeleton, foreground)
	if len(pixels) == 0 {
		t.Fatal("skeleton of a huge solid image must not be empty")
	}
	if len(pixels) > size*size/10 {
		t.Fatalf("skeleton should be thin, got %d pixels", len(pixels))
	}
	assertSkeletonEqual(t, skeleton, Skeletonize(skeleton, foreground))
}
