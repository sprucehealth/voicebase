package client

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/idgen"
	"github.com/sprucehealth/backend/libs/model"
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
		visitSection.Screens = append(visitSection.Screens, tScreen)
	}

	for _, subsection := range section.Subsections {
		for _, screen := range subsection.Screens {
			tScreen, err := transformScreen(screen)
			if err != nil {
				return nil, errors.Trace(err)
			}
			visitSection.Screens = append(visitSection.Screens, tScreen)
		}
	}

	return visitSection, nil
}
