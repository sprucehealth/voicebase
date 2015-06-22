package patient_case

import (
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
				ID:              encoding.NewObjectID(2),
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
	_, err := n.makeHomeCardView(dataAPI, caseData)
	test.OK(t, err)

	// Care provider is not in care team
	n = &messageNotification{
		MessageID: 1,
		DoctorID:  2,
		CaseID:    1,
		Role:      api.RoleCC,
	}
	_, err = n.makeHomeCardView(dataAPI, caseData)
	test.OK(t, err)
}
