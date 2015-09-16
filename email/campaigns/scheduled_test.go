package campaigns

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/libs/cfg"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/test"
	"github.com/sprucehealth/backend/test/config"
)

type mockDataAPI_scheduled struct {
	api.DataAPI
	state     abandonedVisitCampaignState
	summaries []*common.VisitSummary

	campaignKeyRequested string
	campaignKeyUpdated   string
}

func (m *mockDataAPI_scheduled) EmailCampaignState(campaignKey string) ([]byte, error) {
	return json.Marshal(m.state)
}
func (m *mockDataAPI_scheduled) VisitSummaries(states []string, startTime, endTime time.Time) ([]*common.VisitSummary, error) {
	return m.summaries, nil
}
func (m *mockDataAPI_scheduled) UpdateEmailCampaignState(campaignKey string, data []byte) error {
	return nil
}
func (m *mockDataAPI_scheduled) CaseMessageForAttachment(itemType string, itemID, senderPersonID, patientCaseID int64) (*common.CaseMessage, error) {
	return &common.CaseMessage{}, nil
}
func (m *mockDataAPI_scheduled) Doctor(int64, bool) (*common.Doctor, error) {
	return &common.Doctor{}, nil
}
func (m *mockDataAPI_scheduled) Patient(common.PatientID, bool) (*common.Patient, error) {
	return &common.Patient{}, nil
}
func (m *mockDataAPI_scheduled) IsCaseMessageRead(messageID, personID int64) (bool, error) {
	return false, nil
}
func (m *mockDataAPI_scheduled) GetPersonIDByRole(role string, roleID int64) (int64, error) {
	return 10, nil
}

func TestAbandonedCartCampaign(t *testing.T) {
	c := abandonedVisitCampaign{}
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.AbandonedVisitThreshold})
	test.OK(t, err)

	m := &mockDataAPI_scheduled{
		summaries: []*common.VisitSummary{
			{
				PatientDOB: encoding.Date{
					Year:  1920,
					Month: 1,
					Day:   1,
				},
			},
		},
	}

	ci, err := c.Run(m, cfgStore.Snapshot())
	test.OK(t, err)
	test.Equals(t, 1, len(ci.Accounts))

	// now lets go ahead and add a visit summary for a patient that is under18
	// to ensure that the patient is not included in the list of emails to send
	m.summaries = append(m.summaries, &common.VisitSummary{
		PatientDOB: encoding.Date{
			Year:  2015,
			Month: 1,
			Day:   1,
		},
	})

	ci, err = c.Run(m, cfgStore.Snapshot())
	test.OK(t, err)

	test.Equals(t, 1, len(ci.Accounts))
}

func TestUnreadMessageWithPendingFollowupCampaign(t *testing.T) {
	c := unreadMessageWithFollowupCampaign{}
	cfgStore, err := cfg.NewLocalStore([]*cfg.ValueDef{config.UnreadMessageWithPendingFollowupThreshold})
	test.OK(t, err)

	m := &mockDataAPI_scheduled{
		summaries: []*common.VisitSummary{
			{DoctorID: ptr.Int64(10)},
		},
	}

	ci, err := c.Run(m, cfgStore.Snapshot())
	test.OK(t, err)
	test.Equals(t, 1, len(ci.Accounts))
}
