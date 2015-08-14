package patient_file

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

type mockDataAPI_patientParentHandler struct {
	api.DataAPI
	consent   *common.ParentalConsent
	patient   *common.Patient
	doctor    *common.Doctor
	proof     *api.ParentalConsentProof
	cases     []*common.PatientCase
	careTeams map[int64]*common.PatientCareTeam
}

func (m *mockDataAPI_patientParentHandler) ParentalConsent(childPatientID common.PatientID) ([]*common.ParentalConsent, error) {
	return []*common.ParentalConsent{m.consent}, nil
}
func (m *mockDataAPI_patientParentHandler) GetPatientFromID(common.PatientID) (*common.Patient, error) {
	return m.patient, nil
}
func (m *mockDataAPI_patientParentHandler) ParentConsentProof(common.PatientID) (*api.ParentalConsentProof, error) {
	return m.proof, nil
}
func (m *mockDataAPI_patientParentHandler) GetDoctorFromAccountID(accountID int64) (*common.Doctor, error) {
	return m.doctor, nil
}
func (m *mockDataAPI_patientParentHandler) Patient(common.PatientID, bool) (*common.Patient, error) {
	return m.patient, nil
}
func (m *mockDataAPI_patientParentHandler) GetCasesForPatient(id common.PatientID, states []string) ([]*common.PatientCase, error) {
	return m.cases, nil
}
func (m *mockDataAPI_patientParentHandler) CaseCareTeams(caseIDs []int64) (map[int64]*common.PatientCareTeam, error) {
	return m.careTeams, nil
}

func TestPatientParentHandler(t *testing.T) {
	m := &mockDataAPI_patientParentHandler{
		consent: &common.ParentalConsent{
			Relationship: "father",
		},
		patient: &common.Patient{
			FirstName: "Joe",
			LastName:  "Schmoe",
			Email:     "joe@schmoe.com",
			DOB: encoding.Date{
				Month: 11,
				Day:   8,
				Year:  1987,
			},
			Gender: "male",
			PhoneNumbers: []*common.PhoneNumber{
				{
					Phone: common.Phone("206-877-3590"),
				},
			},
		},
		doctor: &common.Doctor{
			ID: encoding.DeprecatedNewObjectID(2),
		},
		cases: []*common.PatientCase{
			{
				ID: encoding.DeprecatedNewObjectID(1),
			},
		},
		careTeams: map[int64]*common.PatientCareTeam{
			1: &common.PatientCareTeam{
				Assignments: []*common.CareProviderAssignment{
					{
						ProviderRole: api.RoleDoctor,
						ProviderID:   2,
					},
				},
			},
		},
		proof: &api.ParentalConsentProof{
			GovernmentIDPhotoID: ptr.Int64(1),
			SelfiePhotoID:       ptr.Int64(2),
		},
	}

	signer, err := sig.NewSigner([][]byte{[]byte("key")}, nil)
	test.OK(t, err)

	ms := media.NewStore("https://test.com", signer, nil)
	dur := 5 * time.Minute
	h := NewPatientParentHandler(m, ms, dur)

	w := httptest.NewRecorder()

	r, err := http.NewRequest("GET", "http://test.com?patient_id=2", nil)
	test.OK(t, err)

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleDoctor})
	h.ServeHTTP(ctx, w, r)

	var res patientParentResponse
	test.OK(t, json.NewDecoder(w.Body).Decode(&res))
	test.Equals(t, 1, len(res.Parents))
	test.Equals(t, "Joe", res.Parents[0].FirstName)
	test.Equals(t, "Schmoe", res.Parents[0].LastName)
	test.Equals(t, "1987-11-08", res.Parents[0].DOB)
	test.Equals(t, "Male", res.Parents[0].Gender)
	test.Equals(t, "father", res.Parents[0].Relationship)
	test.Equals(t, "joe@schmoe.com", res.Parents[0].Email)
	test.Equals(t, "206-877-3590", res.Parents[0].CellPhone)
	test.Equals(t, true, res.Parents[0].Proof.SelfiePhotoURL != "")
	test.Equals(t, true, res.Parents[0].Proof.GovernmentIDPhotoURL != "")
}
