package server

import (
	"testing"

	"github.com/sprucehealth/backend/test"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/client"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
)

func TestTransformToResponse_PhotoSection(t *testing.T) {
	a, err := transformAnswerModelToResponse(&models.Answer{
		Type:       "q_type_photo_section",
		QuestionID: "10",
		Answer: &models.Answer_PhotoSection{
			PhotoSection: &models.PhotoSectionAnswer{
				Sections: []*models.PhotoSectionAnswer_PhotoSectionItem{
					{
						Name: "SectionName",
						Slots: []*models.PhotoSectionAnswer_PhotoSectionItem_PhotoSlotItem{
							{
								Name:    "SlotName",
								SlotID:  "SlotID1",
								MediaID: "PhotoID1",
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, &client.PhotoQuestionAnswer{
		Type: "q_type_photo_section",
		PhotoSections: []*client.PhotoSectionItem{
			{
				Name: "SectionName",
				Slots: []*client.PhotoSlotItem{
					{
						Name:     "SlotName",
						SlotID:   "SlotID1",
						PhotoID:  "PhotoID1",
						PhotoURL: "https://placekitten.com/600/800",
					},
				},
			},
		},
	}, a)
}

func TestTransformToResponse_FreeText(t *testing.T) {
	a, err := transformAnswerModelToResponse(&models.Answer{
		Type:       "q_type_free_text",
		QuestionID: "10",
		Answer: &models.Answer_FreeText{
			FreeText: &models.FreeTextAnswer{
				FreeText: "hello",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, &client.FreeTextQuestionAnswer{
		Type: "q_type_free_text",
		Text: "hello",
	}, a)
}

func TestTransformToResponse_SingleEntry(t *testing.T) {
	a, err := transformAnswerModelToResponse(&models.Answer{
		Type:       "q_type_single_entry",
		QuestionID: "10",
		Answer: &models.Answer_SingleEntry{
			SingleEntry: &models.SingleEntryAnswer{
				FreeText: "hello",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, &client.SingleEntryQuestionAnswer{
		Type: "q_type_single_entry",
		Text: "hello",
	}, a)
}

func TestTransformToResponse_SingleSelect(t *testing.T) {
	a, err := transformAnswerModelToResponse(&models.Answer{
		Type:       "q_type_single_select",
		QuestionID: "10",
		Answer: &models.Answer_SingleSelect{
			SingleSelect: &models.SingleSelectAnswer{
				SelectedAnswer: &models.AnswerOption{
					ID:       "100",
					FreeText: "hello",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, &client.SingleSelectQuestionAnswer{
		Type: "q_type_single_select",
		PotentialAnswer: &client.PotentialAnswerItem{
			ID:   "100",
			Text: "hello",
		},
	}, a)
}

func TestTransformToResponse_MultipleChoice(t *testing.T) {
	a, err := transformAnswerModelToResponse(&models.Answer{
		Type:       "q_type_multiple_choice",
		QuestionID: "10",
		Answer: &models.Answer_MultipleChoice{
			MultipleChoice: &models.MultipleChoiceAnswer{
				SelectedAnswers: []*models.AnswerOption{
					{
						ID:       "100",
						FreeText: "hello",
						SubAnswers: map[string]*models.Answer{
							"101": &models.Answer{
								QuestionID: "101",
								Type:       "q_type_free_text",
								Answer: &models.Answer_FreeText{
									FreeText: &models.FreeTextAnswer{
										FreeText: "hellosup",
									},
								},
							},
							"102": &models.Answer{
								QuestionID: "102",
								Type:       "q_type_segmented_control",
								Answer: &models.Answer_SegmentedControl{
									SegmentedControl: &models.SegmentedControlAnswer{
										SelectedAnswer: &models.AnswerOption{
											ID:       "102.a",
											FreeText: "hellosup",
										},
									},
								},
							},
						},
					},
					{
						ID:         "200",
						FreeText:   "hello2",
						SubAnswers: map[string]*models.Answer{},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, &client.MultipleChoiceQuestionAnswer{
		Type: "q_type_multiple_choice",
		PotentialAnswers: []*client.PotentialAnswerItem{
			{
				ID:   "100",
				Text: "hello",
				Subanswers: map[string]client.Answer{
					"101": &client.FreeTextQuestionAnswer{
						Type: "q_type_free_text",
						Text: "hellosup",
					},
					"102": &client.SegmentedControlQuestionAnswer{
						Type: "q_type_segmented_control",
						PotentialAnswer: &client.PotentialAnswerItem{
							ID:   "102.a",
							Text: "hellosup",
						},
					},
				},
			},
			{
				ID:         "200",
				Text:       "hello2",
				Subanswers: map[string]client.Answer{},
			},
		},
	}, a)
}

func TestTransformToResponse_AutoComplete(t *testing.T) {
	a, err := transformAnswerModelToResponse(&models.Answer{
		Type:       "q_type_autocomplete",
		QuestionID: "10",
		Answer: &models.Answer_Autocomplete{
			Autocomplete: &models.AutocompleteAnswer{
				Items: []*models.AutocompleteAnswerItem{
					{
						Answer: "hello",
						SubAnswers: map[string]*models.Answer{
							"101": &models.Answer{
								QuestionID: "101",
								Type:       "q_type_free_text",
								Answer: &models.Answer_FreeText{
									FreeText: &models.FreeTextAnswer{
										FreeText: "hellosup",
									},
								},
							},
							"102": &models.Answer{
								QuestionID: "102",
								Type:       "q_type_segmented_control",
								Answer: &models.Answer_SegmentedControl{
									SegmentedControl: &models.SegmentedControlAnswer{
										SelectedAnswer: &models.AnswerOption{
											ID:       "102.a",
											FreeText: "hellosup",
										},
									},
								},
							},
						},
					},
					{
						Answer:     "hello2",
						SubAnswers: map[string]*models.Answer{},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, &client.AutocompleteQuestionAnswer{
		Type: "q_type_autocomplete",
		Answers: []*client.AutocompleteItem{
			{
				Text: "hello",
				Subanswers: map[string]client.Answer{
					"101": &client.FreeTextQuestionAnswer{
						Type: "q_type_free_text",
						Text: "hellosup",
					},
					"102": &client.SegmentedControlQuestionAnswer{
						Type: "q_type_segmented_control",
						PotentialAnswer: &client.PotentialAnswerItem{
							ID:   "102.a",
							Text: "hellosup",
						},
					},
				},
			},
			{
				Text:       "hello2",
				Subanswers: map[string]client.Answer{},
			},
		},
	}, a)
}
