package test_intake

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

const EN = 1

func insertQuestionVersion(questionTag, questionText, questionType string, version int64, testData *test_integration.TestData, t *testing.T) int64 {
	insertQuery :=
		`INSERT INTO question 
    (qtype_id, qtext_app_text_id, qtext_short_text_id, subtext_app_text_id, question_tag, alert_app_text_id, language_id, version, question_text, question_type)
    VALUES(1, 1, 1, 1, ?, 1, 1, ?, ?, ?)`
	res, err := testData.DB.Exec(insertQuery, questionTag, version, questionText, questionType)
	if err != nil {
		t.Fatal(err)
	}
	lID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return lID
}

func insertAnswerVersion(answerTag, answerText, answerType string, ordering, questionID int64, testData *test_integration.TestData, t *testing.T) int64 {
	insertQuery :=
		`INSERT INTO potential_answer 
    (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, language_id, answer_text, answer_type, status)
    VALUES(?, 1, 1, ?, ?, 1, ?, ?, 'ACTIVE')`
	res, err := testData.DB.Exec(insertQuery, questionID, answerTag, ordering, answerText, answerType)
	if err != nil {
		t.Fatal(err)
	}
	lID, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return lID
}

func TestVersionedQuestionDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	insertQuestionVersion("myTag", "questionText", "questionType", 1, testData, t)
	vq, err := testData.DataAPI.VersionedQuestion("myTag", EN, 1)
	test.OK(t, err)
	test.Equals(t, vq.QuestionText.String, "questionText")
	test.Equals(t, vq.QuestionType, "questionType")
	test.Equals(t, vq.Version, int64(1))
}

func TestVersionedQuestionMultipleDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	insertQuestionVersion("myTag", "questionText", "questionType", 1, testData, t)
	insertQuestionVersion("myTag", "questionText2", "questionType", 2, testData, t)
	query := []*api.QuestionQueryParams{
		&api.QuestionQueryParams{
			QuestionTag: "myTag",
			LanguageID:  EN,
			Version:     1,
		},
		&api.QuestionQueryParams{
			QuestionTag: "myTag",
			LanguageID:  EN,
			Version:     2,
		},
	}

	vqs, err := testData.DataAPI.VersionedQuestions(query)
	test.OK(t, err)
	test.Equals(t, vqs[0].QuestionText.String, "questionText")
	test.Equals(t, vqs[0].Version, int64(1))
	test.Equals(t, vqs[1].QuestionText.String, "questionText2")
	test.Equals(t, vqs[1].Version, int64(2))
}

func TestVersionedQuestionFromID(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, testData, t)

	vq, err := testData.DataAPI.VersionedQuestionFromID(qid)
	test.OK(t, err)
	test.Equals(t, vq.QuestionText.String, "questionText")
	test.Equals(t, vq.Version, int64(1))
}

func TestVersionedQuestionFromIDNoRows(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	_, err := testData.DataAPI.VersionedQuestionFromID(10000)
	test.Equals(t, api.NoRowsError, err)
}

func TestVersionQuestion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, testData, t)
	insertAnswerVersion("myTag", "answerText", "answerType", 1, qid, testData, t)
	insertAnswerVersion("myTag2", "answerText2", "answerType", 2, qid, testData, t)

	vq, err := testData.DataAPI.VersionedQuestionFromID(qid)
	test.OK(t, err)

	id, err := testData.DataAPI.VersionQuestion(vq)
	test.OK(t, err)

	vas, err := testData.DataAPI.VersionedAnswersForQuestion(id, EN)
	test.OK(t, err)
	test.Equals(t, 2, len(vas))
	test.Equals(t, vas[0].AnswerText.String, "answerText")
	test.Equals(t, vas[1].AnswerText.String, "answerText2")
}

func TestVersionQuestionNoAnswers(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, testData, t)

	vq, err := testData.DataAPI.VersionedQuestionFromID(qid)
	test.OK(t, err)

	id, err := testData.DataAPI.VersionQuestion(vq)
	test.OK(t, err)

	vas, err := testData.DataAPI.VersionedAnswersForQuestion(id, EN)
	test.OK(t, err)
	test.Equals(t, 0, len(vas))
}

//answerTag, answerText, answerType, status string, ordering, questionID, version int64
func TestVersionedAnswerDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, testData, t)
	insertAnswerVersion("myTag", "answerText", "answerType", 1, qid, testData, t)
	va, err := testData.DataAPI.VersionedAnswer("myTag", qid, EN)
	test.OK(t, err)
	test.Equals(t, va.AnswerText.String, "answerText")
	test.Equals(t, va.AnswerType, "answerType")
}

func TestVersionedAnswerMultipleDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, testData, t)
	qid2 := insertQuestionVersion("myTag", "questionText", "questionType", 2, testData, t)
	insertAnswerVersion("myTag", "answerText", "answerType", 1, qid, testData, t)
	insertAnswerVersion("myTag", "answerText2", "answerType", 1, qid2, testData, t)
	query := []*api.AnswerQueryParams{
		&api.AnswerQueryParams{
			AnswerTag:  "myTag",
			QuestionID: qid,
			LanguageID: EN,
		},
		&api.AnswerQueryParams{
			AnswerTag:  "myTag",
			QuestionID: qid2,
			LanguageID: EN,
		},
	}

	vas, err := testData.DataAPI.VersionedAnswers(query)
	test.OK(t, err)
	test.Equals(t, vas[0].AnswerText.String, "answerText")
	test.Equals(t, vas[1].AnswerText.String, "answerText2")
}

func TestVersionedAnswerFromID(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, testData, t)
	aid := insertAnswerVersion("myTag", "answerText", "answerType", 1, qid, testData, t)

	va, err := testData.DataAPI.VersionedAnswerFromID(aid)
	test.OK(t, err)
	test.Equals(t, va.AnswerText.String, "answerText")
}

func TestVersionedAnswerFromIDNoRows(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	_, err := testData.DataAPI.VersionedAnswerFromID(10000)
	test.Equals(t, api.NoRowsError, err)
}
