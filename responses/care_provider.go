package responses

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

type CareProvider struct {
	ProviderID       int64  `json:"provider_id,string"`
	FirstName        string `json:"first_name,omitempty"`
	LastName         string `json:"last_name,omitempty"`
	ShortTitle       string `json:"short_title,omitempty"`
	LongTitle        string `json:"long_title,omitempty"`
	ShortDisplayName string `json:"short_display_name,omitempty"`
	LongDisplayName  string `json:"long_display_name,omitempty"`
	ThumbnailURL     string `json:"thumbnail_url,omitempty"`
}

func NewCareProviderFromDoctorDBModel(d *common.Doctor, apiDomain string) *CareProvider {
	return &CareProvider{
		ProviderID:       d.DoctorID.Int64(),
		FirstName:        d.FirstName,
		LastName:         d.LastName,
		ShortTitle:       d.ShortTitle,
		LongTitle:        d.LongTitle,
		ShortDisplayName: d.ShortDisplayName,
		LongDisplayName:  d.LongDisplayName,
		ThumbnailURL:     app_url.ThumbnailURL(apiDomain, api.DOCTOR_ROLE, d.DoctorID.Int64()),
	}
}
