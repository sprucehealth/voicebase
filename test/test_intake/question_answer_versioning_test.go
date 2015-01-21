package test_intake

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

const EN = 1

func insertQuestionVersion(questionTag, questionText, questionType string, version int64, parentQuestionID *int64, testData *test_integration.TestData, t *testing.T) int64 {
	insertQuery :=
		`INSERT INTO question 
    (qtext_app_text_id, qtext_short_text_id, subtext_app_text_id, question_tag, alert_app_text_id, language_id, version, question_text, question_type, parent_question_id)
    VALUES(1, 1, 1, ?, 1, 1, ?, ?, ?, ?)`
	res, err := testData.DB.Exec(insertQuery, questionTag, version, questionText, questionType, parentQuestionID)
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
    (question_id, answer_localized_text_id, potential_answer_tag, ordering, language_id, answer_text, answer_type, status)
    VALUES(?, 1, ?, ?, 1, ?, ?, 'ACTIVE')`
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

func insertAdditionalQuestionFields(questionID, languageID int64, blobText string, testData *test_integration.TestData, t *testing.T) int64 {
	insertQuery :=
		`INSERT INTO additional_question_fields (question_id, json, language_id) VALUES(?, CAST(? AS BINARY), ?)`
	res, err := testData.DB.Exec(insertQuery, questionID, blobText, languageID)
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
	insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)
	vqs, err := testData.DataAPI.VersionedQuestions([]*api.QuestionQueryParams{&api.QuestionQueryParams{QuestionTag: "myTag", LanguageID: EN, Version: 1}})
	test.OK(t, err)
	test.Equals(t, vqs[0].QuestionText, "questionText")
	test.Equals(t, vqs[0].QuestionType, "questionType")
	test.Equals(t, vqs[0].Version, int64(1))
}

func TestVersionedQuestionMultipleDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)
	insertQuestionVersion("myTag", "questionText2", "questionType", 2, nil, testData, t)
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
	test.Equals(t, vqs[0].QuestionText, "questionText")
	test.Equals(t, vqs[0].Version, int64(1))
	test.Equals(t, vqs[1].QuestionText, "questionText2")
	test.Equals(t, vqs[1].Version, int64(2))
}

func TestVersionedQuestionFromID(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)

	vq, err := testData.DataAPI.VersionedQuestionFromID(qid)
	test.OK(t, err)
	test.Equals(t, vq.QuestionText, "questionText")
	test.Equals(t, vq.Version, int64(1))
}

func TestVersionedQuestionFromIDNoRows(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	_, err := testData.DataAPI.VersionedQuestionFromID(10000)
	test.Equals(t, api.NoRowsError, err)
}

func TestInsertVersionedQuestion(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)
	aid1 := insertAnswerVersion("myTag", "answerText", "answerType", 1, qid, testData, t)
	aid2 := insertAnswerVersion("myTag2", "answerText2", "answerType", 2, qid, testData, t)

	vq, err := testData.DataAPI.VersionedQuestionFromID(qid)
	test.OK(t, err)
	va1, err := testData.DataAPI.VersionedAnswerFromID(aid1)
	test.OK(t, err)
	va2, err := testData.DataAPI.VersionedAnswerFromID(aid2)
	test.OK(t, err)

	id, err := testData.DataAPI.InsertVersionedQuestion(vq, []*common.VersionedAnswer{va1, va2}, nil)
	test.OK(t, err)

	vas, err := testData.DataAPI.VersionedAnswers([]*api.AnswerQueryParams{&api.AnswerQueryParams{QuestionID: id, LanguageID: EN}})
	test.OK(t, err)
	test.Equals(t, 2, len(vas))
	test.Equals(t, vas[0].AnswerText, "answerText")
	test.Equals(t, vas[1].AnswerText, "answerText2")
}

func TestInsertVersionedQuestionNoAnswers(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)

	vq, err := testData.DataAPI.VersionedQuestionFromID(qid)
	test.OK(t, err)

	id, err := testData.DataAPI.InsertVersionedQuestion(vq, []*common.VersionedAnswer{}, nil)
	test.OK(t, err)

	vas, err := testData.DataAPI.VersionedAnswers([]*api.AnswerQueryParams{&api.AnswerQueryParams{QuestionID: id, LanguageID: EN}})
	test.OK(t, err)
	test.Equals(t, 0, len(vas))
}

func TestInsertVersionedQuestionVersionsParent(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pqid := insertQuestionVersion("parentTag", "questionText", "questionType", 1, nil, testData, t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, &pqid, testData, t)

	vq, err := testData.DataAPI.VersionedQuestionFromID(qid)
	test.OK(t, err)

	id, err := testData.DataAPI.InsertVersionedQuestion(vq, []*common.VersionedAnswer{}, nil)
	test.OK(t, err)

	vq, err = testData.DataAPI.VersionedQuestionFromID(id)
	test.OK(t, err)

	vas, err := testData.DataAPI.VersionedAnswers([]*api.AnswerQueryParams{&api.AnswerQueryParams{QuestionID: id, LanguageID: EN}})
	test.OK(t, err)
	test.Equals(t, 0, len(vas))
	test.Assert(t, pqid != *vq.ParentQuestionID, "Expected previous and current parent id's to not match")

	ppvq, err := testData.DataAPI.VersionedQuestionFromID(pqid)
	pvq, err := testData.DataAPI.VersionedQuestionFromID(*vq.ParentQuestionID)
	test.Equals(t, ppvq.QuestionText, pvq.QuestionText)
	test.OK(t, err)
}

func TestInsertVersionedQuestionVersionsAdditionalFields(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)
	insertAdditionalQuestionFields(qid, EN, `{"blobKey":"blobText"}`, testData, t)

	vq, err := testData.DataAPI.VersionedQuestionFromID(qid)
	test.OK(t, err)

	vaqfs, err := testData.DataAPI.VersionedAdditionalQuestionFields(qid, EN)
	test.OK(t, err)
	test.Equals(t, 1, len(vaqfs))
	test.Equals(t, `{"blobKey":"blobText"}`, string(vaqfs[0].JSON))
	test.Equals(t, qid, vaqfs[0].QuestionID)

	id, err := testData.DataAPI.InsertVersionedQuestion(vq, []*common.VersionedAnswer{}, vaqfs[0])
	test.OK(t, err)

	vaqfs, err = testData.DataAPI.VersionedAdditionalQuestionFields(id, EN)
	test.OK(t, err)
	test.Equals(t, 1, len(vaqfs))
	test.Equals(t, `{"blobKey":"blobText"}`, string(vaqfs[0].JSON))
	test.Equals(t, id, vaqfs[0].QuestionID)
}

func TestInsertVersionedQuestionVersionsParentsAdditionalFields(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	pqid := insertQuestionVersion("myTag2", "questionText", "questionType", 1, nil, testData, t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, &pqid, testData, t)
	insertAdditionalQuestionFields(pqid, EN, `{"blobKey":"blobText"}`, testData, t)

	vq, err := testData.DataAPI.VersionedQuestionFromID(qid)
	test.OK(t, err)

	vaqfs, err := testData.DataAPI.VersionedAdditionalQuestionFields(pqid, EN)
	test.OK(t, err)
	test.Equals(t, 1, len(vaqfs))
	test.Equals(t, `{"blobKey":"blobText"}`, string(vaqfs[0].JSON))
	test.Equals(t, pqid, vaqfs[0].QuestionID)

	id, err := testData.DataAPI.InsertVersionedQuestion(vq, []*common.VersionedAnswer{}, nil)
	test.OK(t, err)

	vq, err = testData.DataAPI.VersionedQuestionFromID(id)
	test.OK(t, err)
	test.Assert(t, *vq.ParentQuestionID != pqid, "Expected parent question ID to have changed")

	vaqfs, err = testData.DataAPI.VersionedAdditionalQuestionFields(*vq.ParentQuestionID, EN)
	test.OK(t, err)
	test.Equals(t, 1, len(vaqfs))
	test.Equals(t, `{"blobKey":"blobText"}`, string(vaqfs[0].JSON))
	test.Equals(t, *vq.ParentQuestionID, vaqfs[0].QuestionID)
}

func TestInsertVersionedQuestionCorrectlyQueriesMultipleAdditionalFields(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)
	insertAdditionalQuestionFields(qid, EN, `{"blobKey":"blobText"}`, testData, t)
	insertAdditionalQuestionFields(qid, EN, `{"blobKey2":"blobText2"}`, testData, t)

	vq, err := testData.DataAPI.VersionedQuestionFromID(qid)
	test.OK(t, err)

	vaqfs, err := testData.DataAPI.VersionedAdditionalQuestionFields(qid, EN)
	test.OK(t, err)
	test.Equals(t, 2, len(vaqfs))
	test.Equals(t, `{"blobKey":"blobText"}`, string(vaqfs[0].JSON))
	test.Equals(t, `{"blobKey2":"blobText2"}`, string(vaqfs[1].JSON))
	test.Equals(t, qid, vaqfs[0].QuestionID)
	test.Equals(t, qid, vaqfs[1].QuestionID)

	id, err := testData.DataAPI.InsertVersionedQuestion(vq, []*common.VersionedAnswer{}, vaqfs[0])
	test.OK(t, err)

	vaqfs, err = testData.DataAPI.VersionedAdditionalQuestionFields(id, EN)
	test.OK(t, err)
	test.Equals(t, 1, len(vaqfs))
	test.Equals(t, `{"blobKey":"blobText"}`, string(vaqfs[0].JSON))
	test.Equals(t, id, vaqfs[0].QuestionID)
}

func TestGetQuestionInfoForTagsCorrectlyMergesMultipleAdditionalFields(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)
	insertAdditionalQuestionFields(qid, EN, `{"blobKey":"blobText"}`, testData, t)
	insertAdditionalQuestionFields(qid, EN, `{"blobKey2":"blobText2"}`, testData, t)

	info, err := testData.DataAPI.GetQuestionInfoForTags([]string{"myTag"}, EN)
	test.OK(t, err)
	_, ok1 := info[0].AdditionalFields["blobKey"]
	test.Assert(t, ok1, "blobKey did not exist as expected in map %v", info[0].AdditionalFields)
	_, ok2 := info[0].AdditionalFields["blobKey2"]
	test.Assert(t, ok2, "blobKey2 did not exist as expected in map %v", info[0].AdditionalFields)
}

//answerTag, answerText, answerType, status string, ordering, questionID, version int64
func TestVersionedAnswerDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)
	insertAnswerVersion("myTag", "answerText", "answerType", 1, qid, testData, t)
	vas, err := testData.DataAPI.VersionedAnswers([]*api.AnswerQueryParams{&api.AnswerQueryParams{QuestionID: qid, LanguageID: EN, AnswerTag: "myTag"}})
	test.OK(t, err)
	test.Equals(t, vas[0].AnswerText, "answerText")
	test.Equals(t, vas[0].AnswerType, "answerType")
}

func TestVersionedAnswerMultipleDataAccess(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)
	qid2 := insertQuestionVersion("myTag", "questionText", "questionType", 2, nil, testData, t)
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
	test.Equals(t, vas[0].AnswerText, "answerText")
	test.Equals(t, vas[1].AnswerText, "answerText2")
}

func TestVersionedAnswerFromID(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)
	qid := insertQuestionVersion("myTag", "questionText", "questionType", 1, nil, testData, t)
	aid := insertAnswerVersion("myTag", "answerText", "answerType", 1, qid, testData, t)

	va, err := testData.DataAPI.VersionedAnswerFromID(aid)
	test.OK(t, err)
	test.Equals(t, va.AnswerText, "answerText")
}

func TestVersionedAnswerFromIDNoRows(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()
	testData.StartAPIServer(t)

	_, err := testData.DataAPI.VersionedAnswerFromID(10000)
	test.Equals(t, api.NoRowsError, err)
}
