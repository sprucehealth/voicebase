package app_url

import "fmt"

const (
	spruceURLScheme = "spruce:///"
	spruceImageURL  = spruceURLScheme + "image/"
	spruceActionURL = spruceURLScheme + "action/"
)

func ThumbnailURL(apiDomain, role string, id int64) string {
	return profilImageURL(apiDomain, role, id, "thumbnail")
}

func HeroImageURL(apiDomain, role string, id int64) string {
	return profilImageURL(apiDomain, role, id, "hero")
}

func profilImageURL(apiDomain, role string, id int64, profileImageType string) string {
	return fmt.Sprintf("https://%s/v1/profile_image?role=%s&role_id=%d&type=%s", apiDomain, role, id, profileImageType)
}

func PrescriptionIcon(route string) *SpruceAsset {
	switch route {
	case "topical":
		return IconPrescriptionTopical
	case "oral":
		return IconPrescriptionOral
	}
	return IconRXLarge
}
