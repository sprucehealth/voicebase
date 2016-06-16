package server

import (
	"testing"

	"github.com/sprucehealth/backend/cmd/svc/care/internal/client"
	"github.com/sprucehealth/backend/cmd/svc/care/internal/models"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/media"
	mediamock "github.com/sprucehealth/backend/svc/media/mock"
)

func TestTransformToModel_MediaSection(t *testing.T) {

	mmock := mediamock.New(t)
	defer mmock.Finish()

	mmock.Expect(mock.NewExpectation(mmock.MediaInfos, &media.MediaInfosRequest{
		MediaIDs: []string{"PhotoID1"},
	}).WithReturns(&media.MediaInfosResponse{
		MediaInfos: map[string]*media.MediaInfo{
			"PhotoID1": &media.MediaInfo{
				ID:       "PhotoID1",
				URL:      "photo1.url",
				ThumbURL: "thumbnail1.url",
				MIME: &media.MIME{
					Type: "image",
				},
			},
		},
	}, nil))

	a, err := transformAnswerToModel("10", &client.MediaQuestionAnswer{
		Type: "q_type_media_section",
		Sections: []*client.MediaSectionItem{
			{
				Name: "SectionName",
				Slots: []*client.MediaSlotItem{
					{
						Name:    "SlotName",
						SlotID:  "SlotID1",
						MediaID: "PhotoID1",
					},
				},
			},
		},
	}, mmock)
	if err != nil {
		t.Fatal(err)
	}
	if a.GetMediaSection() == nil {
		t.Fatalf("expected media section to be populated but it wasn't")
	}
	test.Equals(t, &models.MediaSectionAnswer{
		Sections: []*models.MediaSectionAnswer_MediaSectionItem{
			{
				Name: "SectionName",
				Slots: []*models.MediaSectionAnswer_MediaSectionItem_MediaSlotItem{
					{
						Name:    "SlotName",
						SlotID:  "SlotID1",
						MediaID: "PhotoID1",
						Type:    models.MediaType_IMAGE,
					},
				},
			},
		},
	}, a.GetMediaSection())
}

func TestTransformModel_FreeText(t *testing.T) {
	mmock := mediamock.New(t)
	defer mmock.Finish()

	a, err := transformAnswerToModel("10", &client.FreeTextQuestionAnswer{
		Type: "q_type_free_text",
		Text: "hello",
	}, mmock)
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, &models.FreeTextAnswer{
		FreeText: "hello",
	}, a.GetFreeText())
}

func TestTransformModel_SingleEntry(t *testing.T) {
	mmock := mediamock.New(t)
	defer mmock.Finish()

	a, err := transformAnswerToModel("10", &client.SingleEntryQuestionAnswer{
		Type: "q_type_single_entry",
		Text: "hello",
	}, mmock)
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, &models.SingleEntryAnswer{
		FreeText: "hello",
	}, a.GetSingleEntry())
}

func TestTransformModel_SingleSelect(t *testing.T) {
	mmock := mediamock.New(t)
	defer mmock.Finish()

	a, err := transformAnswerToModel("10", &client.SingleSelectQuestionAnswer{
		Type: "q_type_single_select",
		PotentialAnswer: &client.PotentialAnswerItem{
			ID:   "100",
			Text: "hello",
		},
	}, mmock)
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, &models.SingleSelectAnswer{
		SelectedAnswer: &models.AnswerOption{
			ID:       "100",
			FreeText: "hello",
		},
	}, a.GetSingleSelect())
}

func TestTransformModel_MultipleChoice(t *testing.T) {
	mmock := mediamock.New(t)
	defer mmock.Finish()

	a, err := transformAnswerToModel("10", &client.MultipleChoiceQuestionAnswer{
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
				ID:   "200",
				Text: "hello2",
			},
		},
	}, mmock)
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, &models.MultipleChoiceAnswer{
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
	}, a.GetMultipleChoice())
}

func TestTransformModel_Autocomplete(t *testing.T) {
	mmock := mediamock.New(t)
	defer mmock.Finish()

	a, err := transformAnswerToModel("10", &client.AutocompleteQuestionAnswer{
		Type: "q_type_multiple_choice",
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
				Text: "hello2",
			},
		},
	}, mmock)
	if err != nil {
		t.Fatal(err)
	}
	test.Equals(t, &models.AutocompleteAnswer{
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
	}, a.GetAutocomplete())
}
