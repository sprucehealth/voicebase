package test_api

import (
	"testing"

	"github.com/sprucehealth/backend/feedback"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/test_integration"
)

func TestTemplateManagement(t *testing.T) {
	testData := test_integration.SetupTest(t)
	defer testData.Close(t)

	feedbackClient := feedback.NewDAL(testData.DB)

	// create new free text template
	ft := &feedback.FreeTextTemplate{
		Title:           "title",
		PlaceholderText: "placeholder",
		ButtonTitle:     "button title",
	}

	if err := ft.Validate(); err != nil {
		t.Fatalf(err.Error())
	}

	id, err := feedbackClient.CreateFeedbackTemplate(feedback.FeedbackTemplateData{
		Type:     feedback.FTFreetext,
		Tag:      "testing",
		Template: ft,
	})
	if err != nil {
		t.Fatalf(err.Error())
	}

	// attempt to retrieve the template just created
	td, err := feedbackClient.FeedbackTemplate(id)
	if err != nil {
		t.Fatal(err)
	}

	test.Equals(t, true, td.Active)
	test.Equals(t, id, td.ID)
	test.Equals(t, "testing", td.Tag)
	f, ok := td.Template.(*feedback.FreeTextTemplate)
	test.Equals(t, true, ok)
	test.Equals(t, "placeholder", f.PlaceholderText)
	test.Equals(t, "title", f.Title)
	test.Equals(t, "button title", f.ButtonTitle)

	// attempt to retrieve the template just created by tag
	_, err = feedbackClient.ActiveFeedbackTemplate("testing")
	test.OK(t, err)

	// attempt to list active templates
	templates, err := feedbackClient.ListActiveTemplates()
	if err != nil {
		t.Fatalf(err.Error())
	}
	test.Equals(t, 1, len(templates))

	// attempt to revise an existing template
	ft2 := &feedback.FreeTextTemplate{
		Title:           "title2",
		PlaceholderText: "placeholder2",
		ButtonTitle:     "button title2",
	}

	if err := ft2.Validate(); err != nil {
		t.Fatalf(err.Error())
	}

	id2, err := feedbackClient.CreateFeedbackTemplate(feedback.FeedbackTemplateData{
		Type:     feedback.FTFreetext,
		Tag:      "testing",
		Template: ft2,
	})
	if err != nil {
		t.Fatalf(err.Error())
	}

	// ensure that ids are different
	test.Equals(t, true, id != id2)

	// ensure that old template is not active
	td, err = feedbackClient.FeedbackTemplate(id)
	if err != nil {
		t.Fatalf(err.Error())
	}
	test.Equals(t, false, td.Active)

	// ensure that new template is active
	td2, err := feedbackClient.FeedbackTemplate(id2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	test.Equals(t, true, td2.Active)

	// ensure there is just a single active template
	templates, err = feedbackClient.ListActiveTemplates()
	if err != nil {
		t.Fatalf(err.Error())
	}
	test.Equals(t, 1, len(templates))

	// lets create a new template with a different tag
	ft3 := &feedback.FreeTextTemplate{
		Title:           "title2",
		PlaceholderText: "placeholder2",
		ButtonTitle:     "button title2",
	}

	if err := ft3.Validate(); err != nil {
		t.Fatalf(err.Error())
	}

	_, err = feedbackClient.CreateFeedbackTemplate(feedback.FeedbackTemplateData{
		Type:     feedback.FTFreetext,
		Tag:      "testing3",
		Template: ft3,
	})
	if err != nil {
		t.Fatalf(err.Error())
	}

	// now there should be two active templates
	templates, err = feedbackClient.ListActiveTemplates()
	if err != nil {
		t.Fatalf(err.Error())
	}
	test.Equals(t, 2, len(templates))

	// lets attempt to retrieve rating configs (there should be non)
	ratingConfigs, err := feedbackClient.RatingConfigs()
	if err != nil {
		t.Fatalf(err.Error())
	}
	test.Equals(t, 0, len(ratingConfigs))

	// lets attempt to create a rating config for rating 1
	test.OK(t, feedbackClient.UpsertRatingConfigs(map[int]string{
		1: "testing3",
	}))

	// lets attempt to retreive rating config for rating 1
	ratingConfigs, err = feedbackClient.RatingConfigs()
	if err != nil {
		t.Fatalf(err.Error())
	}
	test.Equals(t, 1, len(ratingConfigs))
	test.Equals(t, "testing3", ratingConfigs[1])

	// lets attempt to create rating for 1 with non-existent tag
	// should not be able to create
	test.Equals(t, true, feedbackClient.UpsertRatingConfigs(map[int]string{
		1: "non-existent-tag",
	}) != nil)

	// lets attempt to create rating for 1 with multiple tags
	test.OK(t, feedbackClient.UpsertRatingConfigs(map[int]string{
		1: "testing3,testing",
	}))

	// there should still be single config
	ratingConfigs, err = feedbackClient.RatingConfigs()
	if err != nil {
		t.Fatalf(err.Error())
	}
	test.Equals(t, 1, len(ratingConfigs))

	// lets attempt to create config for rating 2
	test.OK(t, feedbackClient.UpsertRatingConfigs(map[int]string{
		2: "testing3",
	}))

	// there should now be two configs
	ratingConfigs, err = feedbackClient.RatingConfigs()
	if err != nil {
		t.Fatalf(err.Error())
	}
	test.Equals(t, 2, len(ratingConfigs))

}
