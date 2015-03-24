package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
)

type mockedDataAPI_handlerCaseVisit struct {
	api.DataAPI
	Summaries []*common.VisitSummary
}

func (d mockedDataAPI_handlerCaseVisit) VisitSummaries(visitStatuses []string) ([]*common.VisitSummary, error) {
	return d.Summaries, nil
}

func TestHandlerCaseVisitStatusRequired(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	caseVisitHandler := NewCaseVisitsHandler(mockedDataAPI_handlerCaseVisit{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: caseVisitHandler,
	}
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestHandlerCaseVisitSensicalStatusRequired(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?status=BunnyFooFoo", nil)
	test.OK(t, err)
	caseVisitHandler := NewCaseVisitsHandler(mockedDataAPI_handlerCaseVisit{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: caseVisitHandler,
	}
	responseWriter := httptest.NewRecorder()
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, http.StatusBadRequest, responseWriter.Code)
}

func TestHandlerCaseVisitSuccessfulGET(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?status=uncompleted", nil)
	test.OK(t, err)
	ti := time.Time{}
	summary := &common.VisitSummary{
		VisitID:           1,
		CaseID:            1,
		CreationDate:      ti,
		SubmittedDate:     &ti,
		LockTakenDate:     nil,
		RequestedDoctorID: nil,
		DoctorID:          nil,
		RoleTypeTag:       nil,
		PathwayName:       "It's",
		PatientFirstName:  "simple.",
		PatientLastName:   "We",
		CaseName:          ", uh,",
		SKUType:           "kill",
		SubmissionState:   nil,
		Status:            "the Batman.",
		DoctorFirstName:   nil,
		DoctorLastName:    nil,
		LockType:          nil,
	}
	caseVisitHandler := NewCaseVisitsHandler(mockedDataAPI_handlerCaseVisit{DataAPI: &api.DataService{}, Summaries: []*common.VisitSummary{summary}})
	handler := test_handler.MockHandler{
		H: caseVisitHandler,
	}
	resp := caseVisitsGETResponse{
		VisitSummaries: []*responses.PHISafeVisitSummary{
			&responses.PHISafeVisitSummary{
				VisitID:         1,
				CaseID:          1,
				CreationEpoch:   summary.CreationDate.Unix(),
				SubmittedEpoch:  summary.SubmittedDate.Unix(),
				LockTakenEpoch:  0,
				DoctorID:        nil,
				FirstAvailable:  true,
				Pathway:         "It's",
				DoctorWithLock:  "",
				PatientInitials: "sW",
				CaseName:        ", uh,",
				Type:            "kill",
				SubmissionState: nil,
				Status:          "the Batman.",
				LockType:        nil,
			},
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	httputil.JSONResponse(expectedWriter, http.StatusOK, resp)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, expectedWriter.Body.String(), responseWriter.Body.String())
}
