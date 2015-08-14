package patient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/apiservice"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/golang.org/x/net/context"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/ratelimit"
	taggingTest "github.com/sprucehealth/backend/tagging/test"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPI_PatientVisitHandler struct {
	api.DataAPI
	visit            *common.PatientVisit
	cases            []*common.PatientCase
	sku              *common.SKU
	pathway          *common.Pathway
	visits           []*common.PatientVisit
	patient          *common.Patient
	visitUpdate      *api.PatientVisitUpdate
	caseUpdate       *api.PatientCaseUpdate
	doctorInCareTeam *common.CareProviderAssignment
	patientLayout    *api.LayoutVersion

	doctorIDAdded int64

	createVisitFunc         func(visit *common.PatientVisit) (int64, error)
	updateAccountCreditFunc func(accountID int64, credit int, currency string) error
}

func (m *mockDataAPI_PatientVisitHandler) GetPatientVisitFromID(id int64) (*common.PatientVisit, error) {
	return m.visit, nil
}
func (m *mockDataAPI_PatientVisitHandler) UpdatePatientCase(id int64, update *api.PatientCaseUpdate) error {
	m.caseUpdate = update
	return nil
}
func (m *mockDataAPI_PatientVisitHandler) UpdatePatientVisit(id int64, update *api.PatientVisitUpdate) (int, error) {
	m.visitUpdate = update
	return 0, nil
}
func (m *mockDataAPI_PatientVisitHandler) GetPatientIDFromAccountID(accountID int64) (common.PatientID, error) {
	if m.patient != nil {
		return m.patient.ID, nil
	}
	return common.PatientID{}, nil
}
func (m *mockDataAPI_PatientVisitHandler) GetPatientFromAccountID(accountID int64) (*common.Patient, error) {
	return m.patient, nil
}
func (m *mockDataAPI_PatientVisitHandler) CasesForPathway(patientID common.PatientID, pathwayTag string, states []string) ([]*common.PatientCase, error) {
	return m.cases, nil
}
func (m *mockDataAPI_PatientVisitHandler) GetVisitsForCase(patientCaseID int64, states []string) ([]*common.PatientVisit, error) {
	return m.visits, nil
}
func (m *mockDataAPI_PatientVisitHandler) PathwayForTag(tag string, opts api.PathwayOption) (*common.Pathway, error) {
	return m.pathway, nil
}
func (m *mockDataAPI_PatientVisitHandler) SKUForPathway(tag string, category common.SKUCategoryType) (*common.SKU, error) {
	return m.sku, nil
}
func (m *mockDataAPI_PatientVisitHandler) IntakeLayoutVersionIDForAppVersion(appVersion *common.Version, platform common.Platform, pathwayID, languageID int64, skuType string) (int64, error) {
	return 0, nil
}
func (m *mockDataAPI_PatientVisitHandler) CreatePatientVisit(visit *common.PatientVisit, requestedDoctorID *int64) (int64, error) {
	if m.createVisitFunc != nil {
		return m.createVisitFunc(visit)
	}

	return 0, nil
}
func (m *mockDataAPI_PatientVisitHandler) AddDoctorToPatientCase(doctorID, caseID int64) error {
	m.doctorIDAdded = doctorID
	return nil
}
func (m *mockDataAPI_PatientVisitHandler) GetActiveCareTeamMemberForCase(role string, patientCaseID int64) (*common.CareProviderAssignment, error) {
	return m.doctorInCareTeam, nil
}
func (m *mockDataAPI_PatientVisitHandler) GetPatientLayout(versionID, languageID int64) (*api.LayoutVersion, error) {
	return m.patientLayout, nil
}
func (m *mockDataAPI_PatientVisitHandler) PreviousPatientAnswersForQuestions(questionTags []string, patientID common.PatientID, beforeTime time.Time) (map[string][]common.Answer, error) {
	return nil, nil
}
func (m *mockDataAPI_PatientVisitHandler) PatientPhotoSectionsForQuestionIDs(questionIDs []int64, patientID common.PatientID, patientVisitID int64) (map[int64][]common.Answer, error) {
	return nil, nil
}
func (m *mockDataAPI_PatientVisitHandler) AnswersForQuestions(questionIDs []int64, info api.IntakeInfo) (map[int64][]common.Answer, error) {
	return nil, nil
}
func (m *mockDataAPI_PatientVisitHandler) RegisterPatient(*common.Patient) error {
	return nil
}
func (m *mockDataAPI_PatientVisitHandler) TrackPatientAgreements(patientID common.PatientID, agreements map[string]bool) error {
	return nil
}
func (m *mockDataAPI_PatientVisitHandler) ParkedAccount(email string) (*common.ParkedAccount, error) {
	return nil, api.ErrNotFound("parked_account")
}
func (m *mockDataAPI_PatientVisitHandler) State(stateCode string) (string, string, error) {
	return "", "", nil
}
func (m *mockDataAPI_PatientVisitHandler) UpdateCredit(accountID int64, credit int, currency string) error {
	if m.updateAccountCreditFunc != nil {
		return m.updateAccountCreditFunc(accountID, credit, currency)
	}

	return nil
}
func (m *mockDataAPI_PatientVisitHandler) GetMessageForPatientVisit(id int64) (string, error) {
	return "", nil
}

