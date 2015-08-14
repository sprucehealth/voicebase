package patient_case

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

func TestNotifications(t *testing.T) {
	dataAPI := &mockHomeHandlerDataAPI{
		doctors: map[int64]*common.Doctor{
			2: {
				ID:              encoding.DeprecatedNewObjectID(2),
				LongDisplayName: "Care Coordinator",
				IsCC:            true,
			},
		},
	}
	assignments := []*common.CareProviderAssignment{
		{
			ProviderRole:    api.RoleCC,
			ProviderID:      1,
			LongDisplayName: "Care Coordinator",
		},
	}
	patientCase := &common.PatientCase{}
	caseData := &caseData{
		APIDomain:       "cdndomain",
		CareTeamMembers: assignments,
		Case:            patientCase,
	}

	// Care provider is in care team
	n := &messageNotification{
		MessageID: 1,
		DoctorID:  1,
		CaseID:    1,
		Role:      api.RoleCC,
	}
	_, err := n.makeHomeCardView(dataAPI, "", caseData)
	test.OK(t, err)

	// Care provider is not in care team
	n = &messageNotification{
		MessageID: 1,
		DoctorID:  2,
		CaseID:    1,
		Role:      api.RoleCC,
	}
	_, err = n.makeHomeCardView(dataAPI, "", caseData)
	test.OK(t, err)
}

func TestIncompleteVisitNotification(t *testing.T) {
	dataAPI := &mockHomeHandlerDataAPI{
		patientVisits: []*common.PatientVisit{
			{
				ID:     encoding.DeprecatedNewObjectID(1),
				Status: common.PVStatusOpen,
			},
		},
	}
	caseData := &caseData{
		Case: &common.PatientCase{},
	}

	n := &incompleteVisitNotification{PatientVisitID: 1}

	view, err := n.makeHomeCardView(dataAPI, "", caseData)
	test.OK(t, err)
	test.Assert(t, strings.Contains(fmt.Sprintf("%+v", view), "With the First Available Doctor"), "Expected normal incomplete visit card, got %+v", view)

	dataAPI.patientVisits[0].Status = common.PVStatusPendingParentalConsent
	view, err = n.makeHomeCardView(dataAPI, "", caseData)
	test.OK(t, err)
	test.Assert(t, strings.Contains(fmt.Sprintf("%+v", view), "Waiting for Parental Consent"), "Expected waiting for consent card, got %+v", view)

	dataAPI.patientVisits[0].Status = common.PVStatusReceivedParentalConsent
	view, err = n.makeHomeCardView(dataAPI, "", caseData)
	test.OK(t, err)
	test.Assert(t, strings.Contains(fmt.Sprintf("%+v", view), "Your parent has provided consent"), "Expected received consent card, got %+v", view)
}
