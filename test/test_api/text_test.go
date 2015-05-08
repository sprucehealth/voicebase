package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestLocalizedText(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	tags := []string{
		"txt_feedback_screen_title",
		"txt_allergic_medications",
	}
	text, err := testData.DataAPI.LocalizedText(api.LanguageIDEnglish, tags)
	test.OK(t, err)
	test.Equals(t, len(tags), len(text))
	for _, tag := range tags {
		if text[tag] == "" {
			t.Fatalf("No text found for tag '%s'", tag)
		}
	}
}

func TestUpdateLocalizedText(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close()

	tags := []string{
		"txt_feedback_screen_title",
		"txt_allergic_medications",
	}
	text := make(map[string]string, len(tags))
	for _, tag := range tags {
		text[tag] = tag
	}
	test.OK(t, testData.DataAPI.UpdateLocalizedText(api.LanguageIDEnglish, text))
	text2, err := testData.DataAPI.LocalizedText(api.LanguageIDEnglish, tags)
	test.OK(t, err)
	test.Equals(t, len(tags), len(text2))
	for tag, text := range text {
		test.Equals(t, text, text2[tag])
	}
}
