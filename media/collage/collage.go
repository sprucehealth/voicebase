package collage

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/media"
)

// LayoutAlgorithm represents an interface that provides the layout structuring algorithm
type LayoutAlgorithm func(images []image.Image, ops *Options) (image.Image, error)

// Options represents the configurable aspects of the resultant collage image
type Options struct {
	Height int
	Width  int
}

// Collageify performs option validations and then performs the collage creation using the provided algorithm
func Collageify(images []image.Image, layout LayoutAlgorithm, ops *Options) (image.Image, error) {
	if ops.Width == 0 || ops.Height == 0 {
		return nil, errors.Trace(errors.New("Collage options Width, and Height must be non zero"))
	}

	img, err := layout(images, ops)
	return img, errors.Trace(err)
}

var defaultResizeOps = &media.Options{AllowScaleUp: true}

// SpruceProductGridLayout is a layout that uses the following conventions
// * All images fit into equally sized grid sections
// * No cropping of images is performed
// * When an image does not fit the gird layout, it's largest dimension is chosen and scaled tointo a square preserving that dimension
var SpruceProductGridLayout LayoutAlgorithm = func(images []image.Image, ops *Options) (image.Image, error) {
	cols := 1
	if len(images) > 1 {
		cols = 2
	}
	if len(images) > 4 {
		cols = 3
	}
	if len(images) > 9 {
		images = images[:9]
	}

	layout := make([]bool, cols*cols)
	colWidth := float64(ops.Width) / float64(cols)
	rowHeight := colWidth
	// Special case two images to be centered in two columns
	if len(images) == 2 {
		rowHeight = float64(ops.Height)
		layout = make([]bool, cols)
	}
	result := image.NewRGBA(image.Rect(0, 0, ops.Width, ops.Height))
	// draw a uniform white color onto the image
	draw.Draw(result, image.Rect(0, 0, ops.Width, ops.Height), image.NewUniform(color.White), image.Point{X: 0, Y: 0}, draw.Src)
	for i, im := range images {
		heightScalar := 1.0
		widthScalar := 1.0
		layoutLocation := i
		if layout[i] {
			for li, occupied := range layout {
				if !occupied {
					layoutLocation = li
				}
			}
		}
		// Occupy the layout location we are about to populate
		layout[layoutLocation] = true

		// If there is a full row or more empty and this is the first image scale it up to fill 4 slots, ignore this case for our special case 2
		if layoutLocation == 0 && ((cols*cols)-len(images)) >= cols && len(images) != 2 {
			heightScalar = 2.0
			widthScalar = 2.0
			if (layoutLocation % cols) != 0 {
				return nil, errors.Trace(fmt.Errorf("A scaled image can only be placed at the beginning of a row, got index %d for %d cols", layoutLocation, cols))
			}
			// Throw an error if any of the 4 grid locations for a scaled image are already occupied
			if layout[layoutLocation+1] || layout[layoutLocation+cols] || layout[layoutLocation+cols+1] {
				return nil, errors.Trace(fmt.Errorf("Found an image occupying the layout space needed for scaled index at index %d for %d cols", layoutLocation, cols))
			}
			// Occupy the scaled locations to the next images know where to go
			layout[layoutLocation+1], layout[layoutLocation+cols], layout[layoutLocation+cols+1] = true, true, true
		}

		// Determine where we are in the layout
		currentCol := float64(layoutLocation % cols)
		currentRow := math.Floor(float64(layoutLocation / cols))

		// Scale each image down/up to the correct size
		bounds := im.Bounds()
		width := colWidth * widthScalar
		height := colWidth * heightScalar
		if bounds.Dx() != bounds.Dy() {
			if bounds.Dx() >= bounds.Dy() {
				height = (float64(bounds.Dy()) / float64(bounds.Dx())) * (rowHeight * heightScalar)
			} else {
				width = (float64(bounds.Dx()) / float64(bounds.Dy())) * (colWidth * widthScalar)
			}
		}
		var err error
		im, err = media.ResizeImage(im, int(width), int(height), defaultResizeOps)
		if err != nil {
			return nil, errors.Trace(err)
		}

		// Draw each scaled image into the final collage at the appropriate location
		innerDx := ((colWidth * widthScalar) - width) / 2.0
		innerDy := ((rowHeight * heightScalar) - height) / 2.0
		x0 := (currentCol * (colWidth * widthScalar)) + innerDx
		x1 := x0 + width
		y0 := (currentRow * (rowHeight * heightScalar)) + innerDy
		y1 := y0 + height
		draw.Draw(result, image.Rect(int(x0), int(y0), int(x1), int(y1)), im, image.Point{X: 0, Y: 0}, draw.Src)
	}

	return result, nil
}