type mockAuthAPI_PatientVisitHandler struct {
	api.AuthAPI
	account *common.Account
	token   string
}

func (m *mockAuthAPI_PatientVisitHandler) CreateToken(id int64, platform api.Platform, opt api.CreateTokenOption) (string, error) {
	return m.token, nil
}
func (m *mockAuthAPI_PatientVisitHandler) CreateAccount(email, password, role string) (int64, error) {
	return m.account.ID, nil
}

// This test is to ensure that in the event of a successful
// call to abandon a visit, the appropriate objects are updated
// with the appropriate state.
func TestAbandonVisit_Successful(t *testing.T) {
	m := &mockDataAPI_PatientVisitHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusOpen,
		},
	}

	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "api.spruce.local/visit?patient_visit_id=1", nil)
	test.OK(t, err)

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RolePatient})

	h := NewPatientVisitHandler(m, nil, nil, nil, "", "", nil, nil, time.Duration(0), &taggingTest.TestTaggingClient{})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)

	test.Equals(t, true, m.caseUpdate != nil)
	test.Equals(t, common.PCStatusDeleted, *m.caseUpdate.Status)
	test.Equals(t, true, m.visitUpdate != nil)
	test.Equals(t, common.PVStatusDeleted, *m.visitUpdate.Status)
}

func TestAbandonVisit_Idempotent(t *testing.T) {
	m := &mockDataAPI_PatientVisitHandler{
		visit: &common.PatientVisit{
			Status: common.PVStatusDeleted,
		},
	}

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RolePatient})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "api.spruce.local/case?patient_visit_id=1", nil)
	test.OK(t, err)

	h := NewPatientVisitHandler(m, nil, nil, nil, "", "", nil, nil, time.Duration(0), &taggingTest.TestTaggingClient{})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusOK, w.Code)
}

// This test is to ensure that when no doctor is picked at time to of visit creation
// the response does not contain a doctorID
func TestCreateVisit_FirstAvailable(t *testing.T) {
	intakeData, err := json.Marshal(&info_intake.InfoIntakeLayout{})
	test.OK(t, err)

	visitID := int64(123)
	caseID := int64(456)

	m := &mockDataAPI_PatientVisitHandler{
		patient: &common.Patient{
			ID: common.NewPatientID(123),
		},
		pathway: &common.Pathway{
			Tag:     api.AcnePathwayTag,
			Details: &common.PathwayDetails{},
		},
		sku: &common.SKU{
			Type: "visit",
		},
		patientLayout: &api.LayoutVersion{
			Layout: intakeData,
		},
		createVisitFunc: func(visit *common.PatientVisit) (int64, error) {
			visit.ID = encoding.DeprecatedNewObjectID(visitID)
			visit.PatientCaseID = encoding.DeprecatedNewObjectID(caseID)
			return visitID, nil
		},
	}

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RolePatient})

	h := NewPatientVisitHandler(m, nil, nil, nil, "", "", &dispatch.Dispatcher{}, nil, time.Duration(0), &taggingTest.TestTaggingClient{})
	w := httptest.NewRecorder()
	jsonData, err := json.Marshal(&PatientVisitRequestData{})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "api.spruce.local", bytes.NewBuffer(jsonData))
	test.OK(t, err)

	h.ServeHTTP(ctx, w, r)

	test.Equals(t, http.StatusOK, w.Code)

	var res PatientVisitResponse
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &res))
	test.Equals(t, visitID, res.PatientVisitID)
	test.Equals(t, true, res.DoctorID == 0)
}

