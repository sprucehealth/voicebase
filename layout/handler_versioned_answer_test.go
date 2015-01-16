package layout

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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
	vq    *common.VersionedQuestion
}

func (m mockedDataAPI_versionedAnswerHandler) VersionedAnswerFromID(ID int64) (*common.VersionedAnswer, error) {
	return m.va, nil
}

func (m mockedDataAPI_versionedAnswerHandler) VersionedQuestionFromID(ID int64) (*common.VersionedQuestion, error) {
	return m.vq, nil
}

func (m mockedDataAPI_versionedAnswerHandler) VersionedAnswer(questionTag string, questionID, languageID int64) (*common.VersionedAnswer, error) {
	return m.vaTag, nil
}

func (m mockedDataAPI_versionedAnswerHandler) VersionAnswer(va *common.VersionedAnswer) (int64, int64, error) {
	return m.vq.ID, m.va.ID, nil
}

func (m mockedDataAPI_versionedAnswerHandler) DeleteVersionedAnswer(va *common.VersionedAnswer) (int64, error) {
	return m.va.ID, nil
}

func (m mockedDataAPI_versionedAnswerHandler) VersionedAnswersForQuestion(questionID, languageID int64) ([]*common.VersionedAnswer, error) {
	return []*common.VersionedAnswer{m.va}, nil
}

func TestAnswerHandlerDoctorCannotQuery(t *testing.T) {
	r, err := http.NewRequest("POST", "mock.api.request", nil)
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
	r, err := http.NewRequest("POST", "mock.api.request", nil)
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

func TestAnswerHandlerCanInsertNewAnswers(t *testing.T) {
	r := buildAnswersPOSTRequest(t, 1, `type`, `tag`)
	dbmodel := buildDummyVersionedAnswer("dummy")
	vq := buildDummyVersionedQuestion("text")
	careTeamHandler := NewVersionedAnswerHandler(mockedDataAPI_versionedAnswerHandler{DataAPI: &api.DataService{}, va: dbmodel, vq: vq})
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
			ctxt.AccountID = 1
		},
	}
	response := versionedAnswerPOSTResponse{
		VersionedQuestion: responses.NewVersionedQuestionFromDBModel(vq),
		VersionedAnswerID: dbmodel.ID,
	}
	response.VersionedQuestion.VersionedAnswers = []*responses.VersionedAnswer{responses.NewVersionedAnswerFromDBModel(dbmodel)}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteJSON(expectedWriter, response)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestAnswerHandlerCanDeleteExistingAnswer(t *testing.T) {
	r, err := http.NewRequest("DELETE", "mock.api.request?tag=my_tag&language_id=1&question_id=1", nil)
	test.OK(t, err)
	dbmodel := buildDummyVersionedAnswer("dummy")
	vq := buildDummyVersionedQuestion("text")
	careTeamHandler := NewVersionedAnswerHandler(mockedDataAPI_versionedAnswerHandler{DataAPI: &api.DataService{}, va: dbmodel, vq: vq})
	handler := test_handler.MockHandler{
		H: careTeamHandler,
		Setup: func() {
			ctxt := apiservice.GetContext(r)
			ctxt.Role = api.ADMIN_ROLE
			ctxt.AccountID = 1
		},
	}
	response := versionedAnswerDELETEResponse{
		VersionedQuestion: responses.NewVersionedQuestionFromDBModel(vq),
	}
	response.VersionedQuestion.VersionedAnswers = []*responses.VersionedAnswer{responses.NewVersionedAnswerFromDBModel(dbmodel)}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	apiservice.WriteJSON(expectedWriter, response)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func buildAnswersPOSTRequest(t *testing.T, questionID int, questionType, questionTag string) *http.Request {
	vals := url.Values{}
	vals.Set("language_id", strconv.Itoa(1))
	vals.Set("tag", questionTag)
	vals.Set("type", questionType)
	vals.Set("ordering", strconv.Itoa(1))
	vals.Set("question_id", strconv.Itoa(questionID))

	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewBufferString(vals.Encode()))
	test.OK(t, err)
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(vals.Encode())))
	return r
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
