package app_url

import (
	"fmt"
	"strings"

	"github.com/sprucehealth/backend/environment"
)

const (
	spruceUrlScheme = "spruce:///"
	spruceImageUrl  = spruceUrlScheme + "image/"
	spruceActionUrl = spruceUrlScheme + "action/"
)

func GetLargeThumbnail(role string, id int64) *SpruceAsset {
	if environment.IsDev() || environment.IsDemo() {
		id = 1
	}

	return &SpruceAsset{
		name: fmt.Sprintf("%s_%d_large", strings.ToLower(role), id),
	}
}

func GetSmallThumbnail(role string, id int64) *SpruceAsset {
	if environment.IsDev() || environment.IsDemo() {
		id = 1
	}

	return &SpruceAsset{
		name: fmt.Sprintf("%s_%d_small", strings.ToLower(role), id),
	}
}

func GetProfile(role string, id int64) *SpruceAsset {
	if environment.IsDev() || environment.IsDemo() {
		id = 1
	}

	return &SpruceAsset{
		name: fmt.Sprintf("%s_%d_profile", strings.ToLower(role), id),
	}
}
