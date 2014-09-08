package patient_case

import (
	"fmt"

	"github.com/sprucehealth/backend/address"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

func getHomeCards(patientCase *common.PatientCase, cityStateInfo *address.CityState, dataAPI api.DataAPI, apiDomain string) ([]common.ClientView, error) {
	var views []common.ClientView

	if patientCase == nil {
		isAvailable, err := dataAPI.IsEligibleToServePatientsInState(cityStateInfo.StateAbbreviation, apiservice.HEALTH_CONDITION_ACNE_ID)
		if err != nil {
			return nil, err
		}

		// only show the get start visit card if the patient is coming from a state in which we serve patients
		if isAvailable {
			views = []common.ClientView{getStartVisitCard(), getLearnAboutSpruceSection()}
		} else {
			views = []common.ClientView{getLearnAboutSpruceSection()}
		}

	} else {
		caseNotifications, err := dataAPI.GetNotificationsForCase(patientCase.Id.Int64(), notifyTypes)
		if err != nil {
			return nil, err
		}

		assignments, err := dataAPI.GetActiveMembersOfCareTeamForCase(patientCase.Id.Int64(), true)
		if err != nil {
			return nil, err
		}

		// get current doctor assigned to case
		var careProvider *common.CareProviderAssignment
		for _, assignment := range assignments {
			if assignment.Status == api.STATUS_ACTIVE && assignment.ProviderRole == api.DOCTOR_ROLE {
				careProvider = assignment
				break
			}
		}

		// populate home cards based on the notification types and the number of notifications in the case
		switch l := len(caseNotifications); {

		case l == 1:
			hView, err := caseNotifications[0].Data.(notification).makeHomeCardView(dataAPI, apiDomain)
			if err != nil {
				return nil, err
			}

			switch caseNotifications[0].NotificationType {

			case CNIncompleteVisit:
				views = []common.ClientView{hView, getSendUsMessageSection(), getLearnAboutSpruceSection()}

			case CNVisitSubmitted:
				views = []common.ClientView{getViewCaseCard(patientCase, careProvider, hView), getViewResourceLibrarySection()}

			case CNTreatmentPlan:
				careTeamSection, err := getMeetCareTeamSection(assignments, dataAPI)
				if err != nil {
					return nil, err
				}
				views = []common.ClientView{getViewCaseCard(patientCase, careProvider, hView), careTeamSection}

			case CNMessage:
				views = []common.ClientView{getViewCaseCard(patientCase, careProvider, hView)}
			}

		case l > 1:
			spelledNumber := " "
			switch l {
			case 2:
				spelledNumber = " two "
			case 3:
				spelledNumber = " three "
			case 4:
				spelledNumber = " four "
			case 5:
				spelledNumber = " five "
			case 6:
				spelledNumber = " six "
			case 7:
				spelledNumber = " seven "
			case 8:
				spelledNumber = " eight "
			case 9:
				spelledNumber = " nine "
			case 10:
				spelledNumber = " ten "
			}
			views = []common.ClientView{getViewCaseCard(patientCase, careProvider, &phCaseNotificationMultipleView{
				NotificationCount: int64(l),
				Title:             "You have" + spelledNumber + "new updates.",
				ButtonTitle:       "View Case",
				ActionURL:         app_url.ViewCaseAction(patientCase.Id.Int64()),
			}), getSendCareTeamMessageSection(patientCase.Id.Int64())}

		case l == 0:

			imageURL := app_url.IconCaseLarge.String()
			if careProvider != nil {
				imageURL = app_url.LargeThumbnailURL(apiDomain, api.DOCTOR_ROLE, careProvider.ProviderID)
			}

			buttons := []*phTitleActionURL{
				&phTitleActionURL{
					Title:     "Case Details",
					ActionURL: app_url.ViewCaseAction(patientCase.Id.Int64()),
				},
				&phTitleActionURL{
					Title:     "Messages",
					ActionURL: app_url.ViewCaseMessageThreadAction(patientCase.Id.Int64()),
				},
			}

			activeTreatmentPlanExists, err := dataAPI.DoesActiveTreatmentPlanForCaseExist(patientCase.Id.Int64())
			if err != nil {
				return nil, err
			}

			// only include the treatment plans button if the a treatment plan exists
			if activeTreatmentPlanExists {
				buttons = append(buttons, &phTitleActionURL{
					Title:     "Treatment Plan",
					ActionURL: app_url.ViewTreatmentPlanForCaseAction(patientCase.Id.Int64()),
				})
			}

			views = []common.ClientView{getViewCaseCard(patientCase, careProvider, &phCaseNotificationNoUpdatesView{
				Title:    "No new updates.",
				ImageURL: imageURL,
				Buttons:  buttons,
			}), getSendCareTeamMessageSection(patientCase.Id.Int64())}
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
		Description: "Receive an effective, personalized treatment plan from a dermatologist in less than 24 hours.",
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

func getViewCaseCard(patientCase *common.PatientCase, careProvider *common.CareProviderAssignment, notificationView common.ClientView) common.ClientView {
	switch patientCase.Status {

	case common.PCStatusUnclaimed, common.PCStatusTempClaimed:
		return &phCaseView{
			Title:            "Dermatology Case",
			Subtitle:         "Pending Review",
			ActionURL:        app_url.ViewCaseAction(patientCase.Id.Int64()),
			IconURL:          app_url.IconCaseLarge.String(),
			CaseID:           patientCase.Id.Int64(),
			NotificationView: notificationView,
		}

	case common.PCStatusClaimed, common.PCStatusUnsuitable:
		return &phCaseView{
			Title:            "Dermatology Case",
			Subtitle:         fmt.Sprintf("With Dr. %s %s", careProvider.FirstName, careProvider.LastName),
			ActionURL:        app_url.ViewCaseAction(patientCase.Id.Int64()),
			IconURL:          careProvider.LargeThumbnailURL,
			CaseID:           patientCase.Id.Int64(),
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
		sectionView.Views = append(sectionView.Views, &phCareProviderView{
			CareProvider: assignment,
		})
	}

	return sectionView, nil
}

func getViewResourceLibrarySection() common.ClientView {
	return &phSectionView{
		Views: []common.ClientView{
			&phSmallIconText{
				Title:       "Check out Spruce’s skin care guides",
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
				IconURL:     app_url.IconMessagesLarge,
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
				Title:       "Contact Spruce",
				IconURL:     app_url.IconSupport,
				ActionURL:   app_url.ViewSupportAction(),
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
				Title:       "Meet the doctors",
				IconURL:     app_url.IconSpruceDoctors,
				ActionURL:   app_url.ViewSampleDoctorProfilesAction(),
				RoundedIcon: true,
			},
			&phSmallIconText{
				Title:       "What a Spruce visit includes",
				IconURL:     app_url.IconCaseLarge,
				ActionURL:   app_url.ViewPricingFAQAction(),
				RoundedIcon: true,
			},
			&phSmallIconText{
				Title:       "See a sample treatment plan",
				IconURL:     app_url.IconTreatmentPlanLarge,
				ActionURL:   app_url.ViewSampleTreatmentPlanAction(),
				RoundedIcon: true,
			},
			&phSmallIconText{
				Title:       "Frequently Asked Questions",
				IconURL:     app_url.IconFAQ,
				ActionURL:   app_url.ViewSpruceFAQAction(),
				RoundedIcon: true,
			},
		},
	}
}
