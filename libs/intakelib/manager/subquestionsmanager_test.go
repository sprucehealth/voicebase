package manager

import (
	"container/list"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestSubquestionsManager_parse(t *testing.T) {
	subquestionsConfigJSON := `
	{
		"screens": [
		  {
		    "header_title": "Tell us more about <parent_answer_text>:",
		    "header_title_has_tokens": true,
		    "questions": [
		      {
		        "question": "q_derm_rash_form",
		        "id": "40606",
		        "question_title": "What form was it in?",
		        "question_title_has_tokens": false,
		        "question_type": "q_type_segmented_control",
		        "type": "q_type_segmented_control",
		        "question_summary": "Form",
		        "potential_answers": [
		          {
		            "id": "126478",
		            "potential_answer": "Cream",
		            "potential_answer_summary": "Cream",
		            "answer_type": "a_type_segmented_control",
		            "type": "a_type_segmented_control",
		            "ordering": "0",
		            "to_alert": false,
		            "answer_tag": "a_derm_rash_form_cream"
		          },
		          {
		            "id": "126479",
		            "potential_answer": "Ointment",
		            "potential_answer_summary": "Ointment",
		            "answer_type": "a_type_segmented_control",
		            "type": "a_type_segmented_control",
		            "ordering": "1",
		            "to_alert": false,
		            "answer_tag": "a_derm_rash_form_ointment"
		          },
		          {
		            "id": "126480",
		            "potential_answer": "Other",
		            "potential_answer_summary": "Other",
		            "answer_type": "a_type_segmented_control",
		            "type": "a_type_segmented_control",
		            "ordering": "2",
		            "to_alert": false,
		            "answer_tag": "a_derm_rash_form_other"
		          }
		        ],
		        "to_prefill": false,
		        "prefilled_with_previous_answers": false,
		        "required": true,
		        "to_alert": false,
		        "alert_text": ""
		      }
		    ],
		    "type": "screen_type_questions"
		  }
		]
	}`

	q := &multipleChoiceQuestion{
		questionInfo: &questionInfo{
			Parent: &questionScreen{
				screenInfo: &screenInfo{},
			},
		},
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(subquestionsConfigJSON), &data); err != nil {
		t.Fatal(err)
	}

	s := newSubquestionsManagerForQuestion(q, &visitManager{})
	if err := s.unmarshalMapFromClient(data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 1, len(s.screenConfigs))
	qs, ok := s.screenConfigs[0].(*questionScreen)
	if !ok {
		t.Fatalf("Expected question screen instead got %T", s.screenConfigs[0])
	}
	test.Equals(t, 1, len(qs.Questions))

	scq, ok := qs.Questions[0].(*multipleChoiceQuestion)
	if !ok {
		t.Fatalf("Expected multiple choice question instead got %T", scq)
	}

	test.Equals(t, questionTypeSegmentedControl.String(), scq.Type)
	test.Equals(t, 3, len(scq.PotentialAnswers))
}

func TestSubquestionsManager_inflateSubScreens(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/multichoice_with_subquestions_and_answers.json")
	if err != nil {
		t.Fatal(err)
	}

	var dataMap map[string]interface{}
	if err := json.Unmarshal(data, &dataMap); err != nil {
		t.Fatal(err)
	}

	mcq := &multipleChoiceQuestion{
		questionInfo: &questionInfo{},
	}

	// lets set the parent of the question to be a question screen
	qs := &questionScreen{
		screenInfo:    &screenInfo{},
		subscreensMap: map[string][]screen{},
	}
	mcq.setLayoutParent(qs)

	// lets set the question screen to be hosted within a section
	// create a section within which to host a screen
	se := &section{
		LayoutUnitID: "se1",
	}
	qs.setLayoutParent(se)

	// setup the screenList within which to hold the subscreens and section-level screens
	l := list.New()
	l.PushBack(qs)

	// iniitialize the visitManager
	dataSource := &visitManager{
		questionMap: map[string]*questionData{},
		sectionScreensMap: map[string]*list.List{
			"se1": l,
		},
		questionIDToAnswerMap: make(map[string]patientAnswer),
	}

	// unmarshal existing answer
	answer := dataMap["answers"].(map[string]interface{})["which_steroids"].(map[string]interface{})
	dataSource.questionIDToAnswerMap["which_steroids"], err = getPatientAnswer(answer)
	test.OK(t, err)

	// unmarshal the json into the multiple choice question.
	questionMap := dataMap["screen_with_multiplechoice_question"].(map[string]interface{})["questions"].([]interface{})[0].(map[string]interface{})
	if err := mcq.unmarshalMapFromClient(questionMap, qs, dataSource); err != nil {
		t.Fatal(err)
	}

	// ensure at this point that the inflation of subscreens has not happened yet (should only
	// happen once the datasource notifies all listeners that setup is complete)
	test.Equals(t, 0, len(mcq.subquestionsManager.subScreensMap))

	// notify all listeners so that the inflation of the subscreens happens
	if err := dataSource.notifyListenersOfSetupComplete(); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, true, mcq.subquestionsManager != nil)

	// at this point there should be an answer to the question with subscreens populated for each
	// answer
	test.Equals(t, 1, len(mcq.answer.topLevelAnswers()))

	// single screen in the config
	test.Equals(t, 1, len(mcq.subquestionsManager.screenConfigs))

	// confirm questions on the subquestions config
	test.Equals(t, 6, len(mcq.subquestionsManager.screenConfigs[0].(*questionScreen).questions()))

	// there should be 1 entry in the subscreens map one for the top level answer
	test.Equals(t, 1, len(mcq.subquestionsManager.subScreensMap))

	// confirm the number of screens for each top level answer
	for _, aItem := range mcq.answer.topLevelAnswers() {
		test.Equals(t, 1, len(aItem.subscreens()))

		// confirm the number of questions on each screen
		test.Equals(t, 6, len(aItem.subscreens()[0].(*questionScreen).questions()))

		// confirm that each subscreen has the question and the corresponding answer as a condition on the screen
		for _, subscreen := range aItem.subscreens() {
			if subscreen.condition() == nil {
				t.Fatalf("Expected a condition on the subscreen")
			}

			answerAnyCondition, ok := subscreen.condition().(*answerContainsAnyCondition)
			if !ok {
				t.Fatalf("Expected an answer contains any condition on the subscreen")
			}

			test.Equals(t, mcq.id(), answerAnyCondition.QuestionID)
			test.Equals(t, 1, len(answerAnyCondition.PotentialAnswersID))
			test.Equals(t, aItem.potentialAnswerID(), answerAnyCondition.PotentialAnswersID[0])
		}

		// confirm that each of the questions on the screen have an answer set
		for _, subquestionItem := range aItem.subscreens()[0].(*questionScreen).questions() {
			pa, err := subquestionItem.patientAnswer()
			if err != nil {
				t.Fatal(err)
			} else if pa.isEmpty() {
				t.Fatal("Expected a patient answer to exist for each subquestion but it didnt")
			}
		}

		// there should be [(6 questions  + 1 screen) * 1 top level answer] unique layoutUnits listed as dependancies in the dependantsMap
		uniqueDependancies := make(map[string]bool)
		for _, dependencies := range dataSource.dependantsMap {
			for _, dependency := range dependencies {
				uniqueDependancies[dependency.layoutUnitID()] = true
			}
		}

		test.Equals(t, 7, len(uniqueDependancies))

	}

}
