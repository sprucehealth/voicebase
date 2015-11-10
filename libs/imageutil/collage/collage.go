package collage

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/imageutil"
)

// LayoutAlgorithm represents an interface that provides the layout structuring algorithm
type LayoutAlgorithm func(images []image.Image, ops *Options) (image.Image, error)

// Options represents the configurable aspects of the resultant collage image
type Options struct {
	Height            int
	Width             int
	ImageHeightScalar float64
	ImageWidthScalar  float64
	ScaleToFill       bool
	CenterRowIsolated bool
}

// Collageify performs option validations and then performs the collage creation using the provided algorithm
func Collageify(images []image.Image, layout LayoutAlgorithm, ops *Options) (image.Image, error) {
	img, err := layout(images, ops)
	return img, errors.Trace(err)
}

var defaultResizeOps = &imageutil.Options{AllowScaleUp: true}

// SpruceProductGridLayout is a layout that follow the specs laid down for spruce product collages
var SpruceProductGridLayout LayoutAlgorithm = func(images []image.Image, ops *Options) (image.Image, error) {
	if ops.Width == 0 {
		return nil, errors.Trace(errors.New("Collage option Height must be non zero"))
	}

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

	// If a height wasn't provided, figure out our dynamic height
	if ops.Height == 0 {
		rows := math.Ceil(float64(len(images)) / float64(cols))
		ops.Height = int(rowHeight * rows)
	}
	result := image.NewRGBA(image.Rect(0, 0, ops.Width, ops.Height))
	// draw a uniform white color onto the image
	draw.Draw(result, image.Rect(0, 0, ops.Width, ops.Height), image.NewUniform(color.White), image.Point{X: 0, Y: 0}, draw.Src)
	for i, im := range images {
		imageHeightScalar := ops.ImageHeightScalar
		imageWidthScalar := ops.ImageWidthScalar
		columnWidthScalar := 1.0
		rowHeightScalar := 1.0
		layoutLocation := i
		if layout[i] {
			for li, occupied := range layout {
				layoutLocation = li
				if !occupied {
					break
				}
			}
		}

		// Occupy the layout location we are about to populate
		layout[layoutLocation] = true

		// If we're centering row isolated images move the layout location
		if ops.CenterRowIsolated && cols == 3 && (i%cols) == 0 && i == (len(images)-1) {
			layout[layoutLocation] = false
			layoutLocation++
			layout[layoutLocation] = true
		}

		// If there is a full row or more empty and this is the first image scale it up to fill 4 slots, ignore this case for our special case 2
		if ops.ScaleToFill && layoutLocation == 0 && ((cols*cols)-len(images)) >= cols && len(images) != 2 {
			columnWidthScalar = columnWidthScalar * 2
			rowHeightScalar = rowHeightScalar * 2
			if (layoutLocation % cols) != 0 {
				return nil, errors.Trace(fmt.Errorf("A scaled image can only be placed at the beginning of a row, got index %d for %d cols", layoutLocation, cols))
			}
			// Throw an error if any of the 4 grid locations for a scaled image are already occupied
			if layout[layoutLocation+1] || layout[layoutLocation+cols] || layout[layoutLocation+cols+1] {
				return nil, errors.Trace(fmt.Errorf("Found an image occupying the layout space needed for scaled index at index %d for %d cols", layoutLocation, cols))
			}
			// Occupy the scaled locations so the next images know where to go
			layout[layoutLocation+1], layout[layoutLocation+cols], layout[layoutLocation+cols+1] = true, true, true
		}

		// Determine where we are in the layout
		currentCol := float64(layoutLocation % cols)
		currentRow := math.Floor(float64(layoutLocation / cols))

		// Scale each image down/up to the correct size
		bounds := im.Bounds()
		width := colWidth * columnWidthScalar * imageWidthScalar
		height := rowHeight * rowHeightScalar * imageHeightScalar
		if bounds.Dx() != bounds.Dy() {
			if bounds.Dx() >= bounds.Dy() {
				height = (float64(bounds.Dy()) / float64(bounds.Dx())) * height
			} else {
				width = (float64(bounds.Dx()) / float64(bounds.Dy())) * width
			}
		}
		var err error
		im, err = imageutil.ResizeImage(im, int(width), int(height), defaultResizeOps)
		if err != nil {
			return nil, errors.Trace(err)
		}

		// Draw each scaled image into the final collage at the appropriate location
		innerDx := ((colWidth * columnWidthScalar) - width) / 2.0
		innerDy := ((rowHeight * rowHeightScalar) - height) / 2.0
		x0 := (currentCol * (colWidth * columnWidthScalar)) + innerDx
		x1 := x0 + width
		y0 := (currentRow * (rowHeight * rowHeightScalar)) + innerDy
		y1 := y0 + height
		draw.Draw(result, image.Rect(int(x0), int(y0), int(x1), int(y1)), im, image.Point{X: 0, Y: 0}, draw.Over)
	}

	return result, nil
}

func drawLayout(layout []bool, cols int) {
	for i := 0; i < (cols * cols); i++ {
		if (i % cols) == 0 {
			fmt.Println(strings.Repeat("+-", cols) + "+")
		}
		c := " "
		if layout[i] {
			c = "x"
		}
		fmt.Printf("|%s", c)
		if (i % cols) == (cols - 1) {
			fmt.Printf("|\n")
		}
	}
	if cols > 0 {
		fmt.Println(strings.Repeat("+-", cols) + "+")
	}
}
