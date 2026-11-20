// Package skeleton implements iterative thinning (skeletonization) of binary images.
//
// The Skeletonize function reduces the foreground of a black and white image to its
// one pixel wide medial axis (skeleton) using the Zhang-Suen thinning algorithm.
// The skeleton preserves the connected components, the endpoints and the junction
// points of the original foreground, which makes it suitable for placing waypoints
// along thick hand drawn paths, such as rivers on a level map.
package skeleton

import (
	"image"
	"image/color"

	"github.com/ernyoke/imger/utils"
)

// Skeletonize takes a binary (black and white) 8 bit grayscale image and iteratively thins its
// foreground until a one pixel wide skeleton remains. The foreground parameter selects which
// pixel value is treated as foreground (usually utils.MaxUint8 for white foreground on black
// background or utils.MinUint8 for black foreground on white background). Every other pixel is
// treated as background.
//
// The returned image has the same bounds as the input (a non zero origin is preserved) and
// contains the skeleton pixels painted with the foreground value on a uniform background.
// The algorithm is deterministic: the pixel scan order does not influence the result, because
// deletions of each sub-iteration are collected first and applied afterwards. The number of
// iterations is bounded, so degenerate inputs (empty images, single pixels, closed rings or
// huge solid blocks) always terminate with a stable result.
func Skeletonize(img *image.Gray, foreground uint8) *image.Gray {
	bounds := img.Bounds()
	size := bounds.Size()
	result := image.NewGray(bounds)
	if size.X <= 0 || size.Y <= 0 {
		return result
	}

	var background uint8 = utils.MinUint8
	if foreground == utils.MinUint8 {
		background = utils.MaxUint8
	}

	w, h := size.X, size.Y
	offset := bounds.Min
	fg := make([]bool, w*h)
	utils.ParallelForEachPixel(size, func(x, y int) {
		if img.GrayAt(x+offset.X, y+offset.Y).Y == foreground {
			fg[y*w+x] = true
			result.SetGray(x+offset.X, y+offset.Y, color.Gray{Y: foreground})
		} else {
			result.SetGray(x+offset.X, y+offset.Y, color.Gray{Y: background})
		}
	})

	thin(fg, w, h)

	utils.ParallelForEachPixel(size, func(x, y int) {
		if fg[y*w+x] {
			result.SetGray(x+offset.X, y+offset.Y, color.Gray{Y: foreground})
		} else {
			result.SetGray(x+offset.X, y+offset.Y, color.Gray{Y: background})
		}
	})
	return result
}

// thin iteratively removes border pixels from the foreground mask until only the skeleton
// remains. Only pixels on the moving frontier (foreground pixels touching the background)
// are evaluated, which keeps the algorithm fast even for huge solid areas.
func thin(fg []bool, w, h int) {
	queued := make([]bool, w*h)
	var frontier []int
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if fg[y*w+x] && touchesBackground(fg, w, h, x, y) {
				frontier = append(frontier, y*w+x)
				queued[y*w+x] = true
			}
		}
	}

	// Hard upper bound on the number of sub-iterations. Each pair of sub-iterations peels
	// at least one layer from every remaining thick region, so w+h is a safe cap; the loop
	// normally terminates much earlier once no pixel can be deleted anymore.
	maxIterations := 2*(w+h) + 16
	step := 0
	stableSubIterations := 0
	// The frontier is a persistent set of candidate pixels: unevaluated deletable pixels
	// stay in it, because a pixel which survives one sub-iteration may become deletable in
	// the other one (or after a neighbour was deleted). Deleted pixels are skipped through
	// the fg lookup. The loop ends once two consecutive sub-iterations (one of each kind)
	// delete nothing.
	for iteration := 0; iteration < maxIterations && stableSubIterations < 2; iteration++ {
		var toDelete []int
		for _, i := range frontier {
			if fg[i] && isDeletable(fg, w, h, i%w, i/w, step) {
				toDelete = append(toDelete, i)
			}
		}

		if len(toDelete) == 0 {
			stableSubIterations++
			step ^= 1
			continue
		}
		stableSubIterations = 0

		for _, i := range toDelete {
			fg[i] = false
		}

		// The foreground neighbours of the deleted pixels join the frontier.
		for _, i := range toDelete {
			x, y := i%w, i/w
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := x+dx, y+dy
					if nx < 0 || ny < 0 || nx >= w || ny >= h {
						continue
					}
					j := ny*w + nx
					if fg[j] && !queued[j] {
						queued[j] = true
						frontier = append(frontier, j)
					}
				}
			}
		}
		step ^= 1
	}
}

// touchesBackground reports whether the foreground pixel at (x, y) has at least one
// background pixel in its 8-neighbourhood. Pixels outside of the image count as background,
// so shapes touching the image edges are thinned correctly.
func touchesBackground(fg []bool, w, h, x, y int) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx < 0 || ny < 0 || nx >= w || ny >= h || !fg[ny*w+nx] {
				return true
			}
		}
	}
	return false
}

// isDeletable implements the Zhang-Suen thinning conditions for the foreground pixel at
// (x, y). The step parameter selects between the two sub-iterations (0 or 1) of a pass.
//
// The 8-neighbourhood is numbered as follows:
//
//	P9 P2 P3
//	P8 P1 P4
//	P7 P6 P5
func isDeletable(fg []bool, w, h, x, y, step int) bool {
	p := func(dx, dy int) bool {
		nx, ny := x+dx, y+dy
		if nx < 0 || ny < 0 || nx >= w || ny >= h {
			return false
		}
		return fg[ny*w+nx]
	}
	p2 := p(0, -1)
	p3 := p(1, -1)
	p4 := p(1, 0)
	p5 := p(1, 1)
	p6 := p(0, 1)
	p7 := p(-1, 1)
	p8 := p(-1, 0)
	p9 := p(-1, -1)

	neighbours := []bool{p2, p3, p4, p5, p6, p7, p8, p9}

	// B(P1): number of foreground neighbours.
	b := 0
	for _, n := range neighbours {
		if n {
			b++
		}
	}
	if b < 2 || b > 6 {
		return false
	}

	// A(P1): number of 0 -> 1 transitions in the ordered sequence P2, P3, ..., P9, P2.
	a := 0
	for i := 0; i < len(neighbours); i++ {
		if !neighbours[i] && neighbours[(i+1)%len(neighbours)] {
			a++
		}
	}
	if a != 1 {
		return false
	}

	if step == 0 {
		// First sub-iteration: P2 * P4 * P6 = 0 and P4 * P6 * P8 = 0.
		return !(p2 && p4 && p6) && !(p4 && p6 && p8)
	}
	// Second sub-iteration: P2 * P4 * P8 = 0 and P2 * P6 * P8 = 0.
	return !(p2 && p4 && p8) && !(p2 && p6 && p8)
}
