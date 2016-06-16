package client

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/test"
	"github.com/sprucehealth/backend/svc/layout"
)

func TestParsing(t *testing.T) {
	fileData, err := ioutil.ReadFile("testdata/answers.json")
	if err != nil {
		t.Fatal(err)
	}

	visitAnswers, err := Decode(string(fileData))
	if err != nil {
		t.Fatal(err)
	}

	// lets unmarshal decoded object and ensure it is the same as the content
	// in the file
	jsonData, err := json.Marshal(visitAnswers)
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(string(jsonData)) != strings.TrimSpace(string(fileData)) {
		t.Fatalf("Expected:\n '%s'\n\nGot:\n '%s'", string(fileData), string(jsonData))
	}
}

func TestVisitAnswers_NilAnswers(t *testing.T) {
	v := &VisitAnswers{
		Answers: map[string]Answer{
			"10": &FreeTextQuestionAnswer{
				Text: "hello",
			},
			"11": nil,
		},
	}
	v.DeleteNilAnswers()

	test.Equals(t, &VisitAnswers{
		Answers: map[string]Answer{
			"10": &FreeTextQuestionAnswer{
				Text: "hello",
			},
		},
	}, v)

}

func TestPhotoSectionAnswer(t *testing.T) {
	p := &MediaQuestionAnswer{
		Sections: []*MediaSectionItem{
			{
				Name: "Test",
				Slots: []*MediaSlotItem{
					{
						SlotID:  "10",
						MediaID: "100",
					},
				},
			},
		},
	}

	err := p.Validate(&layout.Question{
		Type: "q_type_media_section",
		MediaSlots: []*layout.MediaSlot{
			{
				ID: "10",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	p = &MediaQuestionAnswer{
		Sections: []*MediaSectionItem{
			{
				Name: "Test",
				Slots: []*MediaSlotItem{
					{
						SlotID:  "10",
						MediaID: "100",
					},
				},
			},
			{
				Name: "Test23",
				Slots: []*MediaSlotItem{
					{
						SlotID:  "10",
						MediaID: "101",
					},
				},
			},
		},
	}

	err = p.Validate(&layout.Question{
		Type: "q_type_media_section",
		AdditionalFields: &layout.QuestionAdditionalFields{
			AllowsMultipleSections: ptr.Bool(true),
		},
		MediaSlots: []*layout.MediaSlot{
			{
				ID: "10",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPhotoSectionAnswer_Invalid(t *testing.T) {

	// missing photo section name

	p := &MediaQuestionAnswer{
		Sections: []*MediaSectionItem{
			{
				Slots: []*MediaSlotItem{
					{
						SlotID:  "10",
						MediaID: "100",
					},
				},
			},
		},
	}

	err := p.Validate(&layout.Question{
		Type: "q_type_media_section",
		MediaSlots: []*layout.MediaSlot{
			{
				ID: "10",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error but did not get one")
	}

	// slot id that does not exist

	p = &MediaQuestionAnswer{
		Sections: []*MediaSectionItem{
			{
				Name: "Test",
				Slots: []*MediaSlotItem{
					{
						SlotID:  "11",
						MediaID: "100",
					},
				},
			},
		},
	}

	err = p.Validate(&layout.Question{
		Type: "q_type_media_section",
		MediaSlots: []*layout.MediaSlot{
			{
				ID: "10",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error but did not get one")
	}

	//	 required slot not filled

	p = &MediaQuestionAnswer{
		Sections: []*MediaSectionItem{
			{
				Name: "Test",
				Slots: []*MediaSlotItem{
					{
						SlotID:  "10",
						MediaID: "100",
					},
				},
			},
		},
	}

	err = p.Validate(&layout.Question{
		Type: "q_type_media_section",
		MediaSlots: []*layout.MediaSlot{
			{
				ID: "10",
			},
			{
				ID:       "11",
				Required: ptr.Bool(true),
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error but did not get one")
	}
}

func TestMultipleChoiceQuestion(t *testing.T) {
	m := &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID: "10",
			},
			{
				ID: "11",
			},
		},
	}

	err := m.Validate(&layout.Question{
		Type: "q_type_multiple_choice",
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:   "10",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "11",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "12",
				Type: "a_type_multiple_choice_none",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	m = &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID: "12",
			},
		},
	}

	err = m.Validate(&layout.Question{
		Type: "q_type_multiple_choice",
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:   "10",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "11",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "12",
				Type: "a_type_multiple_choice_none",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	m = &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID:   "11",
				Text: "HEHE",
			},
		},
	}

	err = m.Validate(&layout.Question{
		Type: "q_type_multiple_choice",
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:   "10",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "11",
				Type: "a_type_multiple_choice_other_free_text",
			},
			{
				ID:   "12",
				Type: "a_type_multiple_choice_none",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultipleChoiceQuestion_Invalid(t *testing.T) {
	m := &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID: "10",
			},
			{
				ID: "13",
			},
		},
	}

	err := m.Validate(&layout.Question{
		Type: "q_type_multiple_choice",
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:   "10",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "11",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "12",
				Type: "a_type_multiple_choice_none",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error but got none")
	}

	m = &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID:   "11",
				Text: "test",
			},
		},
	}

	err = m.Validate(&layout.Question{
		Type: "q_type_multiple_choice",
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:   "10",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "11",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "12",
				Type: "a_type_multiple_choice_none",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error but got none")
	}

	m = &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID: "10",
			},
			{
				ID: "12",
			},
		},
	}

	err = m.Validate(&layout.Question{
		Type: "q_type_multiple_choice",
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:   "10",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "11",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "12",
				Type: "a_type_multiple_choice_none",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error but got none")
	}

	m = &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID: "10",
			},
			{
				ID:   "11",
				Text: "SUP",
			},
		},
	}

	err = m.Validate(&layout.Question{
		Type: "q_type_multiple_choice",
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:   "10",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "11",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "12",
				Type: "a_type_multiple_choice_none",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error but got none")
	}
}

func TestMultipleChoiceQuestionWithSubanswers(t *testing.T) {
	m := &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID: "10",
				Subanswers: map[string]Answer{
					"sq2": &SegmentedControlQuestionAnswer{
						Type: "q_type_segmented_control",
						PotentialAnswer: &PotentialAnswerItem{
							ID: "100",
						},
					},
					"sq1": &FreeTextQuestionAnswer{
						Type: "q_type_free_text",
						Text: "hello",
					},
				},
			},
		},
	}

	err := m.Validate(&layout.Question{
		Type:     "q_type_multiple_choice",
		Required: ptr.Bool(true),
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:   "10",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "11",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "12",
				Type: "a_type_multiple_choice_none",
			},
		},
		SubQuestionsConfig: &layout.SubQuestionsConfig{
			Screens: []*layout.Screen{
				{
					Questions: []*layout.Question{
						{
							ID:       "sq1",
							Required: ptr.Bool(true),
							Type:     "q_type_free_text",
						},
						{
							ID:       "sq2",
							Required: ptr.Bool(true),
							Type:     "q_type_segmented_control",
							PotentialAnswers: []*layout.PotentialAnswer{
								{
									ID: "100",
								},
								{
									ID: "101",
								},
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
}

func TestMultipleChoiceQuestionWithSubanswers_NilAnswers(t *testing.T) {
	m := &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID: "10",
				Subanswers: map[string]Answer{
					"sq2": &SegmentedControlQuestionAnswer{
						Type: "q_type_segmented_control",
						PotentialAnswer: &PotentialAnswerItem{
							ID: "100",
						},
					},
					"sq3": nil,
				},
			},
		},
	}

	err := m.Validate(&layout.Question{
		Type:     "q_type_multiple_choice",
		Required: ptr.Bool(true),
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID:   "10",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "11",
				Type: "a_type_multiple_choice",
			},
			{
				ID:   "12",
				Type: "a_type_multiple_choice_none",
			},
		},
		SubQuestionsConfig: &layout.SubQuestionsConfig{
			Screens: []*layout.Screen{
				{
					Questions: []*layout.Question{
						{
							ID:       "sq1",
							Required: ptr.Bool(true),
							Type:     "q_type_free_text",
						},
						{
							ID:       "sq2",
							Required: ptr.Bool(true),
							Type:     "q_type_segmented_control",
							PotentialAnswers: []*layout.PotentialAnswer{
								{
									ID: "100",
								},
								{
									ID: "101",
								},
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
}

func TestSingleSelectAnswer(t *testing.T) {
	s := &SingleSelectQuestionAnswer{
		PotentialAnswer: &PotentialAnswerItem{
			ID: "10",
		},
	}

	err := s.Validate(&layout.Question{
		Type: "q_type_single_select",
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID: "10",
			},
			{
				ID: "11",
			},
		},
	})
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestSingleSelectAnswer_Invalid(t *testing.T) {
	s := &SingleSelectQuestionAnswer{
		PotentialAnswer: &PotentialAnswerItem{
			ID: "15",
		},
	}

	err := s.Validate(&layout.Question{
		Type: "q_type_single_select",
		PotentialAnswers: []*layout.PotentialAnswer{
			{
				ID: "10",
			},
			{
				ID: "11",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error but got none")
	}
}

func TestFreeTextQuestion(t *testing.T) {
	f := &FreeTextQuestionAnswer{
		Text: "HELLO",
	}

	err := f.Validate(&layout.Question{
		Type: "q_type_free_text",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFreeTextQuestion_Invalid(t *testing.T) {
	f := &FreeTextQuestionAnswer{
		Text: "HELLO",
	}

	err := f.Validate(&layout.Question{
		Type: "q_type_multiple_choice",
	})
	if err == nil {
		t.Fatalf("expected error but got none")
	}

	f = &FreeTextQuestionAnswer{
		Text: "",
	}

	err = f.Validate(&layout.Question{
		Type:     "q_type_multiple_choice",
		Required: ptr.Bool(true),
	})
	if err == nil {
		t.Fatalf("expected error but got none")
	}
}

func TestAutocomplete(t *testing.T) {
	a := &AutocompleteQuestionAnswer{
		Answers: []*AutocompleteItem{
			{
				Text: "HI",
			},
		},
	}

	err := a.Validate(&layout.Question{
		Type: "q_type_autocomplete",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAutocomplete_Invalid(t *testing.T) {
	a := &AutocompleteQuestionAnswer{
		Answers: []*AutocompleteItem{
			{
				Text: "HI",
			},
			{},
		},
	}

	err := a.Validate(&layout.Question{
		Type: "q_type_autocomplete",
	})
	if err == nil {
		t.Fatalf("error expected but got none")
	}
}
