package client

import (
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/visitreview"
	"github.com/sprucehealth/backend/saml"
)

func GenerateReviewLayout(intake *saml.Intake) (*visitreview.SectionListView, error) {
	sectionList := &visitreview.SectionListView{
		Sections: []visitreview.View{alertSection()},
	}

	// add media section first
	mediaSectionIDx := -1
	for i, section := range intake.Sections {
		for _, screen := range section.Screens {
			if screen.Type == saml.ScreenTypePhoto || screen.Type == saml.ScreenTypeMedia {
				views, err := viewsForMediaSection(section)
				if err != nil {
					return nil, errors.Trace(err)
				}
				sectionList.Sections = append(sectionList.Sections, views...)
				mediaSectionIDx = i
				break
			}
		}
	}

	// now add rest of sections
	for i, section := range intake.Sections {
		if mediaSectionIDx == i {
			continue
		}

		questionSection, err := viewForQuestionSection(section)
		if err != nil {
			return nil, errors.Trace(err)
		}

		sectionList.Sections = append(sectionList.Sections, questionSection)
	}

	return sectionList, errors.Trace(sectionList.Validate())
}

func alertSection() *visitreview.StandardSectionView {

	section := &visitreview.StandardSectionView{
		Title: "Alerts",
		Subsections: []visitreview.View{
			&visitreview.StandardSubsectionView{
				Title: "Alerts",
				Rows: []visitreview.View{
					&visitreview.StandardOneColumnRowView{
						ContentConfig: &visitreview.ContentConfig{
							ViewCondition: visitreview.ViewCondition{
								Op:  visitreview.ConditionKeyExists,
								Key: "visit_alerts",
							},
						},
						SingleView: &visitreview.AlertLabelsList{
							ContentConfig: &visitreview.ContentConfig{
								Key: "visit_alerts",
							},
							EmptyStateView: &visitreview.EmptyLabelView{
								ContentConfig: &visitreview.ContentConfig{
									Key: "visit_alerts:empty_state_text",
								},
							},
						},
					},
				},
			},
		},
	}

	return section
}
