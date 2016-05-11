package client

import (
	"regexp"

	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
)

var (
	tokenMatcher = regexp.MustCompile("<\\w+>")
)

// GenerateIntakeLayout takes a valid SAML and converts it into a VisitIntake layout.
// It is assumed that the SAML is valid and consequently, no validation is done
// at the transformation layer. This helps ensure that the validation is consolidated
// at a single layer.
//
// It is also assumed that the SAML layer generates a unique tag for each question
// and each potential answer, thereby removing the need to ensure or generate
// tags for any questions or potential answers. Tags will be used as unique
// identifiers to questions and potential answers.
func GenerateIntakeLayout(intake *saml.Intake) (*layout.Intake, error) {
	visitIntake := &layout.Intake{
		Transitions: transformTransitions(intake.Sections),
		Sections:    make([]*layout.Section, len(intake.Sections)),
	}

	var err error
	for i, section := range intake.Sections {
		visitIntake.Sections[i], err = transformSection(section)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	return visitIntake, nil
}

func transformTransitions(sections []*saml.Section) []*layout.TransitionItem {
	items := make([]*layout.TransitionItem, len(sections)+1)
	for i, section := range sections {
		buttonText := "Continue"
		if i == 0 {
			buttonText = "Begin"
		}

		items[i] = &layout.TransitionItem{
			Message: section.TransitionToMessage,
			Buttons: []*layout.Button{
				{
					Text:   buttonText,
					Style:  "filled",
					TapURL: "spruce:///action/view_next_visit_section",
				},
			},
		}
	}
	items[len(sections)] = &layout.TransitionItem{
		Message: "That's all the information your doctor will need!",
		Buttons: []*layout.Button{
			{
				Text:   "Continue",
				Style:  "filled",
				TapURL: "spruce:///action/view_next_visit_section",
			},
		},
	}
	return items
}
