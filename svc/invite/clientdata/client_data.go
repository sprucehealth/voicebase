package clientdata

import (
	"encoding/json"
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/media"
)

type Popover struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	ButtonText string `json:"button_text"`
	PhotoURL   string `json:"photo_url"`
}

type OrganizationInvite struct {
	Popover Popover `json:"popover"`
	OrgID   string  `json:"org_id"`
	OrgName string  `json:"org_name"`
}

type ColleagueInviteClientData struct {
	OrganizationInvite OrganizationInvite `json:"organization_invite"`
}

type Greeting struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	PhotoURL   string `json:"photo_url"`
	ButtonText string `json:"button_text"`
}

type PatientInvite struct {
	Greeting Greeting `json:"greeting"`
	OrgID    string   `json:"org_id"`
	OrgName  string   `json:"org_name"`
}

type PatientInviteClientData struct {
	PatientInvite PatientInvite `json:"patient_invite"`
}

// PatientInviteClientJSON creates the invite client JSON required for patient invites
func PatientInviteClientJSON(org *directory.Entity, firstName, mediaAPIDomain string) (string, error) {
	welcomeText := "Welcome!"
	if firstName != "" {
		welcomeText = fmt.Sprintf("Welcome %s!", firstName)
	}
	pcd := PatientInviteClientData{
		PatientInvite: PatientInvite{
			Greeting: Greeting{
				Title:      welcomeText,
				Message:    fmt.Sprintf("Let's create your account so you can start securely messaging with %s.", org.Info.DisplayName),
				ButtonText: "Get Started",
			},
			OrgID:   org.ID,
			OrgName: org.Info.DisplayName,
		},
	}
	if mediaAPIDomain != "" && org.ImageMediaID != "" {
		pcd.PatientInvite.Greeting.PhotoURL = media.ThumbnailURL(mediaAPIDomain, org.ImageMediaID, 0, 0, false)
	}
	inviteClientDataJSON, err := json.Marshal(pcd)
	if err != nil {
		return "", errors.Trace(err)
	}
	return string(inviteClientDataJSON), nil
}

// ColleagueInviteClientJSON creates the invite client JSON required for colleague invites
func ColleagueInviteClientJSON(org *directory.Entity, inviter *directory.Entity, firstName, mediaAPIDomain string) (string, error) {
	welcomeText := "Welcome to Spruce!"
	if firstName != "" {
		welcomeText = fmt.Sprintf("Welcome %s!", firstName)
	}
	icd := ColleagueInviteClientData{
		OrganizationInvite: OrganizationInvite{
			Popover: Popover{
				Title:      welcomeText,
				Message:    inviter.Info.DisplayName + " has invited you to join them on Spruce.",
				ButtonText: "Okay",
			},
			OrgID:   org.ID,
			OrgName: org.Info.DisplayName,
		},
	}
	if mediaAPIDomain != "" && org.ImageMediaID != "" {
		icd.OrganizationInvite.Popover.PhotoURL = media.ThumbnailURL(mediaAPIDomain, org.ImageMediaID, 0, 0, false)
	}
	inviteClientDataJSON, err := json.Marshal(icd)
	if err != nil {
		return "", errors.Trace(err)
	}
	return string(inviteClientDataJSON), nil
}
