package feedback

import (
	"testing"

	"github.com/sprucehealth/backend/common"
)

type mockFeedbackClient_homecard struct {
	DAL
	pf               *PatientFeedback
	ratingConfigs    map[int]string
	tagToFeedbackMap map[string]*FeedbackTemplateData
}

func (m *mockFeedbackClient_homecard) PatientFeedback(feedbackFor string) (*PatientFeedback, error) {
	return m.pf, nil
}
func (m *mockFeedbackClient_homecard) RatingConfigs() (map[int]string, error) {
	return m.ratingConfigs, nil
}
func (m *mockFeedbackClient_homecard) ActiveFeedbackTemplate(tag string) (*FeedbackTemplateData, error) {
	return m.tagToFeedbackMap[tag], nil
}

func TestHomeCard_NoPendingPatientFeedback(t *testing.T) {
	m := &mockFeedbackClient_homecard{
		pf: &PatientFeedback{},
	}

	hc, err := HomeCardForCase(m, 10, common.IOS)
	if err != nil {
		t.Fatal(err)
	} else if hc != nil {
		t.Fatal("Expected no home card but got one")
	}

	// test feedback that has been dismissed
	m.pf.Dismissed = true

	hc, err = HomeCardForCase(m, 10, common.IOS)
	if err != nil {
		t.Fatal(err)
	} else if hc != nil {
		t.Fatal("Expected no home card but got one")
	}
}

func TestHomeCard_NoRatingConfig(t *testing.T) {
	m := &mockFeedbackClient_homecard{
		pf: &PatientFeedback{
			Pending: true,
		},
	}
	hc, err := HomeCardForCase(m, 10, common.IOS)
	if err != nil {
		t.Fatal(err)
	}

	fhc, ok := hc.(*feedbackHomeCard)
	if !ok {
		t.Fatalf("Expected FeedbackHomeCard instead got %T", fhc)
	}

	if len(fhc.FeedbackPrompt.PromptByID) > 0 {
		t.Fatalf("Expected no prompt configurations instead got %d", len(fhc.FeedbackPrompt.PromptByID))
	}
	if len(fhc.FeedbackPrompt.RatingConfig) > 0 {
		t.Fatalf("Expected no promopts configured for any rating instead got %d", len(fhc.FeedbackPrompt.RatingConfig))
	}
}

func TestHomeCard_WithRatingConfig(t *testing.T) {
	m := &mockFeedbackClient_homecard{
		pf: &PatientFeedback{
			Pending: true,
		},
		ratingConfigs: map[int]string{
			1: "yelp,appstore",
			2: "bad_freetext",
			3: "bad_freetext",
			5: "good_freetext",
		},
		tagToFeedbackMap: map[string]*FeedbackTemplateData{
			"yelp": &FeedbackTemplateData{
				ID:   1,
				Tag:  "yelp",
				Type: FTOpenURL,
				Template: &OpenURLTemplate{
					Title:    "testing",
					BodyText: "body_text",
					AndroidURL: URL{
						IconURL: "android",
						OpenURL: "open_url",
					},
				},
			},
			"appstore": &FeedbackTemplateData{
				ID:   2,
				Tag:  "appstore",
				Type: FTOpenURL,
				Template: &OpenURLTemplate{
					Title:    "testing",
					BodyText: "body_text",
					AndroidURL: URL{
						IconURL: "android",
						OpenURL: "open_url",
					},
				},
			},
			"bad_freetext": &FeedbackTemplateData{
				ID:   3,
				Tag:  "bad_freetext",
				Type: FTFreetext,
				Template: &FreeTextTemplate{
					Title:           "free_text",
					PlaceholderText: "placeholder_text",
					ButtonTitle:     "button_title",
				},
			},
			"good_freetext": &FeedbackTemplateData{
				ID:   4,
				Tag:  "good_freetext",
				Type: FTFreetext,
				Template: &FreeTextTemplate{
					Title:           "free_text",
					PlaceholderText: "placeholder_text",
					ButtonTitle:     "button_title",
				},
			},
		},
	}

	hc, err := HomeCardForCase(m, 10, common.Android)
	if err != nil {
		t.Fatal(err)
	}

	fhc := hc.(*feedbackHomeCard)

	// ensure the config setup is as expected
	if len(fhc.FeedbackPrompt.RatingConfig["1"]) != 2 {
		t.Fatalf("Expected 2 templates but got %d", len(fhc.FeedbackPrompt.RatingConfig["1"]))
	}
	if len(fhc.FeedbackPrompt.RatingConfig["2"]) != 1 {
		t.Fatalf("Expected 1 templates but got %d", len(fhc.FeedbackPrompt.RatingConfig["2"]))
	}
	if len(fhc.FeedbackPrompt.RatingConfig["3"]) != 1 {
		t.Fatalf("Expected 1 templates but got %d", len(fhc.FeedbackPrompt.RatingConfig["3"]))
	}
	if len(fhc.FeedbackPrompt.RatingConfig["4"]) != 0 {
		t.Fatalf("Expected 0 templates but got %d", len(fhc.FeedbackPrompt.RatingConfig["4"]))
	}
	if len(fhc.FeedbackPrompt.RatingConfig["5"]) != 1 {
		t.Fatalf("Expected 1 templates but got %d", len(fhc.FeedbackPrompt.RatingConfig["5"]))
	}

	// ensure the prompts are setup as expected
	if fhc.FeedbackPrompt.PromptByID["1"] == nil {
		t.Fatalf("Expected prompt for id 1 but got none")
	}
	if fhc.FeedbackPrompt.PromptByID["2"] == nil {
		t.Fatalf("Expected prompt for id 2 but got none")
	}
	if fhc.FeedbackPrompt.PromptByID["3"] == nil {
		t.Fatalf("Expected prompt for id 3 but got none")
	}
	if fhc.FeedbackPrompt.PromptByID["4"] == nil {
		t.Fatalf("Expected prompt for id 4 but got none")
	}
}
