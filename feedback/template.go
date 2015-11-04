package feedback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/errors"
)

// FeedbackTemplate is an interface conformed to by
// any object considered a feedback template
// to be used for structured in-app feedback input from
// the patient.
type FeedbackTemplate interface {
	// TemplateType represents the type of the feedback template.
	TemplateType() string

	// Validate validates the contents of the template to ensure that it is complete.
	Validate() error

	// ClientView is the client view representation of the feedback template.
	ClientView(id int64, platform common.Platform) interface{}

	// ParseAndValidateResponse unmarshales the provided jsonData into an instance of appropriate type
	// and then ensures that the response represents a complete response based on the template type.
	ParseAndValidateResponse(templateID int64, jsonData []byte) (StructuredResponse, error)

	// ResponseString parses the provided response in json form into the appropriate type and
	// then outputs a string that represents a complete template and response set for the client.
	ResponseString(templateID int64, resJSON []byte) (string, error)
}

// FeedbackTemplateData represents a template
// persisted in the database along with associated metadata
type FeedbackTemplateData struct {
	// ID represents the unique ID of the feedback template.
	ID int64

	// Active represents whether or not the feedback template is the current
	// active version.
	Active bool

	// Created represents the time at which the feedback was created.
	Created time.Time

	// Tag represents a human readable identifier that groups all versions
	// of a template together.
	Tag string

	// Type represents the type of the template.
	Type string

	// Template represents the contents of the template.
	Template FeedbackTemplate
}

// TemplateFromJSON unmarshals the json data into its appropriate type.
func TemplateFromJSON(templateType string, jsonData []byte) (FeedbackTemplate, error) {
	fDataType, ok := supportedTemplateTypes[templateType]
	if !ok {
		return nil, errors.Trace(fmt.Errorf("Unable to find template type for %s", templateType))
	}

	template := reflect.New(fDataType).Interface().(FeedbackTemplate)
	if err := json.Unmarshal(jsonData, &template); err != nil {
		return nil, errors.Trace(err)
	}

	return template, nil
}

const (
	// FTFreeText is a free text feedback template.
	FTFreetext string = "feedback:freetext"

	// FTMultipleChoice is a multiple choice feedback template.
	FTMultipleChoice string = "feedback:multiple_choice"

	// FTOpenURL is an open url feedback template.
	FTOpenURL string = "feedback:open_url"
)

// FreeTextTemplate is used to solicit free text
// entry once feedback is provided.
type FreeTextTemplate struct {
	// Title is the question for the free text.
	Title string `json:"title"`

	// Placeholder is displayed in the text box as hint text to the user.
	PlaceholderText string `json:"placeholder"`

	// ButtonTitle represents the text to display in the button
	ButtonTitle string `json:"button_title"`
}

func (f *FreeTextTemplate) TemplateType() string {
	return FTFreetext
}

func (f *FreeTextTemplate) Validate() error {
	if f.Title == "" {
		return fmt.Errorf("title required for template type %s", FTFreetext)
	} else if f.PlaceholderText == "" {
		return fmt.Errorf("placeholder text required for remplate type %s", FTFreetext)
	} else if f.ButtonTitle == "" {
		return fmt.Errorf("button title required for template type %s", FTFreetext)
	}
	return nil
}

