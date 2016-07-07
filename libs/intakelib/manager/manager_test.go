package manager

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
	"github.com/sprucehealth/backend/libs/test"
)

func testWrapScreen(s screen, progress *float32, t *testing.T) []byte {
	msg, err := s.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}

	msgData, err := proto.Marshal(msg.(proto.Message))
	if err != nil {
		t.Fatal(err)
	}

	sData := &intake.ScreenData{
		Type:     screenTypeToProtoBufType[s.TypeName()],
		Data:     msgData,
		Progress: progress,
	}

	data, err := proto.Marshal(sData)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestManager_ComputeNextScreen(t *testing.T) {

	vm := initializeManagerWithVisit(t, "testdata/eczema_female_complete.json", nil)
	// iterate through the screens and compute a slice of expected byte buffers,
	// skipping screens that are hidden as well as the overview screen.
	var byteBuffers [][]byte

	screenIDs := []string{"visit_overview"}
	for _, section := range vm.visit.Sections {
		for _, screen := range section.Screens {
			if screen.visibility() == hidden {
				continue
			}

			screenIDs = append(screenIDs, screen.layoutUnitID())

			progress, err := vm.computeProgress(screen)
			if err != nil {
				t.Fatal(err)
			}

			byteBuffers = append(byteBuffers, testWrapScreen(screen, progress, t))

			// add all subscreens
			ssContainer, ok := screen.(subscreensContainer)
			if ok {
				for _, ss := range ssContainer.subscreens() {
					if ss.visibility() == hidden {
						continue
					}
					screenIDs = append(screenIDs, ss.layoutUnitID())

					progress, err := vm.computeProgress(ss)
					if err != nil {
						t.Fatal(err)
					}
					byteBuffers = append(byteBuffers, testWrapScreen(ss, progress, t))
				}
			}
		}
		screenIDs = append(screenIDs, "visit_overview")
	}

	// compare the data retreieved from compute next screen with the expected data.

	i := 0
	j := 0
	for j < len(byteBuffers) {

		var screenData []byte
		var err error

		// if the current screenID represents a visitoverview screen, then get the next screen from the lib
		// to compare it with the expected data in the array
		if screenIDs[i] == "visit_overview" {
			screenData, err = vm.Screen(screenIDs[i+1])
			if err != nil {
				t.Fatal(err)
			}
		} else {
			data, err := vm.ComputeNextScreen(screenIDs[i])
			if err != nil {
				t.Fatal(err)
			}

			// extract screenID from the data
			var sd intake.ScreenIDData
			if err := proto.Unmarshal(data, &sd); err != nil {
				t.Fatal(err)
			}

			screenData, err = vm.Screen(*sd.Id)
			if err != nil {
				t.Fatal(err)
			}
		}

		// if the next screen is going to be the visit_overview screen,
		// this means the data returned from the intake lib represents the visit overview
		// screen, so skip over the id.
		if screenIDs[i+1] != "visit_overview" {
			if bytes.Compare(byteBuffers[j], screenData) != 0 {
				t.Fatalf("Data retrieved at index %d for screenID %s does not match expected data from visit manager", j, screenIDs[i])
			}
			j++
		}

		i++
	}
}

func TestManager_progress(t *testing.T) {
	vm := initializeManagerWithVisit(t, "testdata/eczema.json", nil)

	for _, section := range vm.visit.Sections {
		for i, screen := range section.Screens {
			progress := (float32(i) + 1.0) / float32(len(section.Screens))
			calcProgress, err := vm.computeProgress(screen)
			if err != nil {
				t.Fatal(err)
			}
			test.Equals(t, progress, *calcProgress)
		}
	}
}

