package feedback

import "strings"

// StructuredResponse represents a response
// from the patient to a structured feedback prompt.
type StructuredResponse interface {

	// ResponseType represents the type of template
	// the response corresponds to.
	ResponseType() string

	// TemplateID represents a particular version of a feedback template
	// that the response corresponds to.
	TemplateID() int64

	// IsZero indicates if the response is empty
	IsZero() bool
}

// FreeTextResponse represents a response to a free text feedback request.
type FreeTextResponse struct {
	// FeedbackTemplateID represents the ID of the feedback template
	FeedbackTemplateID int64 `json:"id,string"`

	// Response represents the actual free text response.
	Response string `json:"text"`
}

func (f *FreeTextResponse) ResponseType() string {
	return FTFreetext
}

func (f *FreeTextResponse) TemplateID() int64 {
	return f.FeedbackTemplateID
}

func (f *FreeTextResponse) IsZero() bool {
	return strings.TrimSpace(f.Response) == ""
}

// MultipleChoiceResponse represents a response to a multiple choice feedback request.
type MultipleChoiceResponse struct {
	// FeedbackTemplateID represents the ID of the feedback template
	FeedbackTemplateID int64 `json:"id,string"`

	// AnswerSelections represents the multiple choice selections by the patient.
	AnswerSelections []MultipleChoiceSelection `json:"answers"`
}

func (f *MultipleChoiceResponse) ResponseType() string {
	return FTMultipleChoice
}

func (f *MultipleChoiceResponse) TemplateID() int64 {
	return f.FeedbackTemplateID
}

func (f *MultipleChoiceResponse) IsZero() bool {
	return len(f.AnswerSelections) == 0
}

// MultipleChoiceSelection represents an answer selection for a multiple choice feedback request.
type MultipleChoiceSelection struct {
	// PotentialAnswerID represents a multiple choice potential answer.
	PotentialAnswerID string `json:"id"`
	// Text represents any user inputted text for the multiple choice response.
	Text string `json:"user_text"`
}

// OpenURLResponse represents a response to an OpenURL feedback request.
type OpenURLResponse struct {
	// FeedbackTemplateID represents the ID of the feedback template
	FeedbackTemplateID int64 `json:"id,string"`

	// OpenedURL indicates whether or not user opened the url.
	OpenedURL bool `json:"opened_url"`
}

func (f *OpenURLResponse) ResponseType() string {
	return FTOpenURL
}

func (f *OpenURLResponse) TemplateID() int64 {
	return f.FeedbackTemplateID
}

func (f *OpenURLResponse) IsZero() bool {
	return !f.OpenedURL
}
