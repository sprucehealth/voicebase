package client

import (
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/saml"
)

func viewForPhotoScreen(screen *saml.Screen) visitreview.View {
	sectionView := &visitreview.StandardPhotosSectionView{
		Title:       screen.HeaderSummary,
		Subsections: make([]visitreview.View, len(screen.Questions)),
	}

	keys := make([]string, len(screen.Questions))
	for i, question := range screen.Questions {
		keys[i] = visitreview.PhotosKey(question.Details.Tag)
		sectionView.Subsections[i] = viewForPhotoQuestion(question)
	}

	sectionView.ContentConfig = &visitreview.ContentConfig{
		ViewCondition: visitreview.ViewCondition{
			Op:   visitreview.ConditionAnyKeyExists,
			Keys: keys,
		},
	}

	return sectionView
}

func viewForPhotoQuestion(question *saml.Question) visitreview.View {
	tag := question.Details.Tag
	return &visitreview.StandardPhotosSubsectionView{
		ContentConfig: &visitreview.ContentConfig{
			ViewCondition: visitreview.ViewCondition{
				Op:  visitreview.ConditionKeyExists,
				Key: visitreview.PhotosKey(tag),
			},
		},
		SubsectionView: &visitreview.TitlePhotosItemsListView{
			ContentConfig: &visitreview.ContentConfig{
				Key: visitreview.PhotosKey(tag),
			},
		},
	}
}
