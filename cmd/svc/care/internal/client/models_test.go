package client

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/svc/layout"
	"github.com/sprucehealth/mapstructure"
)

func TestParsing(t *testing.T) {

	var questionToAnswerMap map[string]Answer
	decoderConfig := &mapstructure.DecoderConfig{
		Result:   &questionToAnswerMap,
		TagName:  "json",
		Registry: *typeRegistry,
	}

	d, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		t.Fatal(err)
	}

	fileData, err := ioutil.ReadFile("testdata/answers.json")
	if err != nil {
		t.Fatal(err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(fileData, &jsonMap); err != nil {
		t.Fatal(err)
	}

	if err := d.Decode(jsonMap); err != nil {
		t.Fatal(err)
	}

	// lets unmarshal decoded object and ensure it is the same as the content
	// in the file
	jsonData, err := json.Marshal(questionToAnswerMap)
	if err != nil {
		t.Fatal(err)
	}

	if strings.TrimSpace(string(jsonData)) != strings.TrimSpace(string(fileData)) {
		t.Fatalf("Expected:\n '%s'\n\nGot:\n '%s'", string(fileData), string(jsonData))
	}
}

func TestPhotoSectionAnswer(t *testing.T) {
	p := &PhotoQuestionAnswer{
		PhotoSections: []*PhotoSectionItem{
			{
				Name: "Test",
				Slots: []*PhotoSlotItem{
					{
						SlotID:  "10",
						PhotoID: "100",
					},
				},
			},
		},
	}

	err := p.Validate(&layout.Question{
		Type: "q_type_photo_section",
		PhotoSlots: []*layout.PhotoSlot{
			{
				ID: "10",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	p = &PhotoQuestionAnswer{
		PhotoSections: []*PhotoSectionItem{
			{
				Name: "Test",
				Slots: []*PhotoSlotItem{
					{
						SlotID:  "10",
						PhotoID: "100",
					},
				},
			},
			{
				Name: "Test23",
				Slots: []*PhotoSlotItem{
					{
						SlotID:  "10",
						PhotoID: "101",
					},
				},
			},
		},
	}

	err = p.Validate(&layout.Question{
		Type: "q_type_photo_section",
		AdditionalFields: &layout.QuestionAdditionalFields{
			AllowsMultipleSections: ptr.Bool(true),
		},
		PhotoSlots: []*layout.PhotoSlot{
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

	p := &PhotoQuestionAnswer{
		PhotoSections: []*PhotoSectionItem{
			{
				Slots: []*PhotoSlotItem{
					{
						SlotID:  "10",
						PhotoID: "100",
					},
				},
			},
		},
	}

	err := p.Validate(&layout.Question{
		Type: "q_type_photo_section",
		PhotoSlots: []*layout.PhotoSlot{
			{
				ID: "10",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error but did not get one")
	}

	// slot id that does not exist

	p = &PhotoQuestionAnswer{
		PhotoSections: []*PhotoSectionItem{
			{
				Name: "Test",
				Slots: []*PhotoSlotItem{
					{
						SlotID:  "11",
						PhotoID: "100",
					},
				},
			},
		},
	}

	err = p.Validate(&layout.Question{
		Type: "q_type_photo_section",
		PhotoSlots: []*layout.PhotoSlot{
			{
				ID: "10",
			},
		},
	})
	if err == nil {
		t.Fatalf("expected error but did not get one")
	}

	//	 required slot not filled

	p = &PhotoQuestionAnswer{
		PhotoSections: []*PhotoSectionItem{
			{
				Name: "Test",
				Slots: []*PhotoSlotItem{
					{
						SlotID:  "10",
						PhotoID: "100",
					},
				},
			},
		},
	}

	err = p.Validate(&layout.Question{
		Type: "q_type_photo_section",
		PhotoSlots: []*layout.PhotoSlot{
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
					"sq1": &FreeTextQuestionAnswer{
						Type: "q_type_free_text",
						Text: "hello",
					},
					"sq2": &SegmentedControlQuestionAnswer{
						Type: "q_type_segmented_control",
						PotentialAnswer: &PotentialAnswerItem{
							ID: "100",
						},
					},
				},
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
		SubQuestionsConfig: &layout.SubQuestionsConfig{
			Screens: []*layout.Screen{
				{
					Questions: []*layout.Question{
						{
							ID:   "sq1",
							Type: "q_type_free_text",
						},
						{
							ID:   "sq2",
							Type: "q_type_segmented_control",
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

func TestMultipleChoiceQuestionWithSubanswer_Invalid(t *testing.T) {

	// subanswers for option missing

	m := &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID: "10",
				Subanswers: map[string]Answer{
					"sq1": &FreeTextQuestionAnswer{
						Type: "q_type_free_text",
						Text: "hello",
					},
					"sq2": &SegmentedControlQuestionAnswer{
						Type: "q_type_segmented_control",
						PotentialAnswer: &PotentialAnswerItem{
							ID: "100",
						},
					},
				},
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
		SubQuestionsConfig: &layout.SubQuestionsConfig{
			Screens: []*layout.Screen{
				{
					Questions: []*layout.Question{
						{
							ID:       "sq1",
							Type:     "q_type_free_text",
							Required: ptr.Bool(true),
						},
						{
							ID:       "sq2",
							Type:     "q_type_segmented_control",
							Required: ptr.Bool(true),
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
	if err == nil {
		t.Fatalf("expected error but got none")
	}

	// required subanswer missing

	m = &MultipleChoiceQuestionAnswer{
		PotentialAnswers: []*PotentialAnswerItem{
			{
				ID: "10",
				Subanswers: map[string]Answer{
					"sq1": &FreeTextQuestionAnswer{
						Type: "q_type_free_text",
						Text: "hello",
					},
				},
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
		SubQuestionsConfig: &layout.SubQuestionsConfig{
			Screens: []*layout.Screen{
				{
					Questions: []*layout.Question{
						{
							ID:       "sq1",
							Type:     "q_type_free_text",
							Required: ptr.Bool(true),
						},
						{
							ID:       "sq2",
							Type:     "q_type_segmented_control",
							Required: ptr.Bool(true),
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
	if err == nil {
		t.Fatalf("expected error but got none")
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
