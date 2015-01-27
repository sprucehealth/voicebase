package handlers

import (
	"net/http"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_diagnosisSearch struct {
	api.DataAPI
	patientVisit *common.PatientVisit
}

func (m *mockDataAPI_diagnosisSearch) GetPatientVisitFromID(patientVisitID int64) (*common.PatientVisit, error) {
	return m.patientVisit, nil
}

func TestDiagnosisSearch_RequestParameterParsing(t *testing.T) {

	r, err := http.NewRequest("GET", "https://api.spruce.local?query=eag&pathway_id=acne&patient_visit_id=6", nil)
	test.OK(t, err)

	m := &mockDataAPI_diagnosisSearch{}

	s := searchHandler{
		dataAPI: m,
	}
	pathwayTag := s.determinePathwayTag(r)
	test.Equals(t, "acne", pathwayTag)

	// ensure that if the pathway_id is not present then there is an attempt
	// to pick up the pathway tag from the patient visit
	m.patientVisit = &common.PatientVisit{
		PathwayTag: "tag from visit",
	}
	r, err = http.NewRequest("GET", "https://api.spruce.local?query=eag&patient_visit_id=6", nil)
	test.OK(t, err)
	pathwayTag = s.determinePathwayTag(r)
	test.Equals(t, m.patientVisit.PathwayTag, pathwayTag)

	// ensure that acne pathway tag is returned if not present as pathway_id or via patient_visit_id in query paramters
	r, err = http.NewRequest("GET", "https://api.spruce.local?query=eag", nil)
	test.OK(t, err)
	pathwayTag = s.determinePathwayTag(r)
	test.Equals(t, api.AcnePathwayTag, pathwayTag)

}