func TestCreateVisit_DoctorPicked(t *testing.T) {
	intakeData, err := json.Marshal(&info_intake.InfoIntakeLayout{})
	test.OK(t, err)

	visitID := int64(123)
	caseID := int64(456)
	doctorID := int64(24)

	m := &mockDataAPI_PatientVisitHandler{
		patient: &common.Patient{
			ID: common.NewPatientID(123),
		},
		pathway: &common.Pathway{
			Tag:     api.AcnePathwayTag,
			Details: &common.PathwayDetails{},
		},
		sku: &common.SKU{
			Type: "visit",
		},
		doctorInCareTeam: &common.CareProviderAssignment{
			ProviderID: doctorID,
		},
		patientLayout: &api.LayoutVersion{
			Layout: intakeData,
		},
		createVisitFunc: func(visit *common.PatientVisit) (int64, error) {
			visit.ID = encoding.DeprecatedNewObjectID(visitID)
			visit.PatientCaseID = encoding.DeprecatedNewObjectID(caseID)
			return visitID, nil
		},
	}

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RolePatient})

	h := NewPatientVisitHandler(m, nil, nil, nil, "", "", &dispatch.Dispatcher{}, nil, time.Duration(0), &taggingTest.TestTaggingClient{})
	w := httptest.NewRecorder()
	jsonData, err := json.Marshal(&PatientVisitRequestData{})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "api.spruce.local", bytes.NewBuffer(jsonData))
	test.OK(t, err)

	h.ServeHTTP(ctx, w, r)

	test.Equals(t, http.StatusOK, w.Code)

	var res PatientVisitResponse
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &res))
	test.Equals(t, visitID, res.PatientVisitID)
	test.Equals(t, doctorID, res.DoctorID)
}

func TestCreatePatient_DoctorPicked(t *testing.T) {
	intakeData, err := json.Marshal(&info_intake.InfoIntakeLayout{})
	test.OK(t, err)

	visitID := int64(123)
	caseID := int64(456)
	doctorID := int64(24)

	m := &mockDataAPI_PatientVisitHandler{
		patient: &common.Patient{
			ID: common.NewPatientID(123),
		},
		pathway: &common.Pathway{
			Tag:     api.AcnePathwayTag,
			Details: &common.PathwayDetails{},
		},
		sku: &common.SKU{
			Type: "visit",
		},
		doctorInCareTeam: &common.CareProviderAssignment{
			ProviderID: doctorID,
		},
		patientLayout: &api.LayoutVersion{
			Layout: intakeData,
		},
		createVisitFunc: func(visit *common.PatientVisit) (int64, error) {
			visit.ID = encoding.DeprecatedNewObjectID(visitID)
			visit.PatientCaseID = encoding.DeprecatedNewObjectID(caseID)
			return visitID, nil
		},
	}

	mAuth := &mockAuthAPI_PatientVisitHandler{
		account: &common.Account{
			ID: 10,
		},
		token: "token",
	}

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RolePatient})

	h := NewSignupHandler(m, mAuth, "", "", nil, &dispatch.Dispatcher{}, time.Duration(0), nil, &ratelimit.NullKeyed{}, nil, metrics.NewRegistry())
	w := httptest.NewRecorder()

	jsonData, err := json.Marshal(&SignupPatientRequestData{
		Email:       "test@test.com",
		Password:    "12345",
		FirstName:   "test",
		LastName:    "test",
		DOB:         "1987-11-08",
		Gender:      "male",
		ZipCode:     "94115",
		DoctorID:    doctorID,
		Phone:       "7341234567",
		Agreements:  "tos",
		StateCode:   "CA",
		CreateVisit: true,
		PathwayTag:  api.AcnePathwayTag,
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "api.spruce.local", bytes.NewBuffer(jsonData))
	test.OK(t, err)
	r.Header.Set("Content-Type", "application/json")

	h.ServeHTTP(ctx, w, r)

	test.Equals(t, http.StatusOK, w.Code)

	var res SignedupResponse
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &res))
	test.Equals(t, visitID, res.PatientVisitData.PatientVisitID)
	test.Equals(t, doctorID, res.PatientVisitData.DoctorID)
}

