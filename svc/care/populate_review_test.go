package care

import (
	"testing"

	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/backend/test"
)

func TestAlerts(t *testing.T) {
	// simple intake with questions that require alerting on
	intake := &layout.Intake{
		Sections: []*layout.Section{
			{
				Screens: []*layout.Screen{
					{
						Questions: []*layout.Question{
							{
								ID:                 "test",
								Type:               QuestionTypeMultipleChoice,
								ToAlert:            ptr.Bool(true),
								AlertFormattedText: "Patient picked XXX",
								PotentialAnswers: []*layout.PotentialAnswer{
									{
										ID:      "test1",
										Answer:  "Hi",
										ToAlert: ptr.Bool(true),
									},
									{
										ID:      "test2",
										Answer:  "Hello",
										Summary: "HelloSummary",
										ToAlert: ptr.Bool(true),
									},
									{
										ID:     "test3",
										Answer: "How",
									},
									{
										ID:     "test4",
										Answer: "are",
									},
									{
										ID:      "test5",
										Answer:  "you",
										ToAlert: ptr.Bool(true),
									},
								},
							},
							{
								ID:   "test10",
								Type: QuestionTypeFreeText,
							},
							{
								ID:                 "test20",
								Type:               QuestionTypeAutoComplete,
								ToAlert:            ptr.Bool(true),
								AlertFormattedText: "Patient entered XXX",
							},
						},
					},
				},
			},
		},
	}

	answers := map[string]*Answer{
		"test20": &Answer{
			QuestionID: "test10",
			Answer: &Answer_Autocomplete{
				Autocomplete: &AutocompleteAnswer{
					Items: []*AutocompleteAnswerItem{
						{
							Answer: "1",
						},
						{
							Answer: "2",
						},
						{
							Answer: "3",
						},
					},
				},
			},
		},
		"test10": &Answer{
			QuestionID: "test10",
			Answer: &Answer_FreeText{
				FreeText: &FreeTextAnswer{
					FreeText: "free text response",
				},
			},
		},
		"test": &Answer{
			QuestionID: "test",
			Answer: &Answer_MultipleChoice{
				MultipleChoice: &MultipleChoiceAnswer{
					SelectedAnswers: []*AnswerOption{
						{
							ID: "test1",
						},
						{
							ID: "test2",
						},
						{
							ID: "test4",
						},
						{
							ID: "test5",
						},
					},
				},
			},
		},
	}

	context := visitreview.NewViewContext(nil)
	err := populateAlerts(answers, intake, context)
	test.OK(t, err)
	value, ok := context.Get("visit_alerts")
	test.Equals(t, true, ok)
	test.Equals(t, []string{"Patient picked Hi, HelloSummary and you", "Patient entered 1, 2 and 3"}, value.([]string))
}

func TestAlerts_NoAlerts(t *testing.T) {
	intake := &layout.Intake{
		Sections: []*layout.Section{
			{
				Screens: []*layout.Screen{
					{
						Questions: []*layout.Question{
							{
								ID:   "test10",
								Type: QuestionTypeFreeText,
							},
							{
								ID:   "test20",
								Type: QuestionTypeAutoComplete,
							},
						},
					},
				},
			},
		},
	}

	answers := map[string]*Answer{
		"test20": &Answer{
			QuestionID: "test10",
			Answer: &Answer_Autocomplete{
				Autocomplete: &AutocompleteAnswer{
					Items: []*AutocompleteAnswerItem{
						{
							Answer: "1",
						},
						{
							Answer: "2",
						},
						{
							Answer: "3",
						},
					},
				},
			},
		},
		"test10": &Answer{
			QuestionID: "test10",
			Answer: &Answer_FreeText{
				FreeText: &FreeTextAnswer{
					FreeText: "free text response",
				},
			},
		},
	}

	context := visitreview.NewViewContext(nil)
	err := populateAlerts(answers, intake, context)
	test.OK(t, err)
	value, ok := context.Get("visit_alerts:empty_state_text")
	test.Equals(t, true, ok)
	test.Equals(t, "No alerts", value.(string))

}

