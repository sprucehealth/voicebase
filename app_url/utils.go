package app_url

import "fmt"

const (
	spruceURLScheme = "spruce:///"
	spruceImageURL  = spruceURLScheme + "image/"
	spruceActionURL = spruceURLScheme + "action/"
)

func LargeThumbnailURL(apiDomain, role string, id int64) string {
	return thumbnailURL(apiDomain, role, id, "large")
}

func SmallThumbnailURL(apiDomain, role string, id int64) string {
	return thumbnailURL(apiDomain, role, id, "small")
}

func thumbnailURL(apiDomain, role string, id int64, size string) string {
	return fmt.Sprintf("https://%s/v1/thumbnail?role=%s&role_id=%d&size=%s", apiDomain, role, id, size)
}

func PrescriptionIcon(route, form string) *SpruceAsset {
	switch route {
	case "topic":
		return IconPrescriptionTopical
	case "oral":
		return IconPrescriptionOral
	}
	return IconRXLarge
}
