package patient_file

import (
	"testing"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/info_intake"
	"github.com/sprucehealth/backend/test"
)

func TestBuildContext_PhotoPopulation(t *testing.T) {
	context := common.NewViewContext(nil)
	question := &info_intake.Question{
		QuestionTag: "q_test_me",
	}

	photoSections := []*common.PhotoIntakeSection{
		{
			Name: "section1",
			Photos: []*common.PhotoIntakeSlot{
				{
					PhotoURL: "https://photo1",
					Name:     "slot1",
					PhotoID:  1,
				},
			},
		},
	}

	// generalize the answers
	answers := make([]common.Answer, len(photoSections))
	for i, photoSection := range photoSections {
		answers[i] = photoSection
	}

	// lets populate photo answers for a single question
	test.OK(t, populatePatientPhotos(answers, question, context))
	// the context should have the q_test_me:photos key populated
	data, ok := context.Get("q_test_me:photos")
	test.Equals(t, true, ok)

	// the photos should get populated under the <q_tag>:photos
	// key with data transformed into the view data that is expected by
	// the photo views.
	viewData := data.([]info_intake.TitlePhotoListData)
	testTitlePhotoListData(viewData, photoSections, t)

	// at this point the global collection of photos should be the same
	// as the photos pertaining to the question specific key
	globalPhotosData, ok := context.Get("patient_visit_photos")
	test.Equals(t, true, ok)
	test.Equals(t, data, globalPhotosData)

	// now lets pass another question through the method to ensure that the
	// global list of photos continue to get populated
	question2 := &info_intake.Question{
		QuestionTag: "q_test_me2",
	}

	photoSections2 := []*common.PhotoIntakeSection{
		{
			Name: "section2",
			Photos: []*common.PhotoIntakeSlot{
				{
					PhotoURL: "https://photo2",
					Name:     "slot2",
					PhotoID:  2,
				},
			},
		},
	}

	// generalize the answers
	answers = make([]common.Answer, len(photoSections))
	for i, photoSection := range photoSections2 {
		answers[i] = photoSection
	}

	test.OK(t, populatePatientPhotos(answers, question2, context))

	// the context should now have a cumulative list of photos
	// from both questions for the key patient_visit_photos
	data, ok = context.Get("patient_visit_photos")
	test.Equals(t, true, ok)
	viewData = data.([]info_intake.TitlePhotoListData)
	testTitlePhotoListData(viewData, append(photoSections, photoSections2...), t)

	// the context should have the photo list also populated for
	// q_test_me2:photos
	data, ok = context.Get("q_test_me2:photos")
	test.Equals(t, true, ok)
	testTitlePhotoListData(data.([]info_intake.TitlePhotoListData), photoSections2, t)

}

// This test is to ensure that the other free text gets populated alongside the other selection with
// comma separated values for the other free text entered
func TestOtherFreeTextPopulation(t *testing.T) {
	context := common.NewViewContext(nil)
	question := &info_intake.Question{
		QuestionID:  101,
		QuestionTag: "q_test_me",
		PotentialAnswers: []*info_intake.PotentialAnswer{
			{
				AnswerID: 1,
				Answer:   "answer1",
			},
			{
				AnswerID: 2,
				Answer:   "answer2",
			},
			{
				AnswerID: 3,
				Answer:   "answer3",
			},
		},
	}

	answersSelected := []*common.AnswerIntake{
		{
			QuestionID:        encoding.DeprecatedNewObjectID(101),
			PotentialAnswerID: encoding.DeprecatedNewObjectID(1),
		},
		{
			QuestionID:        encoding.DeprecatedNewObjectID(101),
			PotentialAnswerID: encoding.DeprecatedNewObjectID(1),
			AnswerText:        "other1",
		},
		{
			QuestionID:        encoding.DeprecatedNewObjectID(101),
			PotentialAnswerID: encoding.DeprecatedNewObjectID(1),
			AnswerText:        "other2",
		},
		{
			QuestionID:        encoding.DeprecatedNewObjectID(101),
			PotentialAnswerID: encoding.DeprecatedNewObjectID(1),
			AnswerText:        "other3",
		},
		{
			QuestionID:        encoding.DeprecatedNewObjectID(101),
			PotentialAnswerID: encoding.DeprecatedNewObjectID(3),
		},
		{
			QuestionID:        encoding.DeprecatedNewObjectID(101),
			PotentialAnswerID: encoding.DeprecatedNewObjectID(3),
			AnswerText:        "other4",
		},
	}

	// generalize the answers
	answers := make([]common.Answer, len(answersSelected))
	for i, answerSelected := range answersSelected {
		answers[i] = answerSelected
	}

	test.OK(t, populateMultipleChoiceAnswers(answers, question, context))

	// the context should have the q_test_me:photos key populated
	data, ok := context.Get("q_test_me:answers")
	test.Equals(t, true, ok)
	dataList := data.([]info_intake.CheckedUncheckedData)
	test.Equals(t, 3, len(dataList))
	test.Equals(t, true, dataList[0].IsChecked)
	test.Equals(t, "answer1 - other1, other2, other3", dataList[0].Value)
	test.Equals(t, false, dataList[1].IsChecked)

	test.Equals(t, true, dataList[2].IsChecked)
	test.Equals(t, "answer3 - other4", dataList[2].Value)

}

func testTitlePhotoListData(tpld []info_intake.TitlePhotoListData, photoSections []*common.PhotoIntakeSection, t *testing.T) {
	test.Equals(t, len(photoSections), len(tpld))
	for j, photoSection := range photoSections {
		test.Equals(t, photoSection.Name, tpld[j].Title)
		test.Equals(t, len(photoSection.Photos), len(tpld[j].Photos))
		for i, photo := range photoSection.Photos {
			test.Equals(t, photo.PhotoURL, tpld[j].Photos[i].PhotoURL)
			test.Equals(t, photo.Name, tpld[j].Photos[i].Title)
		}
	}
}
