package app_url

import (
	"fmt"
	"strings"
)

const (
	spruceUrlScheme = "spruce:///"
	spruceImageUrl  = spruceUrlScheme + "image/"
	spruceActionUrl = spruceUrlScheme + "action/"
)

// MapImagesToSingleDoctor enables us to have a single
// set of images for doctors in non-production environments
var MapImagesToSingleDoctor = false

func GetLargeThumbnail(role string, id int64) *SpruceAsset {
	if MapImagesToSingleDoctor {
		id = 1
	}

	return &SpruceAsset{
		name: fmt.Sprintf("%s_%d_large", strings.ToLower(role), id),
	}
}

func GetSmallThumbnail(role string, id int64) *SpruceAsset {
	if MapImagesToSingleDoctor {
		id = 1
	}

	return &SpruceAsset{
		name: fmt.Sprintf("%s_%d_small", strings.ToLower(role), id),
	}
}

func GetProfile(role string, id int64) *SpruceAsset {
	if MapImagesToSingleDoctor {
		id = 1
	}

	return &SpruceAsset{
		name: fmt.Sprintf("%s_%d_profile", strings.ToLower(role), id),
	}
}
