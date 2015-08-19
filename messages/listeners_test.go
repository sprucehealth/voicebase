package messages

import (
	"testing"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/appevent"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/test"
)

type mockDataAPIListeners struct {
	api.DataAPI
	readCaseID   int64
	readPersonID int64
	readOpts     api.CaseMessagesReadOption
}

func (a *mockDataAPIListeners) GetPatientIDFromAccountID(accountID int64) (common.PatientID, error) {
	return common.NewPatientID(uint64(accountID) + 7), nil
}

func (a *mockDataAPIListeners) GetDoctorIDFromAccountID(accountID int64) (int64, error) {
	return accountID + 23, nil
}

func (a *mockDataAPIListeners) GetPersonIDByRole(role string, roleID int64) (int64, error) {
	return roleID + 19, nil
}

func (a *mockDataAPIListeners) AllCaseMessagesRead(caseID, personID int64, opts api.CaseMessagesReadOption) error {
	a.readCaseID = caseID
	a.readPersonID = personID
	a.readOpts = opts
	return nil
}

func (a *mockDataAPIListeners) GetCaseIDFromMessageID(msgID int64) (int64, error) {
	return msgID + 97, nil
}

func TestListeners(t *testing.T) {
	dataAPI := &mockDataAPIListeners{}
	dispatcher := dispatch.New()
	InitListeners(dataAPI, dispatcher)

	cases := []struct {
		event         *appevent.AppEvent
		expPersonID   int64
		expReadCaseID int64
		expOpts       api.CaseMessagesReadOption
	}{
		{
			event: &appevent.AppEvent{
				Action:     appevent.ViewedAction,
				Resource:   "all_case_messages",
				ResourceID: 2,
				AccountID:  1,
				Role:       api.RolePatient,
			},
			expPersonID:   1 + 7 + 19,
			expReadCaseID: 2,
			expOpts:       0,
		},
		{
			event: &appevent.AppEvent{
				Action:     appevent.ViewedAction,
				Resource:   "all_case_messages",
				ResourceID: 2,
				AccountID:  1,
				Role:       api.RoleCC,
			},
			expPersonID:   1 + 23 + 19,
			expReadCaseID: 2,
			expOpts:       api.CMROIncludePrivate,
		},
		{
			event: &appevent.AppEvent{
				Action:     appevent.ViewedAction,
				Resource:   "all_case_messages",
				ResourceID: 2,
				AccountID:  1,
				Role:       api.RoleDoctor,
			},
			expPersonID:   1 + 23 + 19,
			expReadCaseID: 2,
			expOpts:       api.CMROIncludePrivate,
		},
		{
			event: &appevent.AppEvent{
				Action:     appevent.ViewedAction,
				Resource:   "case_message",
				ResourceID: 3,
				AccountID:  1,
				Role:       api.RolePatient,
			},
			expPersonID:   1 + 7 + 19,
			expReadCaseID: 3 + 97,
			expOpts:       0,
		},
		{
			event: &appevent.AppEvent{
				Action:     appevent.ViewedAction,
				Resource:   "case_message",
				ResourceID: 3,
				AccountID:  1,
				Role:       api.RoleCC,
			},
			expPersonID:   1 + 23 + 19,
			expReadCaseID: 3 + 97,
			expOpts:       api.CMROIncludePrivate,
		},
		{
			event: &appevent.AppEvent{
				Action:     appevent.ViewedAction,
				Resource:   "case_message",
				ResourceID: 3,
				AccountID:  1,
				Role:       api.RoleDoctor,
			},
			expPersonID:   1 + 23 + 19,
			expReadCaseID: 3 + 97,
			expOpts:       api.CMROIncludePrivate,
		},
	}

	for _, c := range cases {
		t.Logf("Test case: %+v", c)
		test.OK(t, dispatcher.Publish(c.event))
		test.Equals(t, c.expPersonID, dataAPI.readPersonID)
		test.Equals(t, c.expReadCaseID, dataAPI.readCaseID)
		test.Equals(t, c.expOpts, dataAPI.readOpts)
	}
}
