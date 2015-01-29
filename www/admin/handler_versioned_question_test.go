package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/responses"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_handler"
	"github.com/sprucehealth/backend/www"
)

type mockedDataAPI_versionedQuestionHandler struct {
	api.DataAPI
	vq         *common.VersionedQuestion
	vqTag      *common.VersionedQuestion
	vas        []*common.VersionedAnswer
	vaqfs      []*common.VersionedAdditionalQuestionField
	maxVersion int64
}

func (m mockedDataAPI_versionedQuestionHandler) VersionedQuestionFromID(ID int64) (*common.VersionedQuestion, error) {
	return m.vq, nil
}

func (m mockedDataAPI_versionedQuestionHandler) VersionedQuestions(questionQueryParams []*api.QuestionQueryParams) ([]*common.VersionedQuestion, error) {
	return []*common.VersionedQuestion{m.vq}, nil
}

func (m mockedDataAPI_versionedQuestionHandler) InsertVersionedQuestion(vq *common.VersionedQuestion, va []*common.VersionedAnswer, vaqf *common.VersionedAdditionalQuestionField) (int64, error) {
	return 1, nil
}

func (m mockedDataAPI_versionedQuestionHandler) VersionedAnswers([]*api.AnswerQueryParams) ([]*common.VersionedAnswer, error) {
	return m.vas, nil
}

func (m mockedDataAPI_versionedQuestionHandler) VersionedAdditionalQuestionFields(questionID, languageID int64) ([]*common.VersionedAdditionalQuestionField, error) {
	return m.vaqfs, nil
}

func (m mockedDataAPI_versionedQuestionHandler) MaxQuestionVersion(questionTag string, languageID int64) (int64, error) {
	return m.maxVersion, nil
}

