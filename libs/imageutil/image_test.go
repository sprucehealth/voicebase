package imageutil

import (
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/sprucehealth/backend/test"
)

// TestResizeImageFromReader tests basic functionality on different image types.
// It doesn't try to test all functionality as that's left to TestResizeImage.
func TestResizeImageFromReader(t *testing.T) {
	// Check if we should save the generated files
	outPath := os.Getenv("TEST_MEDIA_OUTPUT_PATH")
	testFiles, err := filepath.Glob("testdata/*")
	test.OK(t, err)
	for _, fn := range testFiles {
		if filepath.Base(fn)[0] == '.' {
			// Ignore files like .DS_Store
			continue
		}
		t.Logf("Testing %s", fn)
		f, err := os.Open(fn)
		test.OK(t, err)
		img, _, err := ResizeImageFromReader(f, 128, 64, nil)
		test.OK(t, err)
		f.Close()
		test.Equals(t, 128, img.Bounds().Dx())
		test.Equals(t, 64, img.Bounds().Dy())
		if outPath != "" {
			f, err := os.Create(filepath.Join(outPath, filepath.Base(fn)))
			test.OK(t, err)
			test.OK(t, png.Encode(f, img))
			f.Close()
		}
	}
}

func TestCalcSize(t *testing.T) {
	cases := []struct {
		// these names are terse to make creating the test cases easier to read
		iw int // image width
		ih int // image height
		rw int // requested width
		rh int // requested height
		o  *Options
		es sizeOp
	}{
		// Same in, sam eout
		{iw: 600, ih: 600, rw: 600, rh: 600, o: &Options{AllowScaleUp: true, Crop: true}, es: sizeOp{w: 600, h: 600, rw: 600, rh: 600, crop: false}},
		// Upscale no crop
		{iw: 300, ih: 300, rw: 600, rh: 600, o: &Options{AllowScaleUp: true, Crop: true}, es: sizeOp{w: 600, h: 600, rw: 600, rh: 600, crop: false}},
		// No upscale no crop (same in, same out)
		{iw: 300, ih: 300, rw: 600, rh: 600, o: &Options{AllowScaleUp: false, Crop: true}, es: sizeOp{w: 300, h: 300, rw: 300, rh: 300, crop: false}},
		// Downscale no crop
		{iw: 300, ih: 300, rw: 200, rh: 200, o: &Options{AllowScaleUp: false, Crop: true}, es: sizeOp{w: 200, h: 200, rw: 200, rh: 200, crop: false}},
		// Crop width
		{iw: 600, ih: 300, rw: 200, rh: 200, o: &Options{AllowScaleUp: false, Crop: true}, es: sizeOp{w: 200, h: 200, rw: 400, rh: 200, crop: true}},
		// Bound same aspect ratio
		{iw: 600, ih: 600, rw: 200, rh: 200, o: &Options{AllowScaleUp: false, Crop: false}, es: sizeOp{w: 200, h: 200, rw: 200, rh: 200, crop: false}},
		// Bound larger width
		{iw: 400, ih: 200, rw: 200, rh: 200, o: &Options{AllowScaleUp: false, Crop: false}, es: sizeOp{w: 200, h: 100, rw: 200, rh: 100, crop: false}},
		// Bound larger height
		{iw: 200, ih: 400, rw: 200, rh: 200, o: &Options{AllowScaleUp: false, Crop: false}, es: sizeOp{w: 100, h: 200, rw: 100, rh: 200, crop: false}},
		// Width only
		{iw: 400, ih: 200, rw: 100, rh: 0, o: &Options{AllowScaleUp: false, Crop: true}, es: sizeOp{w: 100, h: 50, rw: 100, rh: 50, crop: false}},
		{iw: 400, ih: 200, rw: 100, rh: 0, o: &Options{AllowScaleUp: false, Crop: false}, es: sizeOp{w: 100, h: 50, rw: 100, rh: 50, crop: false}},
		// Height only
		{iw: 400, ih: 200, rw: 0, rh: 100, o: &Options{AllowScaleUp: false, Crop: true}, es: sizeOp{w: 200, h: 100, rw: 200, rh: 100, crop: false}},
		{iw: 400, ih: 200, rw: 0, rh: 100, o: &Options{AllowScaleUp: false, Crop: false}, es: sizeOp{w: 200, h: 100, rw: 200, rh: 100, crop: false}},
	}
	for _, c := range cases {
		s := calcSize(c.iw, c.ih, c.rw, c.rh, c.o)
		if s != c.es {
			t.Errorf("calcSize(%dx%d, %dx%d, %+v) = %+v, expected %+v", c.iw, c.ih, c.rw, c.rh, c.o, s, c.es)
		}
	}
}
