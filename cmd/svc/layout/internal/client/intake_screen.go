package client

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
)

type screenID struct {
	model.ObjectID
}

func transformScreen(screen *saml.Screen) (*layout.Screen, error) {

	id, err := idgen.NewID()
	if err != nil {
		return nil, errors.Trace(err)
	}

	screenID := &sectionID{
		model.ObjectID{
			Prefix:  "screen_",
			Val:     id,
			IsValid: true,
		},
	}

	visitScreen := &layout.Screen{
		ID:                   screenID.String(),
		HeaderTitle:          screen.HeaderTitle,
		HeaderTitleHasTokens: tokenMatcher.Match([]byte(screen.HeaderTitle)),
		HeaderSubtitle:       screen.HeaderSubtitle,
		HeaderSummary:        screen.HeaderSummary,
		Questions:            make([]*layout.Question, len(screen.Questions)),
		Type:                 screen.Type,
		Condition:            transformCondition(screen.Condition),
		Body:                 transformScreenBody(screen.Body),
		BottomButtonTitle:    screen.BottomButtonTitle,
		ContentTitle:         screen.ContentHeaderTitle,
		Title:                screen.Title,
		ClientData:           transformClientData(screen.ClientData),
	}

	// If the screen type is triage, then add a condition to ensure that
	// an optional triage user preference is respected.
	//
	// TODO: Move this to the SAML layer. Setting here for now given that the
	// condition is specific to baymax and avoid the complexity of having to manage
	// how to only set the condition for baymax.
	if visitScreen.Type == layout.ScreenTypeTriage {
		var condition *layout.Condition
		preferenceCondition := &layout.Condition{
			Operation:  "boolean_equals",
			BoolValue:  ptr.Bool(false),
			DataSource: "preference.optional_triage",
		}

		if screen.Condition != nil {
			condition = &layout.Condition{
				Operation: "and",
				Operands: []*layout.Condition{
					visitScreen.Condition,
					preferenceCondition,
				},
			}
		} else {
			condition = preferenceCondition
		}
		visitScreen.Condition = condition
	}

	// map all photo screens to media screens
	if visitScreen.Type == saml.ScreenTypePhoto {
		visitScreen.Type = layout.ScreenTypeMedia
	}

	for i, question := range screen.Questions {
		visitScreen.Questions[i], err = transformQuestion(question)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	return visitScreen, nil
}

func transformClientData(clientData *saml.ScreenClientData) *layout.ScreenClientData {
	if clientData == nil {
		return nil
	}
	return &layout.ScreenClientData{
		RequiresAtLeastOneQuestionAnswered: clientData.RequiresAtLeastOneQuestionAnswered,
		Triage: transformTriageParams(clientData.Triage),
		Views:  transformViews(clientData.Views),
	}
}

func transformTriageParams(params *saml.TriageParams) *layout.TriageParams {
	if params == nil {
		return nil
	}

	return &layout.TriageParams{
		Title:         params.Title,
		ActionMessage: params.ActionMessage,
		ActionURL:     params.ActionURL,
		Abandon:       params.Abandon,
	}
}

func transformViews(views []saml.View) []map[string]interface{} {
	tViews := make([]map[string]interface{}, len(views))
	for i, v := range views {
		tViews[i] = map[string]interface{}(v)
	}
	return tViews
}

func transformScreenBody(body *saml.ScreenBody) *layout.Body {
	if body == nil {
		return nil
	}

	return &layout.Body{
		Text: body.Text,
	}
}