type freeTextClientView struct {
	ID          int64  `json:"id,string"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Placeholder string `json:"placeholder"`
	ButtonTitle string `json:"button_title"`
}

func (f *FreeTextTemplate) ClientView(id int64, platform common.Platform) interface{} {
	return &freeTextClientView{
		ID:          id,
		Type:        FTFreetext,
		Title:       f.Title,
		Placeholder: f.PlaceholderText,
		ButtonTitle: f.ButtonTitle,
	}
}

func (f *FreeTextTemplate) ParseAndValidateResponse(templateID int64, jsonData []byte) (StructuredResponse, error) {
	var res FreeTextResponse
	if err := json.Unmarshal(jsonData, &res); err != nil {
		return nil, errors.Trace(err)
	}
	res.FeedbackTemplateID = templateID

	return &res, nil
}

func (f *FreeTextTemplate) ResponseString(templateID int64, resJSON []byte) (string, error) {
	ft, err := f.ParseAndValidateResponse(templateID, resJSON)
	if err != nil {
		return "", errors.Trace(err)
	}

	fr := ft.(*FreeTextResponse)

	return fmt.Sprintf("%s\n%s", f.Title, fr.Response), nil
}

// OpenURLTemplate represents the open url feedback template
// used to direct the client to click a particular link once feedback is provided.
type OpenURLTemplate struct {
	// Title represents the text to display as title in the feedback request.
	Title string `json:"title"`
	// ButtonTitle represents the text in the button of the feedback request.
	ButtonTitle string `json:"button_title"`
	// AndroidURL represents the open and icon url configuration for an android client.
	AndroidConfig OpenURLTemplatePlatformConfig `json:"android"`
	// IOSURL represents the open and icon url configuration for an ios client.
	IOSConfig OpenURLTemplatePlatformConfig `json:"ios"`
}

func (o *OpenURLTemplate) TemplateType() string {
	return FTOpenURL
}

func (o *OpenURLTemplate) Validate() error {
	if o.Title == "" {
		return fmt.Errorf("title required for template type %s", FTOpenURL)
	} else if o.ButtonTitle == "" {
		return fmt.Errorf("button_title required for template type %s", FTOpenURL)
	} else if err := o.AndroidConfig.Validate(); err != nil {
		return fmt.Errorf("android url validation error: '%s' for type %s ", err.Error(), FTOpenURL)
	} else if err := o.IOSConfig.Validate(); err != nil {
		return fmt.Errorf("ios url definition error: '%s' for type %s", err.Error(), FTOpenURL)
	}

	return nil
}

// URL is a structured used for configuring icon and open url.
type OpenURLTemplatePlatformConfig struct {
	IconURL  string `json:"icon_url"`
	OpenURL  string `json:"open_url"`
	BodyText string `json:"body_text"`
}

func (u *OpenURLTemplatePlatformConfig) Validate() error {
	if u.IconURL == "" {
		return fmt.Errorf("icon_url definition required")
	} else if u.OpenURL == "" {
		return fmt.Errorf("open_url definition required")
	} else if u.BodyText == "" {
		return fmt.Errorf("body_text definition required")
	}

	return nil
}

type bodyClientView struct {
	IconURL string `json:"icon_url"`
	Text    string `json:"text"`
}

type openURLClientView struct {
	ID          int64           `json:"id,string"`
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	Body        *bodyClientView `json:"body"`
	ButtonTitle string          `json:"button_title"`
	URL         string          `json:"url"`
}

func (o *OpenURLTemplate) ClientView(id int64, platform common.Platform) interface{} {

	var iconURL, openURL, bodyText string
	switch platform {
	case common.Android:
		iconURL = o.AndroidConfig.IconURL
		openURL = o.AndroidConfig.OpenURL
		bodyText = o.AndroidConfig.BodyText
	case common.IOS:
		iconURL = o.IOSConfig.IconURL
		openURL = o.IOSConfig.OpenURL
		bodyText = o.IOSConfig.BodyText
	}

	return &openURLClientView{
		ID:          id,
		Type:        FTOpenURL,
		Title:       o.Title,
		ButtonTitle: o.ButtonTitle,
		URL:         openURL,
		Body: &bodyClientView{
			IconURL: iconURL,
			Text:    bodyText,
		},
	}
}

func (o *OpenURLTemplate) ParseAndValidateResponse(templateID int64, jsonData []byte) (StructuredResponse, error) {
	var res OpenURLResponse
	if err := json.Unmarshal(jsonData, &res); err != nil {
		return nil, err
	}
	res.FeedbackTemplateID = templateID

	return &res, nil
}

func (o *OpenURLTemplate) ResponseString(templateID int64, resJSON []byte) (string, error) {
	return "", nil
}

// MultipleChoiceTemplate is a template used to configure a multiple choice structured
// feedback request in response to a patient providing a particular rating.
type MultipleChoiceTemplate struct {

	// Title represents the title text to display for the feedback request.
	Title string `json:"title"`

	// Subtitle represents the subtitle text to display below the title for the feedback request.
	Subtitle string `json:"subtitle"`

	// ButtonTitle represents the text in the button of the feedback request.
	ButtonTitle string `json:"button_title"`

	// PotentialAnswers represents the list of possible answers from whcih
	// the patient can select an option.
	PotentialAnswers []*PotentialAnswer `json:"potential_answers"`
}

func (m *MultipleChoiceTemplate) TemplateType() string {
	return FTMultipleChoice
}

func (m *MultipleChoiceTemplate) Validate() error {
	if m.Title == "" {
		return fmt.Errorf("title is required for template type %s", FTMultipleChoice)
	} else if m.Subtitle == "" {
		return fmt.Errorf("subtitle is required for template type %s", FTMultipleChoice)
	} else if m.ButtonTitle == "" {
		return fmt.Errorf("button_title is required for template type %s", FTMultipleChoice)
	} else if len(m.PotentialAnswers) == 0 {
		return fmt.Errorf("at least 1 potential answer is required for template type %s", FTMultipleChoice)
	}

	potentialAnswerIDsSeen := make(map[string]bool)
	for i, pa := range m.PotentialAnswers {

		// assign an ID to every potential answer if one isn't already assigned
		pa.ID = strconv.Itoa(i)

		if err := pa.Validate(); err != nil {
			return fmt.Errorf("potential answer invalid for template type %s", pa.Type)
		}

		// ensure all IDs for potential answers are unique
		if potentialAnswerIDsSeen[pa.ID] {
			return fmt.Errorf("Duplicate potential answer id %s", pa.ID)
		}
		potentialAnswerIDsSeen[pa.ID] = true
	}

	return nil
}

// PotentialAnswer represents a single answer for a patient.
type PotentialAnswer struct {

	// ID represents the identifier for the potential answer.
	ID string `json:"id,omitempty"`
	// Text represents the text to show for this choice.
	Text string `json:"text"`
	// Type represents the  answer type for this choice. Can be left blank and assumed
	// to be a regular multipel choice type.
	Type string `json:"answer_type"`
	// PlaceholderText represents the text to display as hint text for a free text
	// other type. Can be left blank.
	PlaceholderText string `json:"placeholder_text"`
	// FreeTextRequired indicates whether or not a free text response is required for other
	// free text type.
	FreeTextRequired bool `json:"free_text_required"`
}

func (p *PotentialAnswer) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("id is required")
	} else if p.Text == "" {
		return fmt.Errorf("text is required")
	}

	return nil
}

type multipleChoiceClientView struct {
	ID               int64                       `json:"id,string"`
	Type             string                      `json:"type"`
	Title            string                      `json:"title"`
	Subtitle         string                      `json:"subtitle"`
	ButtonTitle      string                      `json:"button_title"`
	PotentialAnswers []potentialAnswerClientView `json:"potential_answers"`
}

type potentialAnswerClientView struct {
	ID               string `json:"id"`
	Text             string `json:"text"`
	Type             string `json:"type,omitempty"`
	PlaceholderText  string `json:"placeholder_text,omitempty"`
	FreeTextRequired bool   `json:"free_text_required"`
}

func (m *MultipleChoiceTemplate) ClientView(id int64, platform common.Platform) interface{} {
	pa := make([]potentialAnswerClientView, len(m.PotentialAnswers))
	for i, pItem := range m.PotentialAnswers {
		pa[i] = potentialAnswerClientView{
			ID:               pItem.ID,
			Text:             pItem.Text,
			Type:             pItem.Type,
			FreeTextRequired: pItem.FreeTextRequired,
			PlaceholderText:  pItem.PlaceholderText,
		}
	}

	return &multipleChoiceClientView{
		ID:               id,
		Type:             FTMultipleChoice,
		Title:            m.Title,
		Subtitle:         m.Subtitle,
		ButtonTitle:      m.ButtonTitle,
		PotentialAnswers: pa,
	}
}

func (m *MultipleChoiceTemplate) ParseAndValidateResponse(templateID int64, jsonData []byte) (StructuredResponse, error) {
	var res MultipleChoiceResponse
	if err := json.Unmarshal(jsonData, &res); err != nil {
		return nil, errors.Trace(err)
	}
	res.FeedbackTemplateID = templateID

	// ensure that every response patient entered is valid
	idToPotentialAnswerMap := make(map[string]*PotentialAnswer)
	for _, pa := range m.PotentialAnswers {
		idToPotentialAnswerMap[pa.ID] = pa
	}

	// lets go through patient response
	for _, a := range res.AnswerSelections {
		pa := idToPotentialAnswerMap[a.PotentialAnswerID]
		if pa == nil {
			return nil, errors.Trace(fmt.Errorf("%s is not a valid answer selection", a.PotentialAnswerID))
		}

		if pa.FreeTextRequired && a.Text == "" {
			return nil, errors.Trace(fmt.Errorf("free text is required for id %s but not specified", a.PotentialAnswerID))
		}
	}

	return &res, nil
}

func (m *MultipleChoiceTemplate) ResponseString(templateID int64, resJSON []byte) (string, error) {
	res, err := m.ParseAndValidateResponse(templateID, resJSON)
	if err != nil {
		return "", errors.Trace(err)
	}

	mr, ok := res.(*MultipleChoiceResponse)
	if !ok {
		return "", nil
	}

	var b bytes.Buffer
	b.WriteString(m.Title)
	b.WriteString("\n\n")

	idToPotentialAnswerMap := make(map[string]*PotentialAnswer)

	for _, pa := range m.PotentialAnswers {
		idToPotentialAnswerMap[pa.ID] = pa
	}

	for _, aItem := range mr.AnswerSelections {
		if aItem.Text != "" {
			b.WriteString(fmt.Sprintf("%s:%s\n", idToPotentialAnswerMap[aItem.PotentialAnswerID].Text, aItem.Text))
		} else {
			b.WriteString(idToPotentialAnswerMap[aItem.PotentialAnswerID].Text)
			b.WriteString("\n")
		}
	}
	return b.String(), nil
}

func init() {
	registerTemplateType(&FreeTextTemplate{})
	registerTemplateType(&MultipleChoiceTemplate{})
	registerTemplateType(&OpenURLTemplate{})
}

var supportedTemplateTypes = make(map[string]reflect.Type)

func registerTemplateType(f FeedbackTemplate) {
	supportedTemplateTypes[f.TemplateType()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(f)).Interface())
}