func TestCreatePatient_FirstAvailable(t *testing.T) {
	intakeData, err := json.Marshal(&info_intake.InfoIntakeLayout{})
	test.OK(t, err)

	visitID := int64(123)
	caseID := int64(456)

	m := &mockDataAPI_PatientVisitHandler{
		patient: &common.Patient{
			ID: common.NewPatientID(123),
		},
		pathway: &common.Pathway{
			Tag:     api.AcnePathwayTag,
			Details: &common.PathwayDetails{},
		},
		sku: &common.SKU{
			Type: "visit",
		},
		patientLayout: &api.LayoutVersion{
			Layout: intakeData,
		},
		createVisitFunc: func(visit *common.PatientVisit) (int64, error) {
			visit.ID = encoding.DeprecatedNewObjectID(visitID)
			visit.PatientCaseID = encoding.DeprecatedNewObjectID(caseID)
			return visitID, nil
		},
	}

	mAuth := &mockAuthAPI_PatientVisitHandler{
		account: &common.Account{
			ID: 10,
		},
		token: "token",
	}

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RolePatient})

	h := NewSignupHandler(m, mAuth, "", "", nil, &dispatch.Dispatcher{}, time.Duration(0), nil, &ratelimit.NullKeyed{}, nil, metrics.NewRegistry())
	w := httptest.NewRecorder()

	jsonData, err := json.Marshal(&SignupPatientRequestData{
		Email:       "test@test.com",
		Password:    "12345",
		FirstName:   "test",
		LastName:    "test",
		DOB:         "1987-11-08",
		Gender:      "male",
		ZipCode:     "94115",
		Phone:       "7341234567",
		Agreements:  "tos",
		StateCode:   "CA",
		CreateVisit: true,
		PathwayTag:  api.AcnePathwayTag,
	})
	test.OK(t, err)
	r, err := http.NewRequest("POST", "api.spruce.local", bytes.NewBuffer(jsonData))
	test.OK(t, err)
	r.Header.Set("Content-Type", "application/json")

	h.ServeHTTP(ctx, w, r)

	test.Equals(t, http.StatusOK, w.Code)

	var res SignedupResponse
	test.OK(t, json.Unmarshal(w.Body.Bytes(), &res))
	test.Equals(t, visitID, res.PatientVisitData.PatientVisitID)
	test.Equals(t, true, res.PatientVisitData.DoctorID == 0)
}

// This test is to ensure that deletion/abandonment of a case in any state other
// than open or deleted is forbidden
func TestAbandonCase_Forbidden(t *testing.T) {
	testForbiddenDelete(t, common.PVStatusRouted)
	testForbiddenDelete(t, common.PVStatusSubmitted)
	testForbiddenDelete(t, common.PVStatusReviewing)
	testForbiddenDelete(t, common.PVStatusTreated)
}

func testForbiddenDelete(t *testing.T, status string) {
	m := &mockDataAPI_PatientVisitHandler{
		visit: &common.PatientVisit{
			Status: status,
		},
	}

	ctx := apiservice.CtxWithAccount(context.Background(), &common.Account{Role: api.RolePatient})

	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "api.spruce.local/case?patient_visit_id=1", nil)
	test.OK(t, err)

	h := NewPatientVisitHandler(m, nil, nil, nil, "", "", nil, nil, time.Duration(0), &taggingTest.TestTaggingClient{})
	h.ServeHTTP(ctx, w, r)
	test.Equals(t, http.StatusForbidden, w.Code)
}
