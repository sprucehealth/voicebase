package client

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/model"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/saml"
	"github.com/sprucehealth/backend/svc/layout"
)

type sectionID struct {
	model.ObjectID
}

func transformSection(section *saml.Section) (*layout.Section, error) {

	numScreens := len(section.Screens)
	for _, subsection := range section.Subsections {
		numScreens += len(subsection.Screens)
	}

	id, err := idgen.NewID()
	if err != nil {
		return nil, errors.Trace(err)
	}

	sectionID := &sectionID{
		model.ObjectID{
			Prefix:  "section_",
			Val:     id,
			IsValid: true,
		},
	}

	visitSection := &layout.Section{
		Title:   section.Title,
		Screens: make([]*layout.Screen, 0, len(section.Screens)),
		ID:      sectionID.String(),
	}

	for _, screen := range section.Screens {
		tScreen, err := transformScreen(screen)
		if err != nil {
			return nil, errors.Trace(err)
		}

		// If the screen type is triage or warning popup screen before a triage screen,
		// then add a condition to ensure that
		// an optional triage user preference is respected.
		if tScreen.Type == layout.ScreenTypeTriage {
			addOptionalTriagePreferenceToScreen(tScreen)

			if len(visitSection.Screens) > 0 && visitSection.Screens[len(visitSection.Screens)-1].Type == layout.ScreenTypeWarningPopup {
				addOptionalTriagePreferenceToScreen(visitSection.Screens[len(visitSection.Screens)-1])
			}
		}

		visitSection.Screens = append(visitSection.Screens, tScreen)
	}

	for _, subsection := range section.Subsections {
		for _, screen := range subsection.Screens {
			tScreen, err := transformScreen(screen)
			if err != nil {
				return nil, errors.Trace(err)
			}
			// If the screen type is triage or warning popup screen before a triage screen,
			// then add a condition to ensure that
			// an optional triage user preference is respected.
			if tScreen.Type == layout.ScreenTypeTriage {
				addOptionalTriagePreferenceToScreen(tScreen)

				if len(visitSection.Screens) > 0 && visitSection.Screens[len(visitSection.Screens)-1].Type == layout.ScreenTypeWarningPopup {
					addOptionalTriagePreferenceToScreen(visitSection.Screens[len(visitSection.Screens)-1])
				}
			}

			visitSection.Screens = append(visitSection.Screens, tScreen)
		}
	}

	return visitSection, nil
}

func addOptionalTriagePreferenceToScreen(screen *layout.Screen) {
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
				screen.Condition,
				preferenceCondition,
			},
		}
	} else {
		condition = preferenceCondition
	}
	screen.Condition = condition
}
