package info_intake

import (
	"encoding/json"
	"github.com/sprucehealth/backend/common"
	"io/ioutil"
	"testing"
)

type PatientVisit struct {
	PatientVisitId int64             `json:"patient_visit_id,string"`
	Visit          *InfoIntakeLayout `json:"health_condition"`
}

func parseFileToGetHealthCondition(t *testing.T) (healthCondition *InfoIntakeLayout) {
	fileContents, err := ioutil.ReadFile("../api-response-examples/v1/patient/visit.json")
	if err != nil {
		t.Fatal("Unable to open the json representation of the patient visit for testing:" + err.Error())
	}
	patientVisit := &PatientVisit{}
	err = json.Unmarshal(fileContents, &patientVisit)
	healthCondition = patientVisit.Visit
	if err != nil {
		t.Fatal("Unable to parse the json representation of a patient visit :" + err.Error())
	}
	return healthCondition
}

func TestParsingOfPatientVisit(t *testing.T) {
	parseFileToGetHealthCondition(t)
}

func TestSectionInformationParsing(t *testing.T) {
	visit := parseFileToGetHealthCondition(t)
	if visit.Sections == nil || len(visit.Sections) == 0 {
		t.Fatal("There are no sections being parsed in the visit when there should be")
	}

	for _, section := range visit.Sections {
		if section.SectionId == 0 {
			t.Fatal("SectionId not set when it should be")
		}

		if section.SectionTitle == "" {
			t.Fatal("SectionTitle not set when it should be")
		}

		if section.SectionTag == "" {
			t.Fatal("SectionTag not set when it should be")
		}
	}
}

func TestQuestionsParsing(t *testing.T) {
	visit := parseFileToGetHealthCondition(t)
	for _, section := range visit.Sections {
		for _, screen := range section.Screens {
			if screen.Questions == nil || len(screen.Questions) == 0 {
				t.Fatal("No questions present when there should be atleast one question for each section")
			}
			for _, question := range screen.Questions {
				if question.QuestionId == 0 {
					t.Fatal("No question id present when it should be")
				}

				if question.QuestionTag == "" {
					t.Fatal("No question tag present when it should be")
				}

				if question.QuestionType == "" {
					t.Fatal("No question type present when it should be")
				}

			}
		}
	}
}

func TestPotentialAnswersParsing(t *testing.T) {
	visit := parseFileToGetHealthCondition(t)
	for _, section := range visit.Sections {
		for _, screen := range section.Screens {

			for _, question := range screen.Questions {
				if question.QuestionType == "q_type_multiple_choice" {
					if question.PotentialAnswers == nil || len(question.PotentialAnswers) == 0 {
						t.Fatalf("No potential answers for question with id %d when there always should be one", question.QuestionId)
					}
				}

				for _, potentialAnswer := range question.PotentialAnswers {
					if potentialAnswer.AnswerId == 0 {
						t.Fatal("There should be a potential answer id when there isnt")
					}

					if potentialAnswer.AnswerType == "" {
						t.Fatalf("There should be an answer type for answer id %d when there isn't", potentialAnswer.AnswerId)
					}

					switch question.QuestionType {
					case "q_type_free_text", "q_type_single_entry":
					default:
						if potentialAnswer.Answer == "" {
							t.Fatalf("There should be an answer when there isn't for answer id %d", potentialAnswer.AnswerId)
						}
					}
				}
			}
		}
	}
}

func TestPatientAnswerParsing(t *testing.T) {
	visit := parseFileToGetHealthCondition(t)
	for _, section := range visit.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.Answers == nil {
					continue
				}
				for _, patientAnswer := range question.Answers {
					answer := patientAnswer.(*common.AnswerIntake)
					if answer.AnswerIntakeId.Int64() == 0 {
						t.Fatal("Patient answer id is not set when it should be")
					}

					if answer.QuestionId.Int64() == 0 {
						t.Fatal("question id not set for subquestion")
					}

					if answer.PotentialAnswerId.Int64() == 0 {
						t.Fatal("potential answer id not set for subquestion")
					}
				}
			}
		}
	}
}

func TestPhotoSlotsParsing(t *testing.T) {
	visit := parseFileToGetHealthCondition(t)
	for _, section := range visit.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.QuestionType == "q_type_photo_section" {
					if len(question.PhotoSlots) == 0 {
						t.Fatal("Expected photoslots to exist for question")
					}
				}
			}
		}
	}
}

func TestSubQuestionsInQuestionsParsing(t *testing.T) {
	visit := parseFileToGetHealthCondition(t)

	currentMedicationsEntryTag := "q_current_medications_entry"
	currentMedicationsEntry := false

	for _, section := range visit.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {

				if question.QuestionTag == currentMedicationsEntryTag {
					currentMedicationsEntry = true

					// if question with subquestions found in this round of the loop check the contents
					if question.SubQuestionsConfig == nil || len(question.SubQuestionsConfig.Questions) == 0 {
						t.Fatalf("Expected subquestions to exist for question %s but they dont", question.QuestionTag)
					}

					for _, subQuestion := range question.SubQuestionsConfig.Questions {
						if subQuestion.QuestionId == 0 {
							t.Fatal("Id not set for subquestion")
						}

						if subQuestion.QuestionTag == "" {
							t.Fatal("Question tag not set for subquestion")
						}

						if subQuestion.QuestionType == "" {
							t.Fatal("Question type not set for subquestion")
						}
					}
					break
				}
			}
		}
	}

	if !currentMedicationsEntry {
		t.Fatalf("Expected question %s to be found in the layout but it wasnt", currentMedicationsEntryTag)
	}
}

func TestSubQuestionsInScreensParsing(t *testing.T) {
	visit := parseFileToGetHealthCondition(t)

	prevPrescriptionsListsFound := false
	prevPrescriptionsSelectTag := "q_acne_prev_prescriptions_select"

	prevOtcProductsFound := false
	prevOtcProductsSelectTag := "q_acne_prev_otc_select"

	for _, section := range visit.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {

				questionWithSubQuestionsExpectedFound := false
				if question.QuestionTag == prevPrescriptionsSelectTag {
					prevPrescriptionsListsFound = true
				} else if question.QuestionTag == prevOtcProductsSelectTag {
					prevOtcProductsFound = true
				}

				// if question with subquestions found in this round of the loop check the contents
				if questionWithSubQuestionsExpectedFound {
					if question.SubQuestionsConfig == nil || len(question.SubQuestionsConfig.Screens) == 0 {
						t.Fatalf("Expected subquestions to exist for question %s but they dont", question.QuestionTag)
					}

					for _, screen := range question.SubQuestionsConfig.Screens {
						if len(screen.Questions) == 0 {
							t.Fatal("Expected screen within subquestion to contain questions but it doesnt")
						}

						for _, subQuestion := range screen.Questions {
							if subQuestion.QuestionId == 0 {
								t.Fatal("Id not set for subquestion")
							}

							if subQuestion.QuestionTag == "" {
								t.Fatal("Question tag not set for subquestion")
							}

							if subQuestion.QuestionType == "" {
								t.Fatal("Question type not set for subquestion")
							}
						}
					}
				}
			}
		}
	}

	if !(prevPrescriptionsListsFound && prevOtcProductsFound) {
		t.Fatal("Expected both questions with subquestions config to be found but they werent")
	}
}
