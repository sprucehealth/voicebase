package client

import (
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/saml"
)

func viewForMediaScreen(screen *saml.Screen) visitreview.View {
	sectionView := &visitreview.StandardMediaSectionView{
		Title:       screen.HeaderSummary,
		Subsections: make([]visitreview.View, len(screen.Questions)),
	}

	keys := make([]string, len(screen.Questions))
	for i, question := range screen.Questions {
		keys[i] = visitreview.MediaKey(question.Details.Tag)
		sectionView.Subsections[i] = viewForMediaQuestion(question)
	}

	sectionView.ContentConfig = &visitreview.ContentConfig{
		ViewCondition: visitreview.ViewCondition{
			Op:   visitreview.ConditionAnyKeyExists,
			Keys: keys,
		},
	}

	return sectionView
}

func viewForMediaQuestion(question *saml.Question) visitreview.View {
	tag := question.Details.Tag
	return &visitreview.StandardMediaSubsectionView{
		ContentConfig: &visitreview.ContentConfig{
			ViewCondition: visitreview.ViewCondition{
				Op:  visitreview.ConditionKeyExists,
				Key: visitreview.MediaKey(tag),
			},
		},
		SubsectionView: &visitreview.TitleMediaItemsListView{
			ContentConfig: &visitreview.ContentConfig{
				Key: visitreview.MediaKey(tag),
			},
		},
	}
}
