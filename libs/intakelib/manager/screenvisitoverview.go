package manager

import (
	"github.com/gogo/protobuf/proto"
	"github.com/sprucehealth/backend/cmd/svc/restapi/app_url"
	"github.com/sprucehealth/backend/libs/intakelib/protobuf/intake"
)

var statusTypeToSectionFilledStateMapping = map[completionStatusType]*intake.VisitOverviewScreen_Section_FilledState{
	statusTypeUncomputed: intake.VisitOverviewScreen_Section_FILLED_STATE_UNDEFINED.Enum(),
	statusTypeIncomplete: intake.VisitOverviewScreen_Section_UNFILLED.Enum(),
	statusTypeComplete:   intake.VisitOverviewScreen_Section_FILLED.Enum(),
}

// createMarshalledVisitOverviewScreen computes the visit overview screen based on the
// current visitCompletionStatus.
func createMarshalledVisitOverviewScreen(vs *visitCompletionStatus) ([]byte, error) {
	tItem := vs.visitManager.visit.Transitions[vs.resumeSectionIndex]
	var tapLink string

	if vs.resumeScreenID != "" {
		tapLink = app_url.ViewVisitScreen(vs.resumeScreenID).String()
	} else {
		// if there is no screen to resume the visit on, instruct the
		// client to move past the visit.
		tapLink = app_url.CompleteVisit().String()
	}

	overviewScreen := &intake.VisitOverviewScreen{
		Header: &intake.VisitOverviewScreen_Header{
			Title:     proto.String(vs.visitManager.visit.OverviewHeader.Title),
			Subtitle:  proto.String(vs.visitManager.visit.OverviewHeader.Subtitle),
			ImageLink: proto.String(vs.visitManager.visit.OverviewHeader.IconURL),
		},
		Id:   proto.String(screenTypeVisitOverview.String()),
		Text: proto.String(tItem.Message),
		BottomButton: &intake.Button{
			Text:    proto.String(tItem.Buttons[0].Text),
			TapLink: proto.String(tapLink),
		},
		Sections: make([]*intake.VisitOverviewScreen_Section, len(vs.statuses)),
	}

	for i, sectionStatus := range vs.statuses {
		overviewScreen.Sections[i] = &intake.VisitOverviewScreen_Section{
			TapLink:            proto.String(app_url.ViewVisitScreen(sectionStatus.resumeScreenID).String()),
			Name:               proto.String(vs.visitManager.visit.Sections[i].Title),
			PrevFilledState:    statusTypeToSectionFilledStateMapping[sectionStatus.lastShownStatus],
			CurrentFilledState: statusTypeToSectionFilledStateMapping[sectionStatus.currentStatus],
		}

		if i <= vs.resumeSectionIndex {
			overviewScreen.Sections[i].CurrentEnabledState = intake.VisitOverviewScreen_Section_ENABLED.Enum()
		} else {
			overviewScreen.Sections[i].CurrentEnabledState = intake.VisitOverviewScreen_Section_DISABLED.Enum()
		}
	}

	// now that the user has been shown the visit overview screen, update the
	// previous states so that the filled state doesn't animate again when the visit overview
	// screen is shown again.
	vs.updateLastShownStatuses()

	data, err := proto.Marshal(overviewScreen)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(&intake.ScreenData{
		Type: intake.ScreenData_VISIT_OVERVIEW.Enum(),
		Data: data,
	})
}