func TestManager_VisitOverviewScreen(t *testing.T) {
	vm := initializeManagerWithVisit(t, "testdata/eczema_female_complete.json", []*intake.KeyValuePair{
		{
			Key:   proto.String("gender"),
			Value: proto.String("female"),
		},
		{
			Key:   proto.String("is_pharmacy_set"),
			Value: proto.String("true"),
		},
	})

	data, err := vm.ComputeNextScreen("")
	if err != nil {
		t.Fatal(err)
	}

	var sID intake.ScreenIDData
	if err := proto.Unmarshal(data, &sID); err != nil {
		t.Fatal(err)
	}

	overviewScreenData, err := vm.Screen(*sID.Id)
	if err != nil {
		t.Fatal(err)
	}

	var sd intake.ScreenData
	if proto.Unmarshal(overviewScreenData, &sd); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, intake.ScreenData_VISIT_OVERVIEW, *sID.Type)

	var vos intake.VisitOverviewScreen
	if err := proto.Unmarshal(sd.Data, &vos); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, 3, len(vos.Sections))
	test.Equals(t, "That's all the information your doctor will need!", *vos.Text)

	test.Equals(t, "Your Symptoms", *vos.Sections[0].Name)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[0].CurrentFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED_STATE_UNDEFINED, *vos.Sections[0].PrevFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_ENABLED, *vos.Sections[0].CurrentEnabledState)

	test.Equals(t, "Photos", *vos.Sections[1].Name)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[1].CurrentFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED_STATE_UNDEFINED, *vos.Sections[1].PrevFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_ENABLED, *vos.Sections[1].CurrentEnabledState)

	test.Equals(t, "Medical History", *vos.Sections[2].Name)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[2].CurrentFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED_STATE_UNDEFINED, *vos.Sections[2].PrevFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_ENABLED, *vos.Sections[2].CurrentEnabledState)

	// lets unset the answers to the photo question to ensure that the visit overview drops the user back into the photos section
	pScreen, ok := vm.visit.Sections[1].Screens[0].(*mediaScreen)
	if !ok {
		t.Fatal("Expected photo screen")
	}
	for _, pq := range pScreen.MediaQuestions {
		pqi, ok := pq.(*mediaQuestion)
		if !ok {
			t.Fatal("Expected photo question")
		}

		pqi.answer = nil
	}

	// recompute status
	test.OK(t, vm.visitStatus.update())

	data, err = vm.ComputeNextScreen("")
	if err != nil {
		t.Fatal(err)
	}

	if err := proto.Unmarshal(data, &sID); err != nil {
		t.Fatal(err)
	}

	overviewScreenData, err = vm.Screen(*sID.Id)
	if err != nil {
		t.Fatal(err)
	}

	if proto.Unmarshal(overviewScreenData, &sd); err != nil {
		t.Fatal(err)
	}

	if err := proto.Unmarshal(sd.Data, &vos); err != nil {
		t.Fatal(err)
	}

	// at this point should be back on photo section
	test.Equals(t, "Now let's take photos for your doctor to make a diagnosis.", *vos.Text)

	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[0].CurrentFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[0].PrevFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_ENABLED, *vos.Sections[0].CurrentEnabledState)

	test.Equals(t, intake.VisitOverviewScreen_Section_UNFILLED, *vos.Sections[1].CurrentFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[1].PrevFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_ENABLED, *vos.Sections[1].CurrentEnabledState)

	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[2].CurrentFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[2].PrevFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_DISABLED, *vos.Sections[2].CurrentEnabledState)

	// ensure that after getting the visit overview yet again the previous state has been updated.
	data, err = vm.ComputeNextScreen("")
	if err != nil {
		t.Fatal(err)
	}

	if err := proto.Unmarshal(data, &sID); err != nil {
		t.Fatal(err)
	}

	overviewScreenData, err = vm.Screen(*sID.Id)
	if err != nil {
		t.Fatal(err)
	}

	if proto.Unmarshal(overviewScreenData, &sd); err != nil {
		t.Fatal(err)
	}

	if err := proto.Unmarshal(sd.Data, &vos); err != nil {
		t.Fatal(err)
	}

	// at this point should be back on photo section
	test.Equals(t, "Now let's take photos for your doctor to make a diagnosis.", *vos.Text)

	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[0].CurrentFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[0].PrevFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_ENABLED, *vos.Sections[0].CurrentEnabledState)

	test.Equals(t, intake.VisitOverviewScreen_Section_UNFILLED, *vos.Sections[1].CurrentFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_UNFILLED, *vos.Sections[1].PrevFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_ENABLED, *vos.Sections[1].CurrentEnabledState)

	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[2].CurrentFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_FILLED, *vos.Sections[2].PrevFilledState)
	test.Equals(t, intake.VisitOverviewScreen_Section_DISABLED, *vos.Sections[2].CurrentEnabledState)

}

type testAnswerData struct {
	Data []byte
}

type mockClientImpl struct {
	answersSet map[string]*testAnswerData
}

func (m *mockClientImpl) PersistAnswerForQuestion(data []byte) error {
	var cd intake.ClientAnswerData
	if err := proto.Unmarshal(data, &cd); err != nil {
		return err
	}

	m.answersSet[*cd.QuestionId] = &testAnswerData{
		Data: cd.ClientAnswerJson,
	}
	return nil
}