func TestPopulateReview_MultipleChoiceQuestion(t *testing.T) {
	question := &layout.Question{
		ID:   "test",
		Type: QuestionTypeMultipleChoice,
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:      "test1",
				Answer:  "Hi",
				ToAlert: ptr.Bool(true),
			},
			{
				ID:      "test2",
				Answer:  "Hello",
				Summary: "HelloSummary",
			},
			{
				ID:     "test3",
				Answer: "How",
			},
			{
				ID:     "test4",
				Answer: "are",
			},
			{
				ID:     "test5",
				Answer: "you",
			},
		},
	}

	answer := &Answer{
		QuestionID: "test",
		Answer: &Answer_MultipleChoice{
			MultipleChoice: &MultipleChoiceAnswer{
				SelectedAnswers: []*AnswerOption{
					{
						ID: "test1",
					},
					{
						ID: "test2",
					},
					{
						ID: "test4",
					},
					{
						ID:       "test5",
						FreeText: "some",
					},
					{
						ID:       "test5",
						FreeText: "text",
					},
				},
			},
		},
	}

	context := visitreview.NewViewContext(nil)
	test.OK(t, builderQuestionWithOptions(question, answer, context))

	answerInContext, ok := context.Get("test:answers")
	test.Equals(t, true, ok)
	test.Equals(t, []visitreview.CheckedUncheckedData{
		{
			Value:     "Hi",
			IsChecked: true,
		},
		{
			Value:     "Hello",
			IsChecked: true,
		},
		{
			Value:     "How",
			IsChecked: false,
		},
		{
			Value:     "are",
			IsChecked: true,
		},
		{
			Value:     "you - some,text",
			IsChecked: true,
		},
	}, answerInContext)
}

