package responses

import (
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

type Doctor struct {
	ID               int64           `json:"id,string"`
	FirstName        string          `json:"first_name"`
	LastName         string          `json:"last_name"`
	MiddleName       string          `json:"middle_name,omitempty"`
	Prefix           string          `json:"prefix,omitempty"`
	Suffix           string          `json:"suffix,omitempty"`
	ShortTitle       string          `json:"short_title,omitempty"`
	LongTitle        string          `json:"long_title,omitempty"`
	ShortDisplayName string          `json:"short_display_name,omitempty"`
	LongDisplayName  string          `json:"long_display_name,omitempty"`
	Email            string          `json:"email"`
	AccountID        int64           `json:"account_id"`
	Phone            string          `json:"phone"`
	ThumbnailURL     string          `json:"thumbnail_url,omitempty"`
	Address          *common.Address `json:"address,omitempty"`
	PersonID         int64           `json:"person_id"`
	PromptStatus     string          `json:"prompt_status"`
	NPI              string          `json:"npi,string"`
	DEA              string          `json:"dea,string"`
	IsCC             bool            `json:"is_ma"`

	// Deprecated
	LargeThumbnailURL string `json:"large_thumbnail_url,omitempty"`
	SmallThumbnailURL string `json:"small_thumbnail_url,omitempty"`
}

// TransformDoctor takes the model object and returns a populated doctor object.
func TransformDoctor(doctor *common.Doctor, apiDomain string) *Doctor {
	role := api.RoleDoctor
	if doctor.IsCC {
		role = api.RoleCC
	}
	return &Doctor{
		ID:                doctor.ID.Int64(),
		FirstName:         doctor.FirstName,
		LastName:          doctor.LastName,
		MiddleName:        doctor.MiddleName,
		Prefix:            doctor.Prefix,
		Suffix:            doctor.Suffix,
		ShortTitle:        doctor.ShortTitle,
		LongTitle:         doctor.LongTitle,
		ShortDisplayName:  doctor.ShortDisplayName,
		LongDisplayName:   doctor.LongDisplayName,
		Email:             doctor.Email,
		AccountID:         doctor.AccountID.Int64(),
		Phone:             doctor.CellPhone.String(),
		LargeThumbnailURL: app_url.ThumbnailURL(apiDomain, role, doctor.ID.Int64()),
		SmallThumbnailURL: app_url.ThumbnailURL(apiDomain, role, doctor.ID.Int64()),
		ThumbnailURL:      app_url.ThumbnailURL(apiDomain, role, doctor.ID.Int64()),
		Address:           doctor.Address,
		PersonID:          doctor.PersonID,
		PromptStatus:      doctor.PromptStatus.String(),
		NPI:               doctor.NPI,
		DEA:               doctor.DEA,
		IsCC:              doctor.IsCC,
	}
}

// Alert represents an intake from the patient that is significant enough to
// call out for the doctor when reviewing a patient's information.
type Alert struct {
	Message      string    `json:"message"`
	CreationDate time.Time `json:"creation_date"`
}

func TransformAlert(alert *common.Alert) *Alert {
	return &Alert{
		Message:      alert.Message,
		CreationDate: alert.CreationDate,
	}
}
