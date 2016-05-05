package client

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/saml"
)

func viewsForPhotoSection(section *saml.Section) ([]visitreview.View, error) {
	var views []visitreview.View
	for _, screen := range section.Screens {
		if screen.Type == saml.ScreenTypePhoto {
			views = append(views, viewForPhotoScreen(screen))
		}
	}

	// once all the photo screens in the section have been accounted for, go ahead
	// and add the rest of the questions in this section to the same set of views
	nonPhotoScreens := make([]*saml.Screen, 0, len(section.Screens))
	for _, screen := range section.Screens {
		if screen.Type != saml.ScreenTypePhoto {
			nonPhotoScreens = append(nonPhotoScreens, screen)
		}
	}
	if len(nonPhotoScreens) > 0 {
		subsectionView, err := subsectionViewForScreens("", nonPhotoScreens)
		if err != nil {
			return nil, errors.Trace(err)
		}

		views = append(views, subsectionView)
	}

	return views, nil
}

func viewForQuestionSection(section *saml.Section) (visitreview.View, error) {
	view := &visitreview.StandardSectionView{
		Title: section.Title,
	}

	if len(section.Subsections) > 0 {
		view.Subsections = make([]visitreview.View, 0, len(section.Subsections))
		for _, subsection := range section.Subsections {
			subsectionView, err := subsectionViewForScreens(subsection.Title, subsection.Screens)
			if err != nil {
				return nil, errors.Trace(err)
			}
			view.Subsections = append(view.Subsections, subsectionView)
		}
	} else {
		subsectionView, err := subsectionViewForScreens(section.Title+" Questions", section.Screens)
		if err != nil {
			return nil, errors.Trace(err)
		}
		view.Subsections = append(view.Subsections, subsectionView)
	}

	return view, nil
}

func subsectionViewForScreens(title string, screens []*saml.Screen) (visitreview.View, error) {
	subsectionView := &visitreview.StandardSubsectionView{
		Title: title,
	}

	keys := make([]string, 0, len(screens))
	for _, screen := range screens {
		views, err := viewsForQuestionScreen(screen)
		if err != nil {
			return nil, errors.Trace(err)
		}
		subsectionView.Rows = append(subsectionView.Rows, views...)

		for _, question := range screen.Questions {
			keys = append(keys, visitreview.AnswersKey(question.Details.Tag))
		}
	}

	subsectionView.ContentConfig = &visitreview.ContentConfig{
		ViewCondition: visitreview.ViewCondition{
			Op:   visitreview.ConditionAnyKeyExists,
			Keys: keys,
		},
	}
	return subsectionView, nil
}