func TestPopulateReview_MultipleChoice_Subquestions(t *testing.T) {
	question := &layout.Question{
		ID:   "test",
		Type: QuestionTypeMultipleChoice,
		SubQuestionsConfig: &layout.SubQuestionsConfig{
			Screens: []*layout.Screen{
				{
					Questions: []*layout.Question{
						{
							ID:      "test.a",
							Summary: "TEST.A",
							Type:    QuestionTypeFreeText,
						},
						{
							ID:      "test.b",
							Summary: "TEST.B",
							Type:    QuestionTypeSingleEntry,
						},
						{
							ID:      "test.c",
							Summary: "TEST.C",
							Type:    QuestionTypeSegmentedControl,
							PotentialAnswers: []*layout.PotentialAnswer{
								{
									ID:     "test.c.1",
									Answer: "answer.c.1",
								},
								{
									ID:     "test.c.2",
									Answer: "answer.c.2",
								},
							},
						},
						{
							ID:      "test.d",
							Summary: "TEST.D",
							Type:    QuestionTypeSingleSelect,
							PotentialAnswers: []*layout.PotentialAnswer{
								{
									ID:     "test.d.1",
									Answer: "answer.d.1",
								},
								{
									ID:     "test.d.2",
									Answer: "answer.d.2",
								},
							},
						},
						{
							ID:      "test.e",
							Summary: "TEST.E",
							Type:    QuestionTypeAutoComplete,
						},
						{
							ID:      "test.f",
							Summary: "TEST.F",
							Type:    QuestionTypeMultipleChoice,
							PotentialAnswers: []*layout.PotentialAnswer{
								{
									ID:     "test.f.1",
									Answer: "answer.f.1",
								},
								{
									ID:     "test.f.2",
									Answer: "answer.f.2",
								},
								{
									ID:     "test.f.3",
									Answer: "answer.f.3",
								},
							},
						},
					},
				},
			},
		},
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:      "test1",
				Answer:  "Hi",
				ToAlert: ptr.Bool(true),
			},
			{
				ID:      "test2",
				Answer:  "Hello",
				Summary: "HelloSummary",
			},
			{
				ID:     "test3",
				Answer: "How",
			},
			{
				ID:     "test4",
				Answer: "are",
			},
			{
				ID:     "test5",
				Answer: "you",
			},
		},
	}

	answer := &Answer{
		QuestionID: "test",
		Answer: &Answer_MultipleChoice{
			MultipleChoice: &MultipleChoiceAnswer{
				SelectedAnswers: []*AnswerOption{
					{
						ID: "test1",
						SubAnswers: map[string]*Answer{
							"test.a": &Answer{
								QuestionID: "test.a",
								Answer: &Answer_FreeText{
									FreeText: &FreeTextAnswer{
										FreeText: "FreeText",
									},
								},
							},
							"test.b": &Answer{
								QuestionID: "test.b",
								Answer: &Answer_SingleEntry{
									SingleEntry: &SingleEntryAnswer{
										FreeText: "SingleEntryFreeText",
									},
								},
							},
							"test.c": &Answer{
								QuestionID: "test.c",
								Answer: &Answer_SegmentedControl{
									SegmentedControl: &SegmentedControlAnswer{
										SelectedAnswer: &AnswerOption{
											ID: "test.c.1",
										},
									},
								},
							},
							"test.d": &Answer{
								QuestionID: "test.d",
								Answer: &Answer_SingleSelect{
									SingleSelect: &SingleSelectAnswer{
										SelectedAnswer: &AnswerOption{
											ID: "test.d.2",
										},
									},
								},
							},
							"test.e": &Answer{
								QuestionID: "test.e",
								Answer: &Answer_Autocomplete{
									Autocomplete: &AutocompleteAnswer{
										Items: []*AutocompleteAnswerItem{
											{
												Answer: "answer.e.item1",
											},
											{
												Answer: "answer.e.item2",
											},
										},
									},
								},
							},
							"test.f": &Answer{
								QuestionID: "test.f",
								Answer: &Answer_MultipleChoice{
									MultipleChoice: &MultipleChoiceAnswer{
										SelectedAnswers: []*AnswerOption{
											{
												ID: "test.f.1",
											},
											{
												ID: "test.f.3",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	context := visitreview.NewViewContext(nil)
	test.OK(t, builderQuestionWithSubanswers(question, answer, context))

	answerInContext, ok := context.Get("test:answers")
	test.Equals(t, true, ok)
	test.Equals(t, []visitreview.TitleSubItemsDescriptionContentData{
		{
			Title: "Hi",
			SubItems: []*visitreview.DescriptionContentData{
				{
					Description: "TEST.A",
					Content:     "FreeText",
				},
				{
					Description: "TEST.B",
					Content:     "SingleEntryFreeText",
				},
				{
					Description: "TEST.C",
					Content:     "answer.c.1",
				},
				{
					Description: "TEST.D",
					Content:     "answer.d.2",
				},

				{
					Description: "TEST.E",
					Content:     "answer.e.item1,answer.e.item2",
				},
				{
					Description: "TEST.F",
					Content:     "answer.f.1,answer.f.3",
				},
			},
		},
	}, answerInContext)
}

func TestPopulateReview_SingleSelect(t *testing.T) {
	question := &layout.Question{
		ID:   "test",
		Type: QuestionTypeSingleSelect,
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:     "test1",
				Answer: "Hi",
			},
			{
				ID:      "test2",
				Answer:  "Hello",
				Summary: "HelloSummary",
			},
			{
				ID:     "test3",
				Answer: "How",
			},
			{
				ID:     "test4",
				Answer: "are",
			},
			{
				ID:     "test5",
				Answer: "you",
			},
		},
	}

	answer := &Answer{
		QuestionID: "test",
		Answer: &Answer_SingleSelect{
			SingleSelect: &SingleSelectAnswer{
				SelectedAnswer: &AnswerOption{
					ID: "test2",
				},
			},
		},
	}

	context := visitreview.NewViewContext(nil)
	test.OK(t, builderQuestionWithSingleResponse(question, answer, context))

	answerInContext, ok := context.Get("test:answers")
	test.Equals(t, true, ok)
	test.Equals(t, "HelloSummary", answerInContext)
}

func TestPopulateReview_SegmentedControl(t *testing.T) {
	question := &layout.Question{
		ID:   "test",
		Type: QuestionTypeSegmentedControl,
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:     "test1",
				Answer: "Hi",
			},
			{
				ID:      "test2",
				Answer:  "Hello",
				Summary: "HelloSummary",
			},
			{
				ID:     "test3",
				Answer: "How",
			},
			{
				ID:     "test4",
				Answer: "are",
			},
			{
				ID:     "test5",
				Answer: "you",
			},
		},
	}

	answer := &Answer{
		QuestionID: "test",
		Answer: &Answer_SegmentedControl{
			SegmentedControl: &SegmentedControlAnswer{
				SelectedAnswer: &AnswerOption{
					ID: "test2",
				},
			},
		},
	}

	context := visitreview.NewViewContext(nil)
	test.OK(t, builderQuestionWithSingleResponse(question, answer, context))

	answerInContext, ok := context.Get("test:answers")
	test.Equals(t, true, ok)
	test.Equals(t, "HelloSummary", answerInContext)
}

func TestPopulateReview_FreeText(t *testing.T) {
	question := &layout.Question{
		ID:   "test",
		Type: QuestionTypeFreeText,
	}

	answer := &Answer{
		QuestionID: "test",
		Answer: &Answer_FreeText{
			FreeText: &FreeTextAnswer{
				FreeText: "FreeText",
			},
		},
	}

	context := visitreview.NewViewContext(nil)
	test.OK(t, builderQuestionFreeText(question, answer, context))

	answerInContext, ok := context.Get("test:answers")
	test.Equals(t, true, ok)
	test.Equals(t, "FreeText", answerInContext)
}

func TestPopulateReview_SingleEntry(t *testing.T) {
	question := &layout.Question{
		ID:   "test",
		Type: QuestionTypeSingleEntry,
	}

	answer := &Answer{
		QuestionID: "test",
		Answer: &Answer_SingleEntry{
			SingleEntry: &SingleEntryAnswer{
				FreeText: "FreeText",
			},
		},
	}

	context := visitreview.NewViewContext(nil)
	test.OK(t, builderQuestionFreeText(question, answer, context))

	answerInContext, ok := context.Get("test:answers")
	test.Equals(t, true, ok)
	test.Equals(t, "FreeText", answerInContext)
}

func TestPopulateReview_PhotoSlots(t *testing.T) {
	question := &layout.Question{
		ID:   "test",
		Type: QuestionTypePhotoSection,
		PhotoSlots: []*layout.PhotoSlot{
			{
				ID:   "slot1",
				Name: "slot1Name",
			},
			{
				ID:   "slot2",
				Name: "slot2Name",
			},
			{
				ID:   "slot3",
				Name: "slot3Name",
			},
		},
	}

	answer := &Answer{
		QuestionID: "test",
		Answer: &Answer_PhotoSection{
			PhotoSection: &PhotoSectionAnswer{
				Sections: []*PhotoSectionAnswer_PhotoSectionItem{
					{
						Name: "Section1",
						Slots: []*PhotoSectionAnswer_PhotoSectionItem_PhotoSlotItem{
							{
								SlotID:  "slot1",
								MediaID: "1",
								Name:    "slot1Name",
								URL:     "https://placekitten.com/600/800",
							},
							{
								SlotID:  "slot2",
								MediaID: "2",
								Name:    "slot2Name",
								URL:     "https://placekitten.com/600/800",
							},
						},
					},
					{
						Name: "Section2",
						Slots: []*PhotoSectionAnswer_PhotoSectionItem_PhotoSlotItem{
							{
								SlotID:  "slot1",
								MediaID: "3",
								Name:    "slot1Name",
								URL:     "https://placekitten.com/600/800",
							},
						},
					},
				},
			},
		},
	}

	context := visitreview.NewViewContext(nil)
	test.OK(t, builderQuestionWithPhotoSlots(question, answer, context))

	answerInContext, ok := context.Get("test:photos")
	test.Equals(t, true, ok)
	test.Equals(t, []visitreview.TitlePhotoListData{
		{
			Title: "Section1",
			Photos: []visitreview.PhotoData{
				{
					Title:    "slot1Name",
					PhotoID:  "1",
					PhotoURL: "https://placekitten.com/600/800",
				},
				{
					Title:    "slot2Name",
					PhotoID:  "2",
					PhotoURL: "https://placekitten.com/600/800",
				},
			},
		},
		{
			Title: "Section2",
			Photos: []visitreview.PhotoData{
				{
					Title:    "slot1Name",
					PhotoID:  "3",
					PhotoURL: "https://placekitten.com/600/800",
				},
			},
		},
	}, answerInContext)
}