// TestManager_PersistAnswers tests to ensure that the framework to
// set answers by the client is working as expected, and the manager
// is directing the client to persist answers when updated.
func TestManager_PersistAnswers(t *testing.T) {
	vm := initializeManagerWithVisit(t, "testdata/eczema.json", nil)

	// let's persist answers to questions

	// Note that the IDs for the questions have been referenced from testfiles/eczema.json
	// to create for a real experience of setting answers.

	aData, err := proto.Marshal(&intake.MultipleChoicePatientAnswer{
		AnswerSelections: []*intake.MultipleChoicePatientAnswer_Selection{
			{
				PotentialAnswerId: proto.String("systemic_or_dangerous_symptoms_none_of_the_above"),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	submittableData, err := proto.Marshal(&intake.PatientAnswerData{
		Type: intake.PatientAnswerData_MULTIPLE_CHOICE.Enum(),
		Data: aData,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := vm.SetAnswerForQuestion("systemic_or_dangerous_symptoms", submittableData); err != nil {
		t.Fatal(err)
	}

	// at this point the client should've received a directive to persist the answer for the question
	cli := vm.cli.(*mockClientImpl)
	if len(cli.answersSet) != 1 {
		t.Fatalf("Expected 1 answer to be persisted on the client side but got %d", len(cli.answersSet))
	} else if cli.answersSet["systemic_or_dangerous_symptoms"] == nil {
		t.Fatalf("Expected answer to be persisted for the question just answered")
	}

	// clear out the answer set to ensure that the answer for the multiple choice is not set again
	// for when we set another answer.
	delete(cli.answersSet, "systemic_or_dangerous_symptoms")

	// lets try persisting a free text question
	aData, err = proto.Marshal(&intake.FreeTextPatientAnswer{
		Text: proto.String("hello"),
	})
	if err != nil {
		t.Fatal(err)
	}

	submittableData, err = proto.Marshal(&intake.PatientAnswerData{
		Type: intake.PatientAnswerData_FREE_TEXT.Enum(),
		Data: aData,
	})
	if err != nil {
		t.Fatal(err)
	}

	questionID := "anything_that_prevents_getting_a_rash_or_makes_it_better"

	if err := vm.SetAnswerForQuestion(questionID, submittableData); err != nil {
		t.Fatal(err)
	}

	if len(cli.answersSet) != 1 {
		t.Fatalf("Expected 1 answer to be persisted on the client side but got %d", len(cli.answersSet))
	} else if cli.answersSet[questionID] == nil {
		t.Fatalf("Expected answer to be persisted for the question just answered")
	}

	// now lets make the question just answered hidden and call persist answer again (should submit an empty answer)
	vm.questionMap[questionID].questionRef.setVisibility(hidden)
	vm.questionMap[questionID].isAnswerDirty = true
	delete(cli.answersSet, questionID)

	vm.persistAllDirtyQuestions()

	// at this point the answer for the question should be reset to an empty answer
	if len(cli.answersSet) != 1 {
		t.Fatalf("Expected 1 answer to be persisted on the client side but got %d", len(cli.answersSet))
	} else if cli.answersSet[questionID] == nil {
		t.Fatalf("Expected answer to be persisted for the question just answered")
	}

	answerData := cli.answersSet[questionID].Data

	var aJSON clientJSONStructure
	if err := json.Unmarshal(answerData, &aJSON); err != nil {
		t.Fatal(err)
	}

	if len(aJSON.Answers) == 1 {
		t.Fatalf("Expected NO answer to be set instead got %d", len(aJSON.Answers))
	} else if len(aJSON.ClearAnswers) != 1 {
		t.Fatalf("Expected 1 answer to be cleared but none were")
	} else if aJSON.ClearAnswers[0] != questionID {
		t.Fatalf("Expected the free text answer to be marked for clearing but it was %s", aJSON.ClearAnswers[0])
	}
}

func TestManager_PhotoAnswers_ToBeUploaded(t *testing.T) {
	vm := initializeManagerWithVisit(t, "testdata/eczema.json", nil)

	aData, err := proto.Marshal(&intake.MediaSectionPatientAnswer{
		Entries: []*intake.MediaSectionPatientAnswer_MediaSectionEntry{
			{
				Name: proto.String("Face"),
				Media: []*intake.MediaSectionPatientAnswer_MediaSectionEntry_MediaSlotAnswer{
					{
						SlotId: proto.String("mediaSlot_0EUH6QGFO1O40"),
						Name:   proto.String("Face Front"),
						Id: &intake.ID{
							Type: intake.ID_LOCAL.Enum(),
							Id:   proto.String("abc"),
						},
					},
					{
						SlotId: proto.String("mediaSlot_0EUH6QGFO1O42"),
						Name:   proto.String("Slide"),
						Id: &intake.ID{
							Type: intake.ID_SERVER.Enum(),
							Id:   proto.String("103"),
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	submittableData, err := proto.Marshal(&intake.PatientAnswerData{
		Type: intake.PatientAnswerData_MEDIA_SECTION.Enum(),
		Data: aData,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := vm.SetAnswerForQuestion("face", submittableData); err != nil {
		t.Fatal(err)
	}

	// there should be an uploadable item being held in the map
	if len(vm.itemsToBeUploaded) != 1 {
		t.Fatalf("Expected 1 item to be held as in the process of being uploaded instead got %d", len(vm.itemsToBeUploaded))
	}

	// at this point there should not be any item in the answer map as no answer should've been persisted
	cli := vm.cli.(*mockClientImpl)
	if len(cli.answersSet) != 0 {
		t.Fatalf("Expected no answers to be persisted but had %d answers persisted", len(cli.answersSet))
	}

	mediaIDReplacementData, err := proto.Marshal(&intake.MediaIDReplacement{
		Id:  proto.String("104"),
		Url: proto.String("www.google.com"),
	})
	if err != nil {
		t.Fatal(err)
	}

	idData, err := proto.Marshal(&intake.IDReplacementData{
		Type: intake.IDReplacementData_MEDIA_ID.Enum(),
		Data: mediaIDReplacementData,
	})
	if err != nil {
		t.Fatal(err)
	}

	// lets now indicate that the photo was uploaded
	if err := vm.ReplaceID("abc", idData); err != nil {
		t.Fatal(err)
	}

	// now there should be an answer persisted
	if len(cli.answersSet) != 1 {
		t.Fatalf("Expected only the photo answer to be persisted instead got %d answers persisted", len(cli.answersSet))
	} else if cli.answersSet["face"] == nil {
		t.Fatalf("Expcected answer to photo answer to be persisted")
	}

	// at this point there should be no items to be uploaded
	if len(vm.itemsToBeUploaded) != 0 {
		t.Fatalf("Expected no more items to be uploaded instead got %d", len(vm.itemsToBeUploaded))
	}

	// the photo URL should also be set for the slot
	photoQ := vm.questionMap["face"].questionRef.(*mediaQuestion)
	url := photoQ.answer.Sections[0].Media[0].URL
	if url != "www.google.com" {
		t.Fatal("Expected the url to be updated along with server side id but it wasnt")
	}
}

func TestManager_PhotoAnswers(t *testing.T) {
	vm := initializeManagerWithVisit(t, "testdata/eczema.json", nil)

	aData, err := proto.Marshal(&intake.MediaSectionPatientAnswer{
		Entries: []*intake.MediaSectionPatientAnswer_MediaSectionEntry{
			{
				Name: proto.String("Face"),
				Media: []*intake.MediaSectionPatientAnswer_MediaSectionEntry_MediaSlotAnswer{
					{
						SlotId: proto.String("mediaSlot_0EUH6QGFO1O42"),
						Name:   proto.String("Slide"),
						Id: &intake.ID{
							Type: intake.ID_SERVER.Enum(),
							Id:   proto.String("103"),
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	submittableData, err := proto.Marshal(&intake.PatientAnswerData{
		Type: intake.PatientAnswerData_MEDIA_SECTION.Enum(),
		Data: aData,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := vm.SetAnswerForQuestion("face", submittableData); err != nil {
		t.Fatal(err)
	}

	// there should be no items to be uploaded given that the serverID was provided.
	if len(vm.itemsToBeUploaded) != 0 {
		t.Fatalf("Expected no  items to be uploaded instead got %d", len(vm.itemsToBeUploaded))
	}

	// now there should be an answer persisted
	cli := vm.cli.(*mockClientImpl)
	if len(cli.answersSet) != 1 {
		t.Fatalf("Expected only the photo answer to be persisted instead got %d answers persisted", len(cli.answersSet))
	} else if cli.answersSet["face"] == nil {
		t.Fatalf("Expcected answer to photo answer to be persisted")
	}

	// lets now indicate that the photo was uploaded
	mediaIDReplacementData, err := proto.Marshal(&intake.MediaIDReplacement{
		Id: proto.String("104"),
	})
	if err != nil {
		t.Fatal(err)
	}

	idData, err := proto.Marshal(&intake.IDReplacementData{
		Type: intake.IDReplacementData_MEDIA_ID.Enum(),
		Data: mediaIDReplacementData,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := vm.ReplaceID("abc", idData); err == nil {
		t.Fatalf("Expected to get an error when an ID was attempted to be set for an item that was already uploaded")
	}
}

func TestManager_DependantMap(t *testing.T) {
	vm := initializeManagerWithVisit(t, "testdata/eczema.json", nil)

	// at this point the dependants map should be populated. Let's test the expected number of results
	// by going through each section, screen and question and testing out if the number of dependants
	// for each component is as expected

	for _, section := range vm.visit.Sections {

		// each section should be 0 dependants

		for _, screen := range section.Screens {
			// by default each screen should be present as a dependant in each of the condition's dependencies
			if screen.condition() != nil {
				for _, item := range screen.condition().layoutUnitDependencies(vm) {
					if !containsUnit(screen, vm.dependantsMap[item.layoutUnitID()]) {
						t.Fatal("Expected the screen to be present as a dependant but it wasnt")
					}
				}
			}

			switch s := screen.(type) {
			case questionsContainer:

				// if there are questions, then the screen should be present as a dependant for each of the questions
				for _, qItem := range s.questions() {
					if !containsUnit(screen, vm.dependantsMap[qItem.layoutUnitID()]) {
						t.Fatal("Expected the screen to be present as a dependant for the question it wasnt")
					}

					// each question should be present as a dependant for each of the conditions' dependencies
					if qItem.condition() != nil {
						for _, cItem := range qItem.condition().layoutUnitDependencies(vm) {
							if !containsUnit(qItem, vm.dependantsMap[cItem.layoutUnitID()]) {
								t.Fatal("Expected the question to be present as a dependant but it wasnt")
							}
						}
					}

					// each question should also be present as a dependant for the screen
					if !containsUnit(qItem, vm.dependantsMap[screen.layoutUnitID()]) {
						t.Fatal("Expected the question to be present as a dependant for the screen but it wasnt")
					}

					// ...and each screen a dependant on the question
					if !containsUnit(screen, vm.dependantsMap[qItem.layoutUnitID()]) {
						t.Fatal("Expected the screen to be present as a dependant for the question but it wasnt")
					}
				}

			}

		}
	}

	// lets go ahead and attempt to deregister the visit node. This should completely empty out the dependentsMap
	deregisterNodeAndChildren(vm.visit, vm)

	test.Equals(t, 0, len(vm.dependantsMap))
}

func containsUnit(l layoutUnit, items []layoutUnit) bool {
	for _, item := range items {
		if item.layoutUnitID() == l.layoutUnitID() {
			return true
		}
	}
	return false
}

type questionAnswer struct {
	q question
	a patientAnswer
}

// TestManager_evaluateDependants works with the given input files that contain a complete visit object with patient answers,
// and runs the answers through the visit manager as a client would by starting from a visit with no patient answers.
// It then compares the end result (overall visibility state of all screens and questions) with the hand vetted
// output files to ensure that the dependancy management is working as expected.
func TestManager_evaluateDependants(t *testing.T) {
	testManager_evaluateDependencies(t, "testdata/eczema_female_complete.json", "testdata/eczema_female_complete_output.txt", "female")
	testManager_evaluateDependencies(t, "testdata/rash_male_complete.json", "testdata/rash_male_complete_output.txt", "male")

	// this test is checking to ensure that answer conditions take into consideration the visibility of the question and evaluate to false if hidden.
	// the visit object represents a patient that answered following series of answers being specifically tested:
	//		1. select asthma + hiv for "Do you have any of the following medical conditions?" in medical history section
	// 		2. select "I don't know" for "Do you currently have AIDS?" (next question)
	//		3. triage popup comes up (pertaining to aids)
	//		4. go back and deselect HIV
	//		5. move forward and triage popup (pertaining to aids) should not be shown (was previously because visibility of question was not taken into consideration)
	// Note that the answers were modified in the file to include the answer from step #2 even though the visit object in reality would not contain that answer
	// once the answer was updated in step 4 (the visit manager would clear out the answer for that particular question)
	testManager_evaluateDependencies(t, "testdata/rash_unhide_hide_screens.json", "testdata/rash_unhide_hide_screens_output.txt", "male")

	// this test is to ensure that we correctly hide screens with questions that can result in subscreens and the subscreens themselves, when the screen that contains a question
	// with potential subquestions is hidden. It simulates the following scenario:
	//		1. Select "Yes" for "Have you tried any topical steroids for psoriasis?"
	//		2. Select a medication on the next screen
	// 		3. This will result in subquestions pertaining to the answer selection
	//		4. Answer the subquestions
	//		5. Go back to two screens and change answer to "No" for "Have you tried any topical steroids for psoriasis?"
	//		6. Press next, and the next screen should be "What topical steroids have you tried?".
	// Note that the answers in psoriasis_male_subquestions_unhide_hide.json have been manually modified to simulate Step 5 to ensure that the subquestions get hidden.
	testManager_evaluateDependencies(t, "testdata/psoriasis_male_subquestions_unhide_hide.json", "testdata/psoriasis_male_subquestions_unhide_hide_output.txt", "male")
}

// TestManager_computeCurrentVisibilityState works with the given input files that contain a complete visit object with patient answers,
// and initializes the visit with the patient answers to compute the current visibility state and ensure that the overall visibility
// state matches that of the hand vetted output file.
func TestManager_computeCurrentVisibilityState(t *testing.T) {
	testManager_initialState(t, "testdata/eczema_female_complete.json", "testdata/eczema_female_complete_output.txt", "female")
	testManager_initialState(t, "testdata/rash_male_complete.json", "testdata/rash_male_complete_output.txt", "male")
	testManager_initialState(t, "testdata/rash_unhide_hide_screens.json", "testdata/rash_unhide_hide_screens_output.txt", "male")
	testManager_initialState(t, "testdata/psoriasis_male_subquestions_unhide_hide.json", "testdata/psoriasis_male_subquestions_unhide_hide_output.txt", "male")
}

// TestManager_validateRequirements ensures that a completed visit (but not submitted) is considered to have all its requirements met.
func TestManager_validateRequirements(t *testing.T) {

	vm := initializeManagerWithVisit(t, "testdata/eczema_female_complete.json", []*intake.KeyValuePair{
		{
			Key:   proto.String("gender"),
			Value: proto.String("female"),
		},
		{
			Key:   proto.String("is_pharmacy_set"),
			Value: proto.String("true"),
		},
	})

	// given that the intput file is expected to represent a completed visit that is yet to be submitted,
	// every screen should report as being valid
	for _, section := range vm.visit.Sections {
		for _, screen := range section.Screens {
			data, err := validateScreen(screen, vm, vm.serializer)
			if err != nil {
				t.Fatal(err)
			} else if len(data) == 0 {
				t.Fatalf("Expected data to be returned for screen %s but got nothing.", screen.layoutUnitID())
			}

			var res intake.ValidateRequirementsResult
			if err := proto.Unmarshal(data, &res); err != nil {
				t.Fatal(err)
			} else if *res.Status != intake.ValidateRequirementsResult_OK {
				t.Fatalf("Expected a valid screen but got an invalid screen")
			}
		}
	}

	resData, err := vm.ValidateRequirementsInLayout()
	if err != nil {
		t.Fatal(err)
	}

	var res intake.ValidateRequirementsResult
	if err := proto.Unmarshal(resData, &res); err != nil {
		t.Fatal(err)
	} else if *res.Status != intake.ValidateRequirementsResult_OK {
		t.Fatal("Expected validation result to indicate that the visit had all its requirements met")
	}

	// on an empty visit, on the other hand, the requirements should not considered complete
	vm = initializeManagerWithVisit(t, "testdata/eczema.json", nil)

	// requirements should not be met
	resData, err = vm.ValidateRequirementsInLayout()
	if err != nil {
		t.Fatal(err)
	}

	res = intake.ValidateRequirementsResult{}
	if err := proto.Unmarshal(resData, &res); err != nil {
		t.Fatal(err)
	} else if *res.Status != intake.ValidateRequirementsResult_ERROR {
		t.Fatal("Expected validation result to indicate that the visit did not have all its requirements met")
	}
}

func TestManager_listeners(t *testing.T) {
	v := &visitManager{}

	var listener1Called bool
	if err := v.subscribeForSetupCompletionNotification(func() error {
		listener1Called = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	var listener2Called bool
	if err := v.subscribeForSetupCompletionNotification(func() error {
		listener2Called = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := v.notifyListenersOfSetupComplete(); err != nil {
		t.Fatal(err)
	}
	if !listener1Called || !listener2Called {
		t.Fatal("One of the listeenrs did not fire as expected")
	}

	test.Equals(t, 0, len(v.setupCompleteListeners))

	// ensure that listener is called immediately if setup is considered complete
	v.setupComplete = true
	var listener3Called bool
	if err := v.subscribeForSetupCompletionNotification(func() error {
		listener3Called = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	test.Equals(t, true, listener3Called)
	test.Equals(t, 0, len(v.setupCompleteListeners))
}

func TestManager_markDirty_visibilityChange(t *testing.T) {
	vm := initializeManagerWithVisit(t, "testdata/eczema_female_complete.json", []*intake.KeyValuePair{
		{
			Key:   proto.String("gender"),
			Value: proto.String("female"),
		},
	})

	// lets go ahead and change the answer to the question
	// re: other medications tried for eczema from Yes to No.
	// this should cause at least the next question to become hidden
	// and therefore be marked as dirty, and an empty answer
	// to be submitted for the question.
	data := prepareSingleSelectAnswer(t, "tried_other_medications_before", "tried_other_medications_before_no")

	if err := vm.SetAnswerForQuestion("tried_other_medications_before", data); err != nil {
		t.Fatal(err)
	}

	mCli := vm.cli.(*mockClientImpl)
	test.Equals(t, 2, len(mCli.answersSet))

	// question just answered should be persisted
	test.Equals(t, true, mCli.answersSet["tried_other_medications_before"] != nil)

	// question following that depends on answer to question just set
	// should become hidden and therefore be required to persist an empty answer.
	test.Equals(t, true, mCli.answersSet["which_other_medications"] != nil)

	// ensure that answer to question following just answered question has empty answer.
	aData := mCli.answersSet["which_other_medications"].Data
	var aJSON clientJSONStructure
	test.OK(t, json.Unmarshal(aData, &aJSON))
	test.Equals(t, 0, len(aJSON.Answers))
	test.Equals(t, 1, len(aJSON.ClearAnswers))

	aData = mCli.answersSet["tried_other_medications_before"].Data
	aJSON = clientJSONStructure{}
	test.OK(t, json.Unmarshal(aData, &aJSON))
	test.Equals(t, 1, len(aJSON.Answers))
	test.Equals(t, 0, len(aJSON.ClearAnswers))

	// now lets go ahead and make the answer to the medications question "Yes"
	// again such that the following question transitions from being hidden to visible
	// causing the question to get marked as being dirty and the complete answer being set.
	data = prepareSingleSelectAnswer(t, "tried_other_medications_before", "tried_other_medications_before_yes")
	if err := vm.SetAnswerForQuestion("tried_other_medications_before", data); err != nil {
		t.Fatal(err)
	}

	test.Equals(t, true, mCli.answersSet["which_other_medications"] != nil)
	aData = mCli.answersSet["which_other_medications"].Data
	aJSON = clientJSONStructure{}
	test.OK(t, json.Unmarshal(aData, &aJSON))
	test.Equals(t, 1, len(aJSON.Answers))

	var mcaClientJSON multipleChoiceAnswerClientJSON
	jsonData, err := json.Marshal(aJSON.Answers["which_other_medications"])
	test.OK(t, err)

	test.OK(t, json.Unmarshal(jsonData, &mcaClientJSON))
	test.Equals(t, true, len(mcaClientJSON.PotentialAnswers) > 0)
}

func prepareMultipleChoiceAnswer(t *testing.T, questionID, potentialAnswerID string) []byte {
	mca := &multipleChoiceAnswer{
		Answers: []topLevelAnswerItem{
			&multipleChoiceAnswerSelection{
				PotentialAnswerID: potentialAnswerID,
			},
		},
	}

	pb, err := mca.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}

	data, err := proto.Marshal(pb)
	if err != nil {
		t.Fatal(err)
	}

	cad := &intake.PatientAnswerData{
		Type: intake.PatientAnswerData_MULTIPLE_CHOICE.Enum(),
		Data: data,
	}

	data, err = proto.Marshal(cad)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func prepareSingleSelectAnswer(t *testing.T, questionID, potentialAnswerID string) []byte {
	mca := &singleSelectAnswer{
		Answer: &multipleChoiceAnswerSelection{
			PotentialAnswerID: potentialAnswerID,
		},
	}

	pb, err := mca.transformToProtobuf()
	if err != nil {
		t.Fatal(err)
	}

	data, err := proto.Marshal(pb)
	if err != nil {
		t.Fatal(err)
	}

	cad := &intake.PatientAnswerData{
		Type: intake.PatientAnswerData_SINGLE_SELECT.Enum(),
		Data: data,
	}

	data, err = proto.Marshal(cad)
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func testManager_evaluateDependencies(t *testing.T, inputFileName, outputFileName, gender string) {

	vm := initializeManagerWithVisit(t, inputFileName, []*intake.KeyValuePair{
		{
			Key:   proto.String("gender"),
			Value: proto.String(gender),
		},
	})

	// create a map to hold on to the patient answers so that they can be re-applied
	questionMap := make(map[string]*questionAnswer)
	for _, qItem := range vm.questionMap {

		a, err := qItem.questionRef.patientAnswer()
		if err == errNoAnswerExists {
			continue
		} else if err != nil {
			t.Fatal(err)
		}

		questionMap[qItem.questionRef.layoutUnitID()] = &questionAnswer{
			q: qItem.questionRef,
			a: a,
		}

		// set the answer to nil for each question so that
		// we can replay the answers on the visit
		switch s := qItem.questionRef.(type) {
		case *freeTextQuestion:
			s.answer = nil
		case *autocompleteQuestion:
			s.answer = nil
		case *mediaQuestion:
			s.answer = nil
		case *multipleChoiceQuestion:
			s.answer = nil
		case *singleEntryQuestion:
			s.answer = nil
		default:
			t.Fatalf("missed question %T", s)
		}
	}

	// re-initialize the state of layoutUnits based on the fact that we no longer have answers
	// to simulate situation where user is starting from scratch.
	vm.computeCurrentVisibilityState(vm.visit)

	// randomly set the answer to each question as the client would via the manager.
	for _, qa := range questionMap {

		pb, err := qa.a.transformToProtobuf()
		if err != nil {
			t.Fatal(err)
		}

		var paType *intake.PatientAnswerData_Type
		switch pb.(type) {
		case *intake.MediaSectionPatientAnswer:
			paType = intake.PatientAnswerData_MEDIA_SECTION.Enum()
		case *intake.MultipleChoicePatientAnswer:
			paType = intake.PatientAnswerData_MULTIPLE_CHOICE.Enum()
		case *intake.SingleSelectPatientAnswer:
			paType = intake.PatientAnswerData_SINGLE_SELECT.Enum()
		case *intake.SingleEntryPatientAnswer:
			paType = intake.PatientAnswerData_SINGLE_ENTRY.Enum()
		case *intake.SegmentedControlPatientAnswer:
			paType = intake.PatientAnswerData_SEGMENTED_CONTROL.Enum()
		case *intake.AutocompletePatientAnswer:
			paType = intake.PatientAnswerData_AUTOCOMPLETE.Enum()
		case *intake.FreeTextPatientAnswer:
			paType = intake.PatientAnswerData_FREE_TEXT.Enum()
		default:
			t.Fatal("missed answer")
		}

		data, err := proto.Marshal(pb)
		if err != nil {
			t.Fatal(err)
		}

		cad := &intake.PatientAnswerData{
			Type: paType,
			Data: data,
		}

		data, err = proto.Marshal(cad)
		if err != nil {
			t.Fatal(err)
		}

		if err := vm.SetAnswerForQuestion(qa.q.id(), data); err != nil {
			t.Fatal(err)
		}
	}

	b := vm.visit.String()

	outputData, err := ioutil.ReadFile(outputFileName)
	if err != nil {
		t.Fatal(err)
	}

	replacer := strings.NewReplacer(" ", "", "\n", "")
	if bytes.Compare([]byte(replacer.Replace(string(outputData))), []byte(replacer.Replace(b))) != 0 {
		t.Fatalf("End result doesn't match expected output for %s", inputFileName)
	}
}

func testManager_initialState(t *testing.T, inputFileName, outputFileName, gender string) {

	vm := initializeManagerWithVisit(t, inputFileName, []*intake.KeyValuePair{
		{
			Key:   proto.String("gender"),
			Value: proto.String(gender),
		},
	})

	// create a map to hold on to the patient answers so that they can be re-applied
	questionMap := make(map[string]*questionAnswer)
	for _, qData := range vm.questionMap {
		qItem := qData.questionRef

		a, err := qItem.patientAnswer()
		if err == errNoAnswerExists {
			continue
		} else if err != nil {
			t.Fatal(err)
		}

		questionMap[qItem.layoutUnitID()] = &questionAnswer{
			q: qItem,
			a: a,
		}

		// set the answer to nil for each question so that
		// we can replay the answers on the visit
		switch s := qItem.(type) {
		case *freeTextQuestion:
			s.answer = nil
		case *autocompleteQuestion:
			s.answer = nil
		case *mediaQuestion:
			s.answer = nil
		case *multipleChoiceQuestion:
			s.answer = nil
		case *singleEntryQuestion:
			s.answer = nil
		default:
			t.Fatal("missed question")
		}
	}

	// re-initialize the state of layoutUnits based on the fact that we no longer have answers
	// to simulate situation where user is starting from scratch.
	vm.computeCurrentVisibilityState(vm.visit)

	// re-apply the answers to every question.
	for _, qa := range questionMap {
		if err := qa.q.setPatientAnswer(qa.a); err != nil {
			t.Fatal(err)
		}
	}

	// re-compute state to ensure that end result is the same as if the client
	// replayed each answer.
	vm.computeCurrentVisibilityState(vm.visit)

	b := vm.visit.String()

	outputData, err := ioutil.ReadFile(outputFileName)
	if err != nil {
		t.Fatal(err)
	}

	replacer := strings.NewReplacer(" ", "", "\n", "")
	if bytes.Compare([]byte(replacer.Replace(string(outputData))), []byte(replacer.Replace(b))) != 0 {
		t.Fatalf("End result doesn't match expected output for %s", inputFileName)
	}
}

func initializeManagerWithVisit(t *testing.T, fileName string, pairs []*intake.KeyValuePair) *visitManager {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fatal(err)
	}

	vd := &intake.VisitData{
		PatientVisitId: proto.String("10"),
		Layout:         data,
		Pairs:          pairs,
		IsSubmitted:    proto.Bool(false),
		Platform:       intake.VisitData_ANDROID.Enum(),
	}

	vdata, err := proto.Marshal(vd)
	if err != nil {
		t.Fatal(err)
	}

	cli := &mockClientImpl{
		answersSet: map[string]*testAnswerData{},
	}

	vm := &visitManager{}
	if err := vm.Init(vdata, cli); err != nil {
		t.Fatal(err)
	}

	return vm
}
