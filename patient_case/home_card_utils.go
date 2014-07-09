package patient_case

import (
	"fmt"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

func getHomeCards(patientCase *common.PatientCase, dataAPI api.DataAPI) ([]common.ClientView, error) {
	var views []common.ClientView

	if patientCase == nil {
		views = []common.ClientView{getStartVisitCard(), getLearnAboutSpruceSection()}
	} else {
		caseNotifications, err := dataAPI.GetNotificationsForCase(patientCase.Id.Int64(), notifyTypes)
		if err != nil {
			return nil, err
		}

		assignments, err := dataAPI.GetActiveMembersOfCareTeamForCase(patientCase.Id.Int64())
		if err != nil {
			return nil, err
		}

		// get current doctor assigned to case
		var currentDoctor *common.Doctor
		for _, assignment := range assignments {
			if assignment.Status == api.STATUS_ACTIVE && assignment.ProviderRole == api.DOCTOR_ROLE {
				currentDoctor = assignment.Doctor
				break
			}
		}

		// populate home cards based on the notification types and the number of notifications in the case
		switch l := len(caseNotifications); {

		case l == 1:
			hView, err := caseNotifications[0].Data.(notification).makeHomeCardView(dataAPI)
			if err != nil {
				return nil, err
			}

			switch caseNotifications[0].NotificationType {

			case CNIncompleteVisit:
				views = []common.ClientView{hView, getSendUsMessageSection(), getLearnAboutSpruceSection()}

			case CNVisitSubmitted:
				views = []common.ClientView{getViewCaseCard(patientCase, currentDoctor, hView), getViewResourceLibrarySection()}

			case CNTreatmentPlan:
				careTeamSection, err := getMeetCareTeamSection(assignments, dataAPI)
				if err != nil {
					return nil, err
				}
				views = []common.ClientView{getViewCaseCard(patientCase, currentDoctor, hView), careTeamSection}

			case CNMessage:
				views = []common.ClientView{getViewCaseCard(patientCase, currentDoctor, hView)}
			}

		case l > 1:
			views = []common.ClientView{getViewCaseCard(patientCase, currentDoctor, &phCaseNotificationMultipleView{
				NotificationCount: int64(l),
				Title:             "New updates to your Dermatology case.",
				ButtonTitle:       "View Case",
				ActionURL:         app_url.ViewCaseAction(patientCase.Id.Int64()),
			}), getSendCareTeamMessageSection(patientCase.Id.Int64())}

		case l == 0:
			views = []common.ClientView{getViewCaseCard(patientCase, currentDoctor, nil), getSendCareTeamMessageSection(patientCase.Id.Int64())}
		}
	}

	for _, v := range views {
		if err := v.Validate(); err != nil {
			return nil, err
		}
	}

	return views, nil
}

func getStartVisitCard() common.ClientView {
	return &phStartVisit{
		Title:       "Start an Acne Visit",
		IconURL:     app_url.IconVisitLarge,
		ActionURL:   app_url.StartVisitAction(),
		ButtonTitle: "Get Started",
		Description: "In less than 24 hours receive an effective, personalized treatment plan from a board-ceritified dermatologist.",
	}
}

func getCompleteVisitCard(patientVisitId int64) common.ClientView {
	return &phContinueVisit{
		Title:       "Continue Your Acne Visit",
		ActionURL:   app_url.ContinueVisitAction(patientVisitId),
		Description: "You're almost there. Complete your visit and get on the path to clear skin.",
		ButtonTitle: "Continue",
	}
}

func getViewCaseCard(patientCase *common.PatientCase, doctor *common.Doctor, notificationView common.ClientView) common.ClientView {
	switch patientCase.Status {

	case common.PCStatusUnclaimed, common.PCStatusTempClaimed:
		return &phCaseView{
			Title:            "Dermatology Case",
			Subtitle:         "Pending Doctor Review",
			ActionURL:        app_url.ViewCaseAction(patientCase.Id.Int64()),
			NotificationView: notificationView,
		}

	case common.PCStatusClaimed:
		return &phCaseView{
			Title:            "Dermatology Case",
			Subtitle:         fmt.Sprintf("With Dr. %s %s", doctor.FirstName, doctor.LastName),
			ActionURL:        app_url.ViewCaseAction(patientCase.Id.Int64()),
			IconURL:          doctor.SmallThumbnailUrl,
			NotificationView: notificationView,
		}
	}

	return nil
}

func getMeetCareTeamSection(careTeamAssignments []*common.CareProviderAssignment, dataAPI api.DataAPI) (common.ClientView, error) {
	sectionView := &phSectionView{
		Title: "Meet your Spruce care team",
		Views: make([]common.ClientView, 0, len(careTeamAssignments)),
	}

	for _, assignment := range careTeamAssignments {
		if assignment.ProviderRole == api.DOCTOR_ROLE {

			doctor, err := dataAPI.GetDoctorFromId(assignment.ProviderId)
			if err != nil {
				return nil, err
			}

			sectionView.Views = append(sectionView.Views, &phSmallIconText{
				Title:       fmt.Sprintf("Dr. %s %s", doctor.FirstName, doctor.LastName),
				Subtitle:    doctor.ShortTitle,
				IconURL:     doctor.SmallThumbnailUrl,
				RoundedIcon: true,
			})
		}
	}

	return sectionView, nil
}

func getViewResourceLibrarySection() common.ClientView {
	return &phSectionView{
		Views: []common.ClientView{
			&phSmallIconText{
				Title:       "Find out what causes acne and more in the resource library",
				IconURL:     app_url.IconResourceLibrary,
				ActionURL:   app_url.ViewResourceLibraryAction(),
				RoundedIcon: true,
			},
		},
	}
}

func getSendCareTeamMessageSection(patientCaseId int64) common.ClientView {
	return &phSectionView{
		Title: "Have a question or a problem?",
		Views: []common.ClientView{
			&phSmallIconText{
				Title:       "Send your care team a message",
				IconURL:     app_url.IconMessagesSmall,
				ActionURL:   app_url.SendCaseMessageAction(patientCaseId),
				RoundedIcon: true,
			},
		},
	}
}

func getSendUsMessageSection() common.ClientView {
	return &phSectionView{
		Title: "Have a question or need help?",
		Views: []common.ClientView{
			&phSmallIconText{
				Title:       "Send us a message",
				IconURL:     app_url.IconMessagesSmall,
				ActionURL:   app_url.EmailSupportAction(),
				RoundedIcon: true,
			},
		},
	}
}

func getLearnAboutSpruceSection() common.ClientView {
	return &phSectionView{
		Title: "Learn more about Spruce",
		Views: []common.ClientView{
			&phSmallIconText{
				Title:       "Meet the Spruce Dermatologists",
				IconURL:     app_url.IconSpruceDoctors,
				ActionURL:   app_url.ViewSampleDoctorProfilesAction(),
				RoundedIcon: true,
			},
			&phSmallIconText{
				Title:       "Learn how a Spruce Visit Works",
				IconURL:     app_url.IconLearnSpruce,
				ActionURL:   app_url.ViewTutorialAction(),
				RoundedIcon: true,
			},
			&phSmallIconText{
				Title:       "See a sample treatment plan",
				IconURL:     app_url.IconBlueTreatmentPlan,
				ActionURL:   app_url.ViewSampleTreatmentPlanAction(),
				RoundedIcon: true,
			},
		},
	}
}
