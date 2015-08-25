package doctor_treatment_plan

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"golang.org/x/net/context"
)

type mockDataAPI_doctorTokensHandler struct {
	api.DataAPI
	doctor    *common.Doctor
	patient   *common.Patient
	cases     []*common.PatientCase
	careTeams map[int64]*common.PatientCareTeam
}

func (m *mockDataAPI_doctorTokensHandler) GetDoctorFromAccountID(accountID int64) (*common.Doctor, error) {
	return m.doctor, nil
}
func (m *mockDataAPI_doctorTokensHandler) Patient(id common.PatientID, basicInfoOnly bool) (*common.Patient, error) {
	return m.patient, nil
}
func (m *mockDataAPI_doctorTokensHandler) GetCasesForPatient(pID common.PatientID, states []string) ([]*common.PatientCase, error) {
	return m.cases, nil
}
func (m *mockDataAPI_doctorTokensHandler) CaseCareTeams(ids []int64) (map[int64]*common.PatientCareTeam, error) {
	return m.careTeams, nil
}

func TestDoctorTokensHandler(t *testing.T) {
	m := &mockDataAPI_doctorTokensHandler{
		doctor: &common.Doctor{
			ID: encoding.DeprecatedNewObjectID(10),
		},
		patient: &common.Patient{
			ID: common.NewPatientID(uint64(20)),
		},
		cases: []*common.PatientCase{
			{
				ID: encoding.DeprecatedNewObjectID(30),
			},
		},
		careTeams: map[int64]*common.PatientCareTeam{
			30: {
				Assignments: []*common.CareProviderAssignment{
					{
						ProviderID:   10,
						ProviderRole: api.RoleDoctor,
					},
				},
			},
		},
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "http://test/local?patient_id=10", nil)
	if err != nil {
		t.Fatal(err)
	}

	h := NewDoctorTokensHandler(m)
	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{ID: 1, Role: api.RoleDoctor})
	h.ServeHTTP(ctx, w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected code %d but got %d", http.StatusOK, w.Code)
	}
	res := doctorTokensResponse{}
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}

	if len(res.Tokens) != 4 {
		t.Fatalf("Expected 4 tokens but got %d", len(res.Tokens))
	}

	// ensure that all tokens are present
	tokenizer := newTokenizerForValidation('{', '}')
	for tType := range tokenizer.tokens {
		var tokenFound bool
		for _, resTokenItem := range res.Tokens {
			if resTokenItem.Token == (string(tokenizer.startDelimiter) + string(tType) + string(tokenizer.endDelimiter)) {
				tokenFound = true
				break
			}
		}
		if !tokenFound {
			t.Fatalf("%s token expected to exist in response but didnt", tType)
		}
	}
}
