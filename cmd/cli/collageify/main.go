package main

import (
	"bytes"
	"flag"
	"image"
	"image/jpeg"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/imageutil"
	"github.com/sprucehealth/backend/libs/imageutil/collage"
)

var config struct {
	imagePaths string
	outputPath string
}

func init() {
	flag.StringVar(&config.imagePaths, "image.paths", "", "space seperated list of local image paths")
	flag.StringVar(&config.outputPath, "out", "./out.jpg", "path for the resultant image")
}

func main() {
	boot.ParseFlags("COLLAGEIFY_")
	var images []image.Image
	imagePaths := strings.Fields(config.imagePaths)
	for _, imagePath := range imagePaths {
		dat, err := ioutil.ReadFile(imagePath)
		if err != nil {
			golog.Fatalf("Error while reading image file %s: %s", imagePath, err)
		}

		image, _, _, err := imageutil.DecodeImageAndExif(bytes.NewReader(dat))
		if err != nil {
			golog.Fatalf("Error while decoding image file %s: %s", imagePath, err)
		}

		images = append(images, image)
	}
	res, err := collage.Collageify(images, collage.SpruceProductGridLayout, &collage.Options{Height: 500, Width: 500})
	if err != nil {
		golog.Fatalf("Error while creating collage: %s", err)
	}

	out, err := os.Create(config.outputPath)
	if err != nil {
		golog.Fatalf("Error while creating output file: %s", err)
	}
	defer out.Close()

	if err := jpeg.Encode(out, res, &jpeg.Options{Quality: jpeg.DefaultQuality}); err != nil {
		golog.Fatalf("Error while writing out encoded image %s: %s", config.outputPath, err)
	}
}
