package info_intake

import (
	"encoding/json"
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

				if question.QuestionTypes == nil || len(question.QuestionTypes) == 0 {
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
				if (question.PotentialAnswers == nil || len(question.PotentialAnswers) == 0) && !(question.QuestionTypes[0] == "q_type_free_text" || question.QuestionTypes[0] == "q_type_autocomplete") {
					t.Fatalf("No potential answers for question with id %d when there always should be one", question.QuestionId)
				}
				for _, potentialAnswer := range question.PotentialAnswers {
					if potentialAnswer.AnswerId == 0 {
						t.Fatal("There should be a potential answer id when there isnt")
					}

					if potentialAnswer.AnswerTypes == nil || len(potentialAnswer.AnswerTypes) == 0 {
						t.Fatalf("There should be an answer type for answer id %d when there isn't", potentialAnswer.AnswerId)
					}

					if potentialAnswer.Answer == "" {
						t.Fatalf("There should be an answer when there isn't for answer id %d", potentialAnswer.AnswerId)
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
				if question.PatientAnswers == nil {
					continue
				}
				for _, patientAnswer := range question.PatientAnswers {
					if patientAnswer.AnswerIntakeId == 0 {
						t.Fatal("Patient answer id is not set when it should be")
					}

					if patientAnswer.QuestionId == 0 {
						t.Fatal("question id not set for subquestion")
					}

					if patientAnswer.PotentialAnswerId == 0 {
						t.Fatal("potential answer id not set for subquestion")
					}
				}
			}
		}
	}
}

func TestSubQuestionsParsing(t *testing.T) {
	visit := parseFileToGetHealthCondition(t)
	for _, section := range visit.Sections {
		for _, screen := range section.Screens {
			for _, question := range screen.Questions {
				if question.Questions != nil {
					for _, subQuestion := range question.Questions {

						if subQuestion.QuestionId == 0 {
							t.Fatal("Id not set for subquestion")
						}

						if subQuestion.QuestionTag == "" {
							t.Fatal("Question tag not set for subquestion")
						}

						if subQuestion.QuestionTypes == nil || len(subQuestion.QuestionTypes) == 0 {
							t.Fatal("Question type not set for subquestion")
						}
					}
				}
			}
		}
	}
}
