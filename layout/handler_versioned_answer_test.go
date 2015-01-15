package layout

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
)

type mockedDataAPI_versionedAnswerHandler struct {
	api.DataAPI
	va    *common.VersionedAnswer
	vaTag *common.VersionedAnswer
}

func (m mockedDataAPI_versionedAnswerHandler) VersionedAnswerFromID(ID int64) (*common.VersionedAnswer, error) {
	return m.va, nil
}

func (m mockedDataAPI_versionedAnswerHandler) VersionedAnswer(questionTag string, questionID, languageID, version int64) (*common.VersionedAnswer, error) {
	return m.vaTag, nil
}

func TestAnswerHandlerDoctorCannotQuery(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	careTeamHandler := NewVersionedAnswerHandler(mockedDataAPI_versionedAnswerHandler{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.DOCTOR_ROLE
			ctxt.AccountID = 1
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteError(apiservice.NewAccessForbiddenError(), expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestAnswerHandlerPatientCannotQuery(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	careTeamHandler := NewVersionedAnswerHandler(mockedDataAPI_versionedAnswerHandler{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.PATIENT_ROLE
			ctxt.AccountID = 1
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteError(apiservice.NewAccessForbiddenError(), expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestAnswerHandlerRequiresParams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request", nil)
	test.OK(t, err)
	careTeamHandler := NewVersionedAnswerHandler(mockedDataAPI_versionedAnswerHandler{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
			ctxt.AccountID = 1
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteError(apiservice.NewValidationError("insufficent parameters supplied to form complete query"), expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestAnswerHandlerRequiresCompleteTagQuery(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?tag=my_tag&question_id=1", nil)
	test.OK(t, err)
	careTeamHandler := NewVersionedAnswerHandler(mockedDataAPI_versionedAnswerHandler{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
			ctxt.AccountID = 1
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteError(apiservice.NewValidationError("insufficent parameters supplied to form complete query"), expectedWriter, r)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestAnswerHandlerCanQueryByID(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?id=1", nil)
	test.OK(t, err)
	dbmodel := buildDummyVersionedAnswer("dummy")
	careTeamHandler := NewVersionedAnswerHandler(mockedDataAPI_versionedAnswerHandler{DataAPI: &api.DataService{}, va: dbmodel})
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
			ctxt.AccountID = 1
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteJSON(expectedWriter, &versionedAnswerGETResponse{VersionedAnswer: responses.NewVersionedAnswerFromDBModel(dbmodel)})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestAnswerHandlerCanQueryByTagSet(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?tag=my_tag&language_id=1&question_id=1", nil)
	test.OK(t, err)
	dbmodel := buildDummyVersionedAnswer("dummy2")
	careTeamHandler := NewVersionedAnswerHandler(mockedDataAPI_versionedAnswerHandler{DataAPI: &api.DataService{}, vaTag: dbmodel})
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
			ctxt.AccountID = 1
		},
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteJSON(expectedWriter, &versionedAnswerGETResponse{VersionedAnswer: responses.NewVersionedAnswerFromDBModel(dbmodel)})
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func buildDummyVersionedAnswer(answerText string) *common.VersionedAnswer {
	return &common.VersionedAnswer{
		ID:           1,
		AnswerTypeID: 1,
		AnswerTag:    answerText,
		ToAlert: sql.NullBool{
			Bool:  true,
			Valid: true,
		},
		Ordering:   1,
		QuestionID: 1,
		LanguageID: 1,
		Version:    1,
		AnswerText: sql.NullString{
			String: answerText,
			Valid:  true,
		},
		AnswerSummaryText: sql.NullString{
			String: answerText,
			Valid:  true,
		},
		AnswerType: answerText,
	}
}
