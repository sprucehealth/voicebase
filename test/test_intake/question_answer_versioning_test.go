package test_intake

import (
	"testing"

	"github.com/sprucehealth/backend/common"
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

func insertAnswerVersion(answerTag, answerText, answerType string, ordering, questionID, version int64, testData *test_integration.TestData, t *testing.T) int64 {
	insertQuery :=
		`INSERT INTO potential_answer 
    (question_id, answer_localized_text_id, atype_id, potential_answer_tag, ordering, language_id, version, answer_text, answer_type, status)
    VALUES(?, 1, 1, ?, ?, 1, ?, ?, ?, 'ACTIVE')`
	res, err := testData.DB.Exec(insertQuery, questionID, answerTag, ordering, version, answerText, answerType)
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
	query := []*common.QuestionQueryParams{
		&common.QuestionQueryParams{
			QuestionTag: "myTag",
			LanguageID:  EN,
			Version:     1,
		},
		&common.QuestionQueryParams{
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

//answerTag, answerText, answerType, status string, ordering, questionID, version int64
func TestVersionedAnswerDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, testData, t)
	insertAnswerVersion("myTag", "answerText", "answerType", 1, qid, 1, testData, t)
	va, err := testData.DataAPI.VersionedAnswer("myTag", qid, EN, 1)
	test.OK(t, err)
	test.Equals(t, va.AnswerText.String, "answerText")
	test.Equals(t, va.AnswerType, "answerType")
	test.Equals(t, va.Version, int64(1))
}

func TestVersionedAnswerMultipleDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, testData, t)
	insertAnswerVersion("myTag", "answerText", "answerType", 1, qid, 1, testData, t)
	insertAnswerVersion("myTag", "answerText2", "answerType", 1, qid, 2, testData, t)
	query := []*common.AnswerQueryParams{
		&common.AnswerQueryParams{
			AnswerTag:  "myTag",
			QuestionID: qid,
			LanguageID: EN,
			Version:    1,
		},
		&common.AnswerQueryParams{
			AnswerTag:  "myTag",
			QuestionID: qid,
			LanguageID: EN,
			Version:    2,
		},
	}

	vas, err := testData.DataAPI.VersionedAnswers(query)
	test.OK(t, err)
	test.Equals(t, vas[0].AnswerText.String, "answerText")
	test.Equals(t, vas[0].Version, int64(1))
	test.Equals(t, vas[1].AnswerText.String, "answerText2")
	test.Equals(t, vas[1].Version, int64(2))
}
