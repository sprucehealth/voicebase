package care

import (
	"encoding/json"
	"fmt"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/svc/directory"
	"github.com/sprucehealth/backend/svc/layout"
)

// Visit

type header struct {
	Title    string `json:"title,omitempy"`
	Subtitle string `json:"subtitle,omitempty"`
	IconURL  string `json:"icon_url,omitempty"`
}

type checkout struct {
	HeaderImageURL string `json:"header_image_url,omitempty"`
	HeaderText     string `json:"header_text,omitempty"`
	FooterText     string `json:"footer_text,omitempty"`
}

type submissionConfirmation struct {
	Title       string `json:"title,omitempty"`
	TopText     string `json:"top_text,omitempty"`
	BottomText  string `json:"bottom_text,omitempty"`
	ButtonTitle string `json:"button_title,omitempty"`
}

type intakeContainer struct {
	ID                     string                  `json:"id"`
	Header                 *header                 `json:"header,omitempty"`
	Checkout               *checkout               `json:"checkout,omitempty"`
	SubmissionConfirmation *submissionConfirmation `json:"submission_confirmation,omitempty"`
	Intake                 *layout.Intake          `json:"intake,omitempty"`
	Answers                json.RawMessage         `json:"answers,omitempty"`
	RequireAddress         bool                    `json:"require_address,omitempty"`
	Preferences            map[string]interface{}  `json:"preferences,omitempty"`
}

type VisitData struct {
	PatientAnswersJSON []byte
	Visit              *Visit
	OrgEntity          *directory.Entity
	Preferences        map[string]interface{}
}

// PopulateVisitIntake returns a json representation of the visit as understood by the clients to parse and process
// the visit.
func PopulateVisitIntake(intake *layout.Intake, data *VisitData) ([]byte, error) {
	var orgName string
	if data.OrgEntity != nil {
		orgName = data.OrgEntity.Info.DisplayName
	}
	container := &intakeContainer{
		ID: data.Visit.ID,
		Header: &header{
			Title:    "Submit Visit",
			Subtitle: "", // TODO
		},
		Checkout: &checkout{
			HeaderText: fmt.Sprintf("Submit your visit for %s to review.", orgName),
			FooterText: "", // TODO
		},
		SubmissionConfirmation: &submissionConfirmation{
			Title:       "Visit Submitted",
			TopText:     "Your visit has been submitted!",
			BottomText:  fmt.Sprintf("%s will review your visit shortly.", orgName),
			ButtonTitle: "Continue",
		},
		Intake:      intake,
		Answers:     json.RawMessage(data.PatientAnswersJSON),
		Preferences: data.Preferences,
	}

	containerData, err := json.Marshal(container)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return containerData, nil
}
