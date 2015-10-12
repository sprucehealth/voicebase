package feedback

import (
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/errors"
)

type feedbackPrompt struct {
	RatingConfig map[string][]string    `json:"prompt_priority_by_rating"`
	PromptByID   map[string]interface{} `json:"prompt_by_id"`
}

type feedbackHomeCard struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	Title          string         `json:"title"`
	Dismissable    bool           `json:"dismissable"`
	FeedbackPrompt feedbackPrompt `json:"feedback_prompt"`
}

func (f *feedbackHomeCard) Validate() error {
	return nil
}

// HomeCardForCase returns a home card to get feedback from the patient
// based on the current rating level configuration.
func HomeCardForCase(feedbackClient DAL, caseID int64, platform common.Platform) (common.ClientView, error) {

	feedbackFor := ForCase(caseID)

	// don't return feedback home card if patient has already dismissed card
	// or if there is no pending feedback or if no entry exists altogether
	pf, err := feedbackClient.PatientFeedback(feedbackFor)
	if errors.Cause(err) == ErrNoPatientFeedback {
		return nil, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	} else if pf == nil {
		return nil, nil
	} else if !pf.Pending || pf.Dismissed {
		return nil, nil
	}

	// get the config for each rating level
	ratingConfigs, err := feedbackClient.RatingConfigs()
	if err != nil {
		return nil, err
	}

	// collect all the unique template tags
	// as it is possible that they are duplicated across ratings
	uniqueTemplateTags := make(map[string]bool)
	for _, templateTagsCSV := range ratingConfigs {
		if strings.TrimSpace(templateTagsCSV) == "" {
			continue
		}

		templateTags := strings.Split(templateTagsCSV, ",")
		for _, tt := range templateTags {
			uniqueTemplateTags[tt] = true
		}
	}
	// concurrently get the active template for each tag
	p := conc.NewParallel()
	templates := make(chan *FeedbackTemplateData, len(uniqueTemplateTags))
	for tag := range uniqueTemplateTags {
		tagCopy := tag
		p.Go(func() error {
			activeTemplate, err := feedbackClient.ActiveFeedbackTemplate(tagCopy)
			if err != nil {
				return err
			}
			templates <- activeTemplate
			return nil
		})
	}

	if err := p.Wait(); err != nil {
		return nil, err
	}

	// create a map of template tag -> template
	tagToTemplateMap := make(map[string]*FeedbackTemplateData)
	for i := 0; i < len(uniqueTemplateTags); i++ {
		activeTemplate := <-templates
		tagToTemplateMap[activeTemplate.Tag] = activeTemplate
	}

	hc := &feedbackHomeCard{
		ID:          feedbackFor,
		Type:        "patient_home:feedback",
		Title:       "How would you rate your overall Spruce experience so far?",
		Dismissable: true,
		FeedbackPrompt: feedbackPrompt{
			RatingConfig: make(map[string][]string),
			PromptByID:   make(map[string]interface{}),
		},
	}

	// populate prompt by id (with the active template IDs rather than the tags)
	for r, templateTagsCSV := range ratingConfigs {
		ratingString := strconv.Itoa(r)

		if strings.TrimSpace(templateTagsCSV) == "" {
			continue
		}

		templateTags := strings.Split(templateTagsCSV, ",")
		for _, tt := range templateTags {
			activeTemplate := tagToTemplateMap[tt]
			activeTemplateID := strconv.FormatInt(activeTemplate.ID, 10)
			hc.FeedbackPrompt.RatingConfig[ratingString] = append(hc.FeedbackPrompt.RatingConfig[ratingString], activeTemplateID)
		}
	}

	// for every unique tag populate the client view representation of the template
	for _, activeTemplate := range tagToTemplateMap {
		activeTemplateID := strconv.FormatInt(activeTemplate.ID, 10)
		hc.FeedbackPrompt.PromptByID[activeTemplateID] = activeTemplate.Template.ClientView(activeTemplate.ID, platform)
	}

	return hc, nil
}