func TestQuestionHandlerRequiresParams(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?language_id=1", nil)
	test.OK(t, err)
	questionHandler := NewVersionedQuestionHandler(mockedDataAPI_versionedQuestionHandler{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: questionHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.APIBadRequestError(expectedWriter, r, fmt.Errorf("insufficent parameters supplied to form complete query").Error())
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestQuestionHandlerRequiresCompleteTagQuery(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?language_id=1", nil)
	test.OK(t, err)
	questionHandler := NewVersionedQuestionHandler(mockedDataAPI_versionedQuestionHandler{DataAPI: &api.DataService{}})
	handler := test_handler.MockHandler{
		H: questionHandler,
	}
	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.APIBadRequestError(expectedWriter, r, fmt.Errorf("insufficent parameters supplied to form complete query").Error())
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestQuestionHandlerCanQueryByID(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?id=1&language_id=1", nil)
	test.OK(t, err)
	dbmodel := buildDummyVersionedQuestion("dummy")
	questionHandler := NewVersionedQuestionHandler(mockedDataAPI_versionedQuestionHandler{DataAPI: &api.DataService{}, vq: dbmodel, vaqfs: []*common.VersionedAdditionalQuestionField{}})
	handler := test_handler.MockHandler{
		H: questionHandler,
	}

	response := versionedQuestionGETResponse{
		VersionedQuestion: responses.NewVersionedQuestionFromDBModel(dbmodel),
	}
	response.VersionedQuestion.VersionedAnswers = []*responses.VersionedAnswer{}
	response.VersionedQuestion.VersionedAdditionalQuestionFields = &responses.VersionedAdditionalQuestionFields{}

	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.JSONResponse(expectedWriter, r, http.StatusOK, response)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
	test.Equals(t, http.StatusOK, responseWriter.Code)
}

func TestQuestionHandlerCanQueryByTagSet(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?tag=my_tag&version=1&language_id=1", nil)
	test.OK(t, err)
	dbmodel := buildDummyVersionedQuestion("dummy2")
	va := buildDummyVersionedAnswer("answer")
	questionHandler := NewVersionedQuestionHandler(mockedDataAPI_versionedQuestionHandler{DataAPI: &api.DataService{}, vq: dbmodel, vas: []*common.VersionedAnswer{va}, vaqfs: []*common.VersionedAdditionalQuestionField{}})
	handler := test_handler.MockHandler{
		H: questionHandler,
	}

	response := versionedQuestionGETResponse{
		VersionedQuestion: responses.NewVersionedQuestionFromDBModel(dbmodel),
	}
	response.VersionedQuestion.VersionedAnswers = []*responses.VersionedAnswer{responses.NewVersionedAnswerFromDBModel(va)}
	response.VersionedQuestion.VersionedAdditionalQuestionFields = &responses.VersionedAdditionalQuestionFields{}

	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.JSONResponse(expectedWriter, r, http.StatusOK, response)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func TestQuestionHandlerCanQueryByTagSetNoVersion(t *testing.T) {
	r, err := http.NewRequest("GET", "mock.api.request?tag=my_tag&language_id=1", nil)
	test.OK(t, err)
	dbmodel := buildDummyVersionedQuestion("dummy2")
	dbmodel.Version = 99
	va := buildDummyVersionedAnswer("answer")
	questionHandler := NewVersionedQuestionHandler(mockedDataAPI_versionedQuestionHandler{DataAPI: &api.DataService{}, vq: dbmodel, vas: []*common.VersionedAnswer{va}, vaqfs: []*common.VersionedAdditionalQuestionField{}, maxVersion: dbmodel.Version})
	handler := test_handler.MockHandler{
		H: questionHandler,
	}

	response := versionedQuestionGETResponse{
		VersionedQuestion: responses.NewVersionedQuestionFromDBModel(dbmodel),
	}
	response.VersionedQuestion.VersionedAnswers = []*responses.VersionedAnswer{responses.NewVersionedAnswerFromDBModel(va)}
	response.VersionedQuestion.VersionedAdditionalQuestionFields = &responses.VersionedAdditionalQuestionFields{}

	expectedWriter, responseWriter := httptest.NewRecorder(), httptest.NewRecorder()
	www.JSONResponse(expectedWriter, r, http.StatusOK, response)
	handler.ServeHTTP(responseWriter, r)
	test.Equals(t, string(expectedWriter.Body.Bytes()), string(responseWriter.Body.Bytes()))
}

func buildQuestionsPOSTRequest(t *testing.T, questionTag string, answerTags ...string) *http.Request {
	requestBody := &versionedQuestionPOSTRequest{
		LanguageID: 1,
		Tag:        questionTag,
		Type:       questionTag,
		Text:       questionTag,
	}
	ordering := int64(0)
	requestBody.VersionedAnswers = make([]*versionedAnswerPOSTRequest, len(answerTags))
	for i, t := range answerTags {
		requestBody.VersionedAnswers[i] = &versionedAnswerPOSTRequest{
			Tag:        t,
			LanguageID: 1,
			Type:       t,
			Status:     "ACTIVE",
			Ordering:   ordering,
		}
		ordering++
	}

	jsonData, err := json.Marshal(requestBody)
	test.OK(t, err)

	r, err := http.NewRequest("POST", "mock.api.request", bytes.NewBuffer(jsonData))
	test.OK(t, err)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
	return r
}

func buildDummyVersionedAnswer(answerText string) *common.VersionedAnswer {
	return &common.VersionedAnswer{
		ID:                1,
		AnswerTag:         answerText,
		ToAlert:           false,
		Ordering:          1,
		QuestionID:        1,
		LanguageID:        1,
		AnswerText:        answerText,
		AnswerSummaryText: answerText,
		AnswerType:        answerText,
	}
}

func buildDummyVersionedAdditionalQuestionField(answerText string) *common.VersionedAdditionalQuestionField {
	return &common.VersionedAdditionalQuestionField{
		ID:         1,
		QuestionID: 1,
		JSON:       []byte(answerText),
		LanguageID: 1,
	}
}

func buildDummyVersionedQuestion(questionText string) *common.VersionedQuestion {
	return &common.VersionedQuestion{
		ID:                 1,
		QuestionTag:        questionText,
		ParentQuestionID:   nil,
		Required:           false,
		FormattedFieldTags: ``,
		ToAlert:            false,
		TextHasTokens:      false,
		LanguageID:         1,
		Version:            1,
		QuestionText:       questionText,
		SubtextText:        questionText,
		SummaryText:        questionText,
		AlertText:          questionText,
		QuestionType:       questionText,
	}
}
