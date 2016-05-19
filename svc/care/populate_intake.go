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
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	IconURL  string `json:"icon_url"`
}

type checkout struct {
	HeaderImageURL string `json:"header_image_url"`
	HeaderText     string `json:"header_text"`
	FooterText     string `json:"footer_text"`
}

type submissionConfirmation struct {
	Title       string `json:"title"`
	TopText     string `json:"top_text"`
	BottomText  string `json:"bottom_text"`
	ButtonTitle string `json:"button_title"`
}

type intakeContainer struct {
	ID                     string                  `json:"id"`
	Header                 *header                 `json:"header"`
	Checkout               *checkout               `json:"checkout"`
	SubmissionConfirmation *submissionConfirmation `json:"submission_confirmation"`
	Intake                 *layout.Intake          `json:"intake"`
	Answers                json.RawMessage         `json:"answers"`
	RequireAddress         bool                    `json:"require_address"`
}

type VisitData struct {
	PatientAnswersJSON []byte
	Visit              *Visit
	OrgEntity          *directory.Entity
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
			TopText:     fmt.Sprintf("Your %s visit has been submitted.", data.Visit.Name),
			BottomText:  "Your care team will review your visit and respond for any additional questions.",
			ButtonTitle: "Continue",
		},
		Intake:  intake,
		Answers: json.RawMessage(data.PatientAnswersJSON),
	}

	containerData, err := json.Marshal(container)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return containerData, nil
}
