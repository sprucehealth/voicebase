package messages

import (
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/appevent"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/errors"
)

// InitListeners subscribes to dispatched events.
func InitListeners(dataAPI api.DataAPI, dispatcher *dispatch.Dispatcher) {
	dispatcher.Subscribe(func(ev *appevent.AppEvent) error {
		// all_case_messages = all messages in a case was read (currently only the doctor app sends this)
		// case_message = the latest message in a case was read (currently onlt the patient app sends this)
		if (ev.Resource == "all_case_messages" || ev.Resource == "case_message") && ev.Action == appevent.ViewedAction {
			var roleID int64
			var opts api.CaseMessagesReadOption
			switch ev.Role {
			case api.RolePatient:
				patientID, err := dataAPI.GetPatientIDFromAccountID(ev.AccountID)
				if err != nil {
					return errors.Trace(err)
				}
				roleID = patientID.Int64()
			case api.RoleCC, api.RoleDoctor:
				var err error
				roleID, err = dataAPI.GetDoctorIDFromAccountID(ev.AccountID)
				if err != nil {
					return errors.Trace(err)
				}
				opts |= api.CMROIncludePrivate
			}
			personID, err := dataAPI.GetPersonIDByRole(ev.Role, roleID)
			if err != nil {
				return errors.Trace(err)
			}
			caseID := ev.ResourceID
			if ev.Resource == "case_message" {
				caseID, err = dataAPI.GetCaseIDFromMessageID(ev.ResourceID)
				if err != nil {
					return errors.Trace(err)
				}
			}
			if err := dataAPI.AllCaseMessagesRead(caseID, personID, opts); err != nil {
				return errors.Trace(err)
			}
		}
		return nil
	})
}
