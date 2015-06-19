package response

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/patient_case/model"
)

type PatientCaseNote struct {
	ID             int64  `json:"id,string"`
	CaseID         int64  `json:"case_id,string"`
	AuthorDoctorID int64  `json:"author_doctor_id,string"`
	AuthorName     string `json:"author_name,omitempty"`
	AuthorPhotoURL string `json:"author_photo_url,omitempty"`
	Created        int64  `json:"created"`
	Modified       int64  `json:"modified"`
	NoteText       string `json:"note_text"`
}

type PatientCaseNoteOptionalData struct {
	AuthorName     string
	AuthorPhotoURL string
}

func TransformPatientCaseNote(n *model.PatientCaseNote) *PatientCaseNote {
	return &PatientCaseNote{
		ID:             n.ID,
		CaseID:         n.CaseID,
		AuthorDoctorID: n.AuthorDoctorID,
		Created:        n.Created.Unix(),
		Modified:       n.Modified.Unix(),
		NoteText:       string(n.NoteText),
	}
}

func AddPatientCaseNoteOptionalData(n *PatientCaseNote, od *PatientCaseNoteOptionalData) {
	n.AuthorName = od.AuthorName
	n.AuthorPhotoURL = od.AuthorPhotoURL
}

func NewPatientCaseNoteOptionalData(d *common.Doctor, apiDomain string) *PatientCaseNoteOptionalData {
	role := api.RoleDoctor
	if d.IsCC {
		role = api.RoleCC
	}
	return &PatientCaseNoteOptionalData{
		AuthorName:     d.ShortDisplayName,
		AuthorPhotoURL: app_url.ThumbnailURL(apiDomain, role, d.ID.Int64()),
	}
}
